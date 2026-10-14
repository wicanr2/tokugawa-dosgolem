"""核對天氣 HD 及缺圖兩次正常路徑的完整來源與共同畫面。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageChops


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, root = Path('/project'), Path('/output')
    source_path, profiles_path = root/'source-verification.json', root/'weather-profiles.json'
    source = json.loads(source_path.read_text())
    assert source['verified'] and source['source_counts']['WEATHER1.OZM'] == 1
    profiles = {r['source']:r for r in json.loads(profiles_path.read_text())['records']}
    assert set(profiles) == {'WEATHER1.OZM','WEATHER2.OZM'}
    for profile in profiles.values():
        assert sha(project/'assets/hd-b/art'/profile['asset']) == profile['asset_sha256']
        assert sha(project/'workplace/hd-prototype/catalog/decoded'/f"{profile['source']}.idx") == profile['source_indices_sha256']
    receipts = []
    enhanced_images = []
    original_images = []
    for folder, mode in [('desktop','weather'),('desktop-missing','weather-missing')]:
        path = root/folder
        receipt_path, proof_path = path/f'desktop-{mode}.json', path/f'verification-{mode}.json'
        receipt, proof = json.loads(receipt_path.read_text()), json.loads(proof_path.read_text())
        assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
        assert proof['tool_sha256'] == sha(path/'verify-hd-weather.py') == sha(project/'tools/verify-hd-weather.py')
        assert proof['weather_reference_pixels_verified'] == 42624
        assert receipt['successful_steps'] == 750000000 and len(receipt['mouse_input_events']) == 390
        assert receipt['fd2_startup_calls'] == 0 and proof['complete_RAM_sha256'] == sha(path/'startup-memory.bin')
        for name, digest in receipt['integration_input_sha256'].items():
            assert sha(project/name) == digest == sha(path/'source-snapshot'/name), name
        for picture in proof['pictures']:
            assert picture['window_pixels_equal'] and sha(path/picture['actual_png']) == picture['sha256']
        observation = receipt['desktop']['hd_observation']['graphics']
        assert observation['completed_returns'] == 1 and len(observation['verified_draws']) == 1
        observed = observation['verified_draws'][0]
        assert observed['resource'] == 'WEATHER1.OZM'
        assert observed['verified'] and observed['ExpectedSHA'] == profiles['WEATHER1.OZM']['source_indices_sha256']
        assets = receipt['desktop']['assets']['graphics']
        for name, profile in profiles.items():
            assert assets[name]['source_indices_sha256'] == profile['source_indices_sha256']
            if mode == 'weather':
                assert assets[name]['validation'] == 'ready' and assets[name]['hd_sha256'] == profile['asset_sha256']
            else:
                assert assets[name]['validation'] == 'fallback_missing_asset' and 'hd_sha256' not in assets[name]
        assert receipt['desktop']['terrain_scene']['floor_hd_count'] == 67
        assert receipt['desktop']['battle_frame']['mode'] == 'hd'
        enhanced_images.append(Image.open(path/proof['pictures'][0]['actual_png']).convert('RGB'))
        original_images.append(Image.open(path/proof['pictures'][1]['actual_png']).convert('RGB'))
        receipts.append({'mode':mode,'receipt_sha256':sha(receipt_path),'GUI_verification_sha256':sha(proof_path),
                         'binary_sha256':sha(path/'tokugawa-desktop-prototype'),
                         'weather_changed_pixels':proof['weather_changed_pixels'],
                         'weather_reference_pixels_verified':proof['weather_reference_pixels_verified']})
    assert len({r['binary_sha256'] for r in receipts}) == 1
    assert original_images[0].tobytes() == original_images[1].tobytes()
    box = ImageChops.difference(*enhanced_images).getbbox()
    assert box and box[0]>=928 and box[1]>=582 and box[2]<=1216 and box[3]<=730
    for region in [(0,0,1280,582),(0,730,1280,800),(0,582,928,730),(1216,582,1280,730)]:
        assert enhanced_images[0].crop(region).tobytes() == enhanced_images[1].crop(region).tobytes()
    result = {'verified':True,'classification':'normal first-battle WEATHER1 HD and missing-item fallback',
              'tool_sha256':sha(Path(__file__)),'source_verification_sha256':sha(source_path),
              'profiles_sha256':sha(profiles_path),'records':receipts,
              'only_weather_differs_between_HD_and_missing':True,'difference_bounds':list(box),
              'limitations':['WEATHER2 exact source and consumer integrated; normal overcast path not sampled',
                             'Linux container GUI only; whole-game art and other platforms remain incomplete']}
    target = root/'verification-summary.json'
    assert not target.exists(), 'preserve previous verification'
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'records':receipts,'sha256':sha(target)},ensure_ascii=False))


if __name__ == '__main__':
    main()
