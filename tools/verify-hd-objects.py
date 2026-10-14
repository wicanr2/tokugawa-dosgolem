"""核對物件原生索引、透明裁切、共同場景與正常首戰；城牆草稿不列入接入。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, root = Path('/project'), Path('/output')
    profiles_path, review_path, manifest_path = root/'profiles.json', root/'review.json', root/'draft-profiles.json'
    profiles, review, manifest = [json.loads(p.read_text()) for p in [profiles_path,review_path,manifest_path]]
    assert profiles['tool_sha256'] == manifest['tool_sha256'] == sha(project/'tools/hd-object-prepare.py')
    assert profiles['review_sha256'] == sha(review_path)
    assert profiles['draft_profiles_sha256'] == review['draft_profiles_sha256'] == sha(manifest_path)
    approved = profiles['approved_indices']
    assert review['verified'] and review['approved_indices'] == approved and review['reviewed_indices'] == list(range(24))
    assert review['empty_indices'] == [24,25] and profiles['source_slots'] == len(profiles['records']) == 26
    assert profiles['nonempty_sources'] == 24
    idx_path = project/'workplace/hd-prototype/catalog/decoded/MAP_CHIP.OZM.idx'
    idx = idx_path.read_bytes()
    assert sha(idx_path) == '0e4e6f63e71a39309e21888724ed0e4da29e8f58cfb49b55095340132f4b010b'
    empty = []
    for r in profiles['records']:
        i = r['index']
        x, y = i%13*48, 256+i//13*48
        raw = b''.join(idx[row*640+x:row*640+x+48] for row in range(y,y+48))
        assert hashlib.sha256(raw).hexdigest() == r['source_indices_sha256']
        source_path = project/r['source']
        assert sha(source_path) == r['source_png_sha256']
        source = Image.open(source_path).convert('RGBA')
        assert source.getchannel('A').tobytes() == bytes(255 if value else 0 for value in raw)
        if not r['asset']:
            assert not any(raw) and r['source_bounds'] == r['crop'] == [0,0,0,0]
            empty.append(i)
            continue
        assert list(source.getchannel('A').getbbox()) == r['source_bounds']
        asset = project/'assets/hd-b/terrain'/r['asset']
        assert sha(asset) == r['asset_sha256']
        prepared = root/r['prepared']
        assert sha(prepared) == r['prepared_sha256']
        im = Image.open(prepared).convert('RGBA')
        low, high = im.getchannel('A').getextrema()
        assert im.size == (96,96) and low == 0 and 0 < high <= 255
        if i == 0:
            assert r['asset_sha256'] == 'f6885ec6d1d557973fdb304fca537e095eec4c65e37439340ef2c39da2d94cd2'
            assert r['source_bounds'] == [0,0,47,47] and r['crop'] == [7,0,1247,1245]
    assert empty == [24,25]
    folder = root/'desktop'
    receipt_path, proof_path = folder/'desktop-weather.json', folder/'verification-weather.json'
    receipt, proof = json.loads(receipt_path.read_text()), json.loads(proof_path.read_text())
    assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
    assert proof['tool_sha256'] == sha(project/'tools/verify-hd-weather.py') == sha(folder/'verify-hd-weather.py')
    assert receipt['successful_steps'] == 750000000 and len(receipt['mouse_input_events']) == 390
    assert sha(folder/'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name, digest in receipt['integration_input_sha256'].items():
        assert sha(project/name) == digest == sha(folder/'source-snapshot'/name)
    loaded = receipt['desktop']['assets']['terrain_objects']
    assert set(loaded) == {str(i) for i in approved}
    for r in profiles['records']:
        if r['index'] not in approved:
            continue
        info = loaded[str(r['index'])]
        assert info['validation'] == 'ready' and info['source_sha256'] == r['source_png_sha256'] and info['hd_sha256'] == r['asset_sha256']
    observation = receipt['desktop']['hd_observation']['terrain']
    actual = [s['index'] for s in observation['sprites'] if s['group'] == 'objects']
    assert set(actual) == {0}
    scene = receipt['desktop']['terrain_scene']
    assert scene['object_hd_count'] == scene['tree_hd_count'] == 4 and scene['unit_hd_count'] == 2
    previous = project/'workplace/hd-prototype/catalog/terrain/units/desktop'
    for picture in proof['pictures']:
        path = folder/picture['actual_png']
        assert picture['window_pixels_equal'] and sha(path) == picture['sha256']
        assert Image.open(path).convert('RGBA').tobytes() == Image.open(previous/picture['actual_png']).convert('RGBA').tobytes()
    log_path = root/'runtime/startup-probe/platform-test.log'
    log = log_path.read_text()
    assert 'FAIL' not in log and len([s for s in log.splitlines() if s.startswith('ok')]) == 4
    result = {'verified':True, 'classification':'approved independent object sources have reviewed HD and shared scene tests; normal first battle forest source only',
              'tool_sha256':sha(Path(__file__)), 'profiles_sha256':sha(profiles_path), 'review_sha256':sha(review_path),
              'source_indices_sha256':sha(idx_path), 'source_slots':26, 'approved_indices':approved, 'empty_indices':empty,
              'normal_object_source_indices':[0], 'normal_object_instances':4,
              'receipt_sha256':sha(receipt_path), 'GUI_verification_sha256':sha(proof_path),
              'binary_sha256':sha(folder/'tokugawa-desktop-prototype'), 'platform_test_log_sha256':sha(log_path),
              'prior_normal_native_and_HD_images_identical':True,
              'limitations':['new independent objects not individually sampled in normal player paths',
                             'wall sprite geometry and connecting roof heights remain draft; not integrated',
                             'other artwork incomplete; no platform packages yet']}
    target = root/'verification-summary.json'
    assert not target.exists()
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'approved_objects':len(approved),'empty_indices':empty,'sha256':sha(target)}))


if __name__ == '__main__':
    main()
