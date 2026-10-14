"""核對完整物件來源、共用校準材質與正常首戰，不把離線接縫圖當玩家路徑。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project,root = Path('/project'),Path('/output')
    terrain = project/'workplace/hd-prototype/catalog/terrain'
    path = root/'profiles.json'
    profiles = json.loads(path.read_text())
    assert profiles['tool_sha256'] == sha(root/'profiles-source.txt') == sha(project/'tools/hd-wall-prepare.py')
    draft_path = root/profiles['draft_manifest']
    draft = json.loads(draft_path.read_text())
    assert profiles['draft_manifest_sha256'] == sha(draft_path)
    assert draft['tool_sha256'] == profiles['draft_tool_sha256'] == sha(draft_path.parent/'prepare-source.txt')
    review_path = root/profiles['review']
    review = json.loads(review_path.read_text())
    assert profiles['review_sha256'] == sha(review_path) and review['verified']
    assert profiles['approved_indices'] == list(range(24)) and profiles['empty_indices'] == [24,25]
    assert len(profiles['records']) == profiles['source_slots'] == 26
    base_path = terrain/'objects/expanded/profiles.json'
    base = json.loads(base_path.read_text())
    assert profiles['base_profiles_sha256'] == sha(base_path)
    idx_path = project/'workplace/hd-prototype/catalog/decoded/MAP_CHIP.OZM.idx'
    idx = idx_path.read_bytes()
    assert sha(idx_path) == '0e4e6f63e71a39309e21888724ed0e4da29e8f58cfb49b55095340132f4b010b'
    # 整個 640×400 圖集只有右側 16 欄在已盤點來源格以外。
    padding = b''.join(idx[y*640+624:y*640+640] for y in range(400))
    assert not any(padding), 'unclassified atlas pixels remain'
    for r in profiles['records']:
        i = r['index']; x,y = i%13*48,256+i//13*48
        raw = b''.join(idx[row*640+x:row*640+x+48] for row in range(y,y+48))
        assert hashlib.sha256(raw).hexdigest() == r['source_indices_sha256']
        source_path = project/r['source']
        assert sha(source_path) == r['source_png_sha256']
        original = Image.open(source_path).convert('RGBA')
        assert original.getchannel('A').tobytes() == bytes(255 if v else 0 for v in raw)
        if i in [24,25]:
            assert not r['asset'] and not any(raw)
            continue
        assert list(original.getchannel('A').getbbox()) == r['source_bounds']
        asset_path = project/'assets/hd-b/terrain'/r['asset']
        assert sha(asset_path) == r['asset_sha256']
        assert sha(root/r['prepared']) == r['prepared_sha256']
        if i in profiles['unchanged_indices']:
            for key in ['asset','asset_sha256','crop','source_bounds','prepared_sha256']:
                assert r[key] == base['records'][i][key], (i,key)
        if str(i) in profiles['exact_source_crop_reuse']:
            other = profiles['records'][profiles['exact_source_crop_reuse'][str(i)]]
            assert r['asset'] == other['asset'] and r['crop'] == other['crop']
    folder = root/'desktop'
    receipt_path, proof_path = folder/'desktop-weather.json', folder/'verification-weather.json'
    receipt,proof = json.loads(receipt_path.read_text()),json.loads(proof_path.read_text())
    assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
    assert proof['tool_sha256'] == sha(project/'tools/verify-hd-weather.py') == sha(folder/'verify-hd-weather.py')
    assert receipt['successful_steps'] == 750000000 and len(receipt['mouse_input_events']) == 390
    assert sha(folder/'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name,digest in receipt['integration_input_sha256'].items():
        assert sha(project/name) == digest == sha(folder/'source-snapshot'/name)
    loaded = receipt['desktop']['assets']['terrain_objects']
    assert set(loaded) == {str(i) for i in range(24)}
    for r in profiles['records'][:24]:
        info = loaded[str(r['index'])]
        assert info['validation'] == 'ready' and info['source_sha256'] == r['source_png_sha256'] and info['hd_sha256'] == r['asset_sha256']
    observation = receipt['desktop']['hd_observation']['terrain']
    assert {r['index'] for r in observation['sprites'] if r['group'] == 'objects'} == {0}
    scene = receipt['desktop']['terrain_scene']
    assert scene['object_hd_count'] == scene['tree_hd_count'] == 4 and scene['unit_hd_count'] == 2
    previous = terrain/'objects/expanded/desktop'
    for picture in proof['pictures']:
        actual = folder/picture['actual_png']
        assert picture['window_pixels_equal'] and sha(actual) == picture['sha256']
        assert Image.open(actual).convert('RGBA').tobytes() == Image.open(previous/picture['actual_png']).convert('RGBA').tobytes()
    log_path = root/'runtime/startup-probe/platform-test.log'
    log = log_path.read_text()
    assert 'FAIL' not in log and len([s for s in log.splitlines() if s.startswith('ok')]) == 4
    result = {'verified':True,'classification':'all 24 nonempty object sources integrated; calibrated wall fixtures and shared scene tests; normal first battle object zero',
              'tool_sha256':sha(Path(__file__)),'profiles_sha256':sha(path),'review_sha256':sha(review_path),
              'source_indices_sha256':sha(idx_path),'atlas_padding_is_zero':True,'approved_objects':24,'calibrated_wall_sources':16,
              'empty_indices':[24,25],'normal_object_source_indices':[0],'normal_object_instances':4,
              'receipt_sha256':sha(receipt_path),'GUI_verification_sha256':sha(proof_path),
              'binary_sha256':sha(folder/'tokugawa-desktop-prototype'),'platform_test_log_sha256':sha(log_path),
              'prior_native_and_HD_images_identical':True,
              'limitations':['wall enclosures are offline art fixtures, not normal player-path captures',
                             'other 23 object sources not individually sampled in normal player paths',
                             'other OZM and animation artwork incomplete; no platform packages yet']}
    target = root/'verification-summary.json'
    assert not target.exists()
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'approved_objects':24,'calibrated_walls':16,'sha256':sha(target)}))


if __name__ == '__main__':
    main()
