"""核對全地面圖集素材、共同呈現測試及正常首戰的來源閉環。"""
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
    terrain = project/'workplace/hd-prototype/catalog/terrain'
    profiles_path, manifest_path, review_path = root/'profiles.json', root/'draft-manifest-v2.json', root/'review-v2.json'
    profiles, manifest, review = [json.loads(p.read_text()) for p in [profiles_path,manifest_path,review_path]]
    assert profiles['tool_sha256'] == manifest['tool_sha256'] == sha(project/'tools/hd-terrain-materials.py')
    assert profiles['draft_manifest_sha256'] == review['draft_manifest_sha256'] == sha(manifest_path)
    assert profiles['review_sha256'] == sha(review_path) and review['verified']
    assert review['reviewed_indices'] == list(range(104))
    baseline_path = terrain/'prepared/profiles.json'
    baseline = json.loads(baseline_path.read_text())
    assert profiles['baseline_sha256'] == manifest['baseline_sha256'] == sha(baseline_path)
    assert profiles['source_count'] == profiles['reviewed_HD_count'] == len(profiles['records']) == 104
    assert manifest['new_unique_images'] == 93 and manifest['new_source_cases'] == 96
    original_indices = (project/'workplace/hd-prototype/catalog/decoded/MAP_CHIP.OZM.idx').read_bytes()
    source_hash = hashlib.sha256(original_indices).hexdigest()
    assert source_hash == manifest['source_indices_sha256']
    inherited, reused, assets = [], [], set()
    for record, old in zip(profiles['records'],baseline['records']):
        index = record['index']
        assert index == old['index'] and record['source_png_sha256'] == old['source_png_sha256']
        assert sha(terrain/record['source']) == record['source_png_sha256']
        x,y = index%13*48,index//13*32
        raw = b''.join(original_indices[row*640+x:row*640+x+48] for row in range(y,y+32))
        assert hashlib.sha256(raw).hexdigest() == record['indices_sha256']
        path = project/'assets/hd-b/terrain'/record['asset']
        assert sha(path) == record['asset_sha256']
        im = Image.open(path)
        assert im.convert('RGBA').getextrema()[3] == (255,255)
        if record['inherited']:
            inherited.append(index)
            assert record['asset'] == old['asset'] and record['asset_sha256'] == old['asset_sha256']
        else:
            assert im.size == (384,256)
        if record['asset_source_index'] != index:
            owner = profiles['records'][record['asset_source_index']]
            assert owner['indices_sha256'] == record['indices_sha256'] and owner['asset'] == record['asset']
            reused.append([index,record['asset_source_index']])
        assets.add(record['asset'])
    assert inherited == [0,2,6,13,15,22,62,67] and reused == [[33,10],[34,11],[35,12]]
    assert len(assets) == 101
    v1 = json.loads((root/'draft-manifest.json').read_text())
    assert v1['tool_sha256'] == sha(root/'prepare-v1-source.txt')
    folder = root/'desktop'
    receipt_path, proof_path = folder/'desktop-weather.json', folder/'verification-weather.json'
    receipt, proof = json.loads(receipt_path.read_text()), json.loads(proof_path.read_text())
    assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
    assert proof['tool_sha256'] == sha(project/'tools/verify-hd-weather.py') == sha(folder/'verify-hd-weather.py')
    assert receipt['successful_steps'] == 750000000 and len(receipt['mouse_input_events']) == 390
    assert sha(folder/'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name,digest in receipt['integration_input_sha256'].items():
        assert sha(project/name) == digest == sha(folder/'source-snapshot'/name)
    loaded = receipt['desktop']['assets']['terrain']
    assert len(loaded) == 104
    for record in profiles['records']:
        info = loaded[str(record['index'])]
        assert info['validation'] == 'ready'
        assert info['source_sha256'] == record['source_png_sha256'] and info['hd_sha256'] == record['asset_sha256']
    prior = project/'workplace/hd-prototype/catalog/graphics/desktop'
    for picture in proof['pictures']:
        assert picture['window_pixels_equal'] and sha(folder/picture['actual_png']) == picture['sha256']
        old_path = prior/picture['actual_png']
        assert Image.open(folder/picture['actual_png']).convert('RGBA').tobytes() == Image.open(old_path).convert('RGBA').tobytes()
    assert receipt['desktop']['terrain_scene']['floor_hd_count'] == 67
    log_path = root/'runtime/startup-probe/platform-test.log'
    log = log_path.read_text()
    assert 'FAIL' not in log and len([line for line in log.splitlines() if line.startswith('ok')]) == 4
    result = {'verified':True,'classification':'all 104 floor sources have reviewed HD and common consumer tests; normal first battle still six visible source types',
              'tool_sha256':sha(Path(__file__)),'profiles_sha256':sha(profiles_path),'manifest_sha256':sha(manifest_path),
              'review_sha256':sha(review_path),'source_indices_sha256':source_hash,'reviewed_source_count':104,
              'new_source_cases':96,'new_unique_images':93,'unique_HD_images':101,'unchanged_original_HD_indices':inherited,
              'exact_source_reuse':reused,'receipt_sha256':sha(receipt_path),'GUI_verification_sha256':sha(proof_path),
              'binary_sha256':sha(folder/'tokugawa-desktop-prototype'),'platform_test_log_sha256':sha(log_path),
              'normal_source_types_verified':6,'normal_visible_floor_regions':67,
              'prior_normal_native_and_HD_images_identical':True,
              'limitations':['other 98 floor source types not individually sampled in normal player paths',
                             'art material masks are documented HD geometry approximation, not original terrain-rule semantics',
                             'remaining object/unit atlas and other game artwork incomplete; no platform package yet']}
    target = root/'verification-summary.json'
    assert not target.exists(), 'preserve previous verification'
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'reviewed_sources':104,'new_unique_images':93,'sha256':sha(target)}))


if __name__ == '__main__':
    main()
