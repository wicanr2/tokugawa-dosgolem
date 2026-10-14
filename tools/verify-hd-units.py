"""核對全部非空單位的來源、透明裁切、共同場景及正常首戰。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000
    project,root=Path('/project'),Path('/output')
    profiles_path,review_path,manifest_path=root/'profiles.json',root/'review.json',root/'draft-profiles.json'
    profiles,review,manifest=[json.loads(p.read_text()) for p in [profiles_path,review_path,manifest_path]]
    assert profiles['tool_sha256']==manifest['tool_sha256']==sha(project/'tools/hd-unit-prepare.py')
    assert profiles['review_sha256']==sha(review_path) and profiles['draft_profiles_sha256']==review['draft_profiles_sha256']==sha(manifest_path)
    assert review['verified'] and review['reviewed_indices']==list(range(10)) and review['empty_indices']==[10,11,12]
    assert profiles['source_slots']==len(profiles['records'])==13 and profiles['nonempty_sources']==10
    idx_path=project/'workplace/hd-prototype/catalog/decoded/MAP_CHIP.OZM.idx'
    idx=idx_path.read_bytes()
    assert sha(idx_path)=='0e4e6f63e71a39309e21888724ed0e4da29e8f58cfb49b55095340132f4b010b'
    empty=[]
    for record in profiles['records']:
        i=record['index'];x=i*48
        raw=b''.join(idx[y*640+x:y*640+x+48] for y in range(352,400))
        assert hashlib.sha256(raw).hexdigest()==record['source_indices_sha256']
        source_path=project/record['source']
        assert sha(source_path)==record['source_png_sha256']
        source=Image.open(source_path).convert('RGBA')
        alpha=source.getchannel('A').tobytes()
        assert alpha==bytes(255 if value else 0 for value in raw)
        if not record['asset']:
            assert not any(raw) and record['source_bounds']==record['crop']==[0,0,0,0]
            empty.append(i);continue
        assert list(source.getchannel('A').getbbox())==record['source_bounds']
        asset=project/'assets/hd-b/terrain'/record['asset']
        assert sha(asset)==record['asset_sha256']
        prepared=root/record['prepared']
        assert sha(prepared)==record['prepared_sha256']
        im=Image.open(prepared).convert('RGBA')
        assert im.size==(96,96) and im.getchannel('A').getextrema()==(0,255)
        if i in [1,3]:
            assert sha(prepared)==sha(root/'attempt-01'/record['prepared'])
    assert empty==[10,11,12]
    attempt=json.loads((root/'attempt-01/draft-profiles.json').read_text())
    assert attempt['tool_sha256']==sha(root/'attempt-01/prepare-source.txt')
    folder=root/'desktop';receipt_path,proof_path=folder/'desktop-weather.json',folder/'verification-weather.json'
    receipt,proof=json.loads(receipt_path.read_text()),json.loads(proof_path.read_text())
    assert proof['verified'] and proof['receipt_sha256']==sha(receipt_path)
    assert proof['tool_sha256']==sha(project/'tools/verify-hd-weather.py')==sha(folder/'verify-hd-weather.py')
    assert receipt['successful_steps']==750000000 and len(receipt['mouse_input_events'])==390
    assert sha(folder/'startup-memory.bin')=='4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name,digest in receipt['integration_input_sha256'].items():
        assert sha(project/name)==digest==sha(folder/'source-snapshot'/name)
    loaded=receipt['desktop']['assets']['terrain_units']
    assert set(loaded)=={str(i) for i in range(10)}
    for record in profiles['records'][:10]:
        info=loaded[str(record['index'])]
        assert info['validation']=='ready' and info['source_sha256']==record['source_png_sha256'] and info['hd_sha256']==record['asset_sha256']
    observation=receipt['desktop']['hd_observation']['terrain']
    actual_units=[s['index'] for s in observation['sprites'] if s['group']=='units']
    assert actual_units==[1,3] and receipt['desktop']['terrain_scene']['unit_hd_count']==2
    previous=project/'workplace/hd-prototype/catalog/terrain/materials/desktop'
    for picture in proof['pictures']:
        path=folder/picture['actual_png']
        assert picture['window_pixels_equal'] and sha(path)==picture['sha256']
        assert Image.open(path).convert('RGBA').tobytes()==Image.open(previous/picture['actual_png']).convert('RGBA').tobytes()
    log_path=root/'runtime/startup-probe/platform-test.log';log=log_path.read_text()
    assert 'FAIL' not in log and len([s for s in log.splitlines() if s.startswith('ok')])==4
    result={'verified':True,'classification':'all 10 nonempty unit sources have reviewed HD and shared scene tests; normal first battle two sources',
            'tool_sha256':sha(Path(__file__)),'profiles_sha256':sha(profiles_path),'review_sha256':sha(review_path),
            'source_indices_sha256':sha(idx_path),'source_slots':13,'approved_nonempty_units':10,'empty_indices':empty,
            'unchanged_normal_unit_indices':[1,3],'normal_unit_source_indices':actual_units,
            'receipt_sha256':sha(receipt_path),'GUI_verification_sha256':sha(proof_path),
            'binary_sha256':sha(folder/'tokugawa-desktop-prototype'),'platform_test_log_sha256':sha(log_path),
            'prior_normal_native_and_HD_images_identical':True,
            'limitations':['other eight unit sources not individually sampled in normal player paths',
                           'original unit-type mapping not inferred from art or source indices',
                           'remaining object atlas and other artwork incomplete; no platform packages yet']}
    target=root/'verification-summary.json';assert not target.exists()
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'approved_units':10,'empty_indices':empty,'sha256':sha(target)}))


if __name__=='__main__':
    main()
