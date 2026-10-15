"""部分來源條件式接入的普通歷史、既有首戰與來源驗證。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def prefix_assets(receipt, project):
    profiles = read(project/'workplace/hd-prototype/catalog/remaining/partial-story/profiles.json')['records']
    assets = receipt['desktop']['assets']['story_prefix_portraits']
    assert set(assets) == {r['resource'] for r in profiles}
    for profile in profiles:
        record = assets[profile['resource']]
        assert record['validation'] == 'ready'
        assert record['source_sha256'] == profile['known_indices_sha256']
        assert record['hd_sha256'] == profile['asset_sha256']
    return list(assets)


def history(root, project):
    assert not (root/'verifier-source.txt').exists()
    (root/'verifier-source.txt').write_bytes(Path(__file__).read_bytes())
    command = [str(root/'tokugawa-desktop-prototype'), '-desktop', '-desktop-speed', '60', '-steps', '600000000',
               '-mouse-file', '/project/workplace/hd-prototype/catalog/remaining/source-boundary/menu-history/mouse.json',
               '-hd-battle-frame-proof']
    env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
    children = []

    def state():
        try:
            return read(root/'desktop-status.json')
        except (FileNotFoundError, json.JSONDecodeError):
            return {}

    def wait(predicate, seconds):
        end = time.monotonic()+seconds
        while time.monotonic() < end:
            current = state()
            if predicate(current):
                return current
            assert process.poll() is None, f'desktop exited {process.returncode}'
            time.sleep(.02)
        raise TimeoutError(f'history at {state().get("step")}')

    try:
        with (root/'xvfb.log').open('wb') as stdout:
            xvfb = subprocess.Popen(['Xvfb', ':98', '-screen', '0', '1600x1000x24', '-nolisten', 'tcp', '-ac'],
                                    env=env, stdout=stdout, stderr=subprocess.STDOUT)
        children.append(xvfb)
        end = time.monotonic()+10
        while not Path('/tmp/.X11-unix/X98').exists():
            assert xvfb.poll() is None and time.monotonic() < end
            time.sleep(.05)
        receipt_path = root/'desktop-prefix-history.json'
        with receipt_path.open('wb') as stdout, (root/'desktop.stderr.log').open('wb') as stderr:
            process = subprocess.Popen(command, env=env, stdout=stdout, stderr=stderr)
            children.append(process)
            wait(lambda s: s.get('step', 0) >= 1000000, 60)
            window = subprocess.run(['xdotool', 'search', '--name', '德川家康'], env=env, check=True, timeout=15,
                                    stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
            subprocess.run(['xdotool', 'windowfocus', '--sync', window], env=env, check=True, timeout=15)
            subprocess.run(['xdotool', 'mousemove', '--window', window, '700', '500'], env=env, check=True, timeout=15)
            subprocess.run(['xdotool', 'keydown', '--clearmodifiers', 'F6'], env=env, check=True, timeout=15)
            try:
                wait(lambda s: s.get('enabled'), 15)
            finally:
                subprocess.run(['xdotool', 'keyup', 'F6'], env=env, check=True, timeout=15)
            process.wait(timeout=800)
            assert process.returncode == 0
        receipt = read(receipt_path)
        catalog = project/'workplace/hd-prototype/catalog'
        baseline_path = catalog/'remaining/source-boundary/menu-history/rcr/receipt.json'
        baseline = read(baseline_path)
        fields = read(catalog/'remaining/history/source-verification.json')['compared_fields']
        for field in fields:
            assert receipt[field] == baseline[field], field
        assert sha(root/'startup-memory.bin') == 'c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2'
        assert len(receipt['integration_input_sha256']) == 90
        for name, digest in receipt['integration_input_sha256'].items():
            assert sha(project/name) == digest == sha(root/'source-snapshot'/name)
        toggles = [e for e in receipt['desktop']['events'] if e['kind'] == 'toggle']
        assert len(toggles) == 1 and toggles[0]['guest_unchanged'] and toggles[0]['before'] == toggles[0]['after']
        assert receipt['desktop']['hd_observation']['graphics']['completed_returns'] == 700
        loaded = prefix_assets(receipt, project)
        original_loads = receipt['desktop']['hd_observation']['loads']['events']
        assert not any(e.get('name') in loaded for e in original_loads)
        result = {'verified': True, 'mode': 'prefix-history', 'classification': 'existing ordinary history regression; prefix assets loaded, their scenes not encountered',
                  'tool_sha256': sha(root/'verifier-source.txt'), 'receipt_sha256': sha(receipt_path), 'baseline_sha256': sha(baseline_path),
                  'binary_sha256': sha(root/'tokugawa-desktop-prototype'), 'complete_RAM_sha256': sha(root/'startup-memory.bin'),
                  'compared_fields': fields, 'source_snapshot_count': 90, 'prefix_assets_validated': loaded, 'normal_prefix_scenes': []}
        (root/'verification-prefix-history.json').write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
        print(json.dumps({'verified': True, 'mode': 'prefix-history', 'steps': receipt['successful_steps']}))
    finally:
        for child in reversed(children):
            if child.poll() is None:
                child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)


def summary(root, project):
    frozen = read(root/'runtime/frozen-integration-input.json')
    assert len(frozen) == 90
    binary = sha(root/'runtime/tokugawa-desktop-prototype')
    log = root/'runtime/startup-probe/platform-test.log'
    text = log.read_text()
    assert 'FAIL' not in text and sum(l.startswith('ok') for l in text.splitlines()) == 4
    profiles = read(root/'profiles.json')
    review = read(root/'review.json')
    assert profiles['tool_sha256'] == sha(project/'tools/hd-prefix-prepare.py')
    assert review['profiles_sha256'] == sha(root/'profiles.json')
    assert len(profiles['records']) == 4
    for record in profiles['records']:
        source = root/record['known_indices']
        assert sha(source) == record['known_indices_sha256'] and source.read_bytes()[-8:] == b'\xff'*8
        assert sha(project/'assets/hd-b/art'/record['asset']) == record['asset_sha256']
    proofs = {}
    for directory, mode, tool in [('history', 'prefix-history', 'verify-hd-prefix-runtime.py'), ('desktop', 'weather', 'verify-hd-weather.py')]:
        folder = root/directory
        receipt_path = folder/f'desktop-{mode}.json'
        proof_path = folder/f'verification-{mode}.json'
        receipt, proof = read(receipt_path), read(proof_path)
        assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
        assert proof['tool_sha256'] == sha(project/'tools'/tool)
        assert sha(folder/'tokugawa-desktop-prototype') == binary
        assert receipt['integration_input_sha256'] == frozen
        for name, digest in frozen.items():
            assert digest == sha(project/name) == sha(folder/'source-snapshot'/name)
        prefix_assets(receipt, project)
        assert proof['complete_RAM_sha256'] == sha(folder/'startup-memory.bin')
        if mode == 'weather':
            previous = project/'workplace/hd-prototype/catalog/remaining/history/scrolling/regression/ready'
            for picture in proof['pictures']:
                name = picture['actual_png']
                assert Image.open(folder/name).convert('RGBA').tobytes() == Image.open(previous/name).convert('RGBA').tobytes()
        proofs[mode] = {'path': str(proof_path.relative_to(root)), 'sha256': sha(proof_path), 'receipt_sha256': sha(receipt_path),
                        'complete_RAM_sha256': proof['complete_RAM_sha256']}
    result = {'verified': True, 'classification': 'four known-region story profiles registered; source/shared-consumer tests and existing ordinary-path regression',
              'tool_sha256': sha(Path(__file__)), 'binary_sha256': binary, 'source_snapshot_count': 90,
              'profiles_sha256': sha(root/'profiles.json'), 'review_sha256': sha(root/'review.json'),
              'platform_test_log_sha256': sha(log), 'frozen_inputs_sha256': sha(root/'runtime/frozen-integration-input.json'),
              'registered_sources': [r['resource'] for r in profiles['records']], 'unknown_native_pixels_per_source': 8,
              'normal_prefix_sources': [], 'proofs': proofs,
              'limitations': ['four scene-specific natural player paths not sampled; conditional verified consumer only',
                              'original missing high-plane byte remains unknown; synthetic 256-byte tests are not original evidence',
                              'other animation consumers, PCX, whole-game HD and platform packaging remain incomplete']}
    target = root/'verification-summary.json'
    assert not target.exists()
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'verified': True, 'sources': 4, 'sha256': sha(target)}))


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', choices=['history', 'summary'], required=True)
    args = parser.parse_args()
    root, project = Path('/output'), Path('/project')
    assert root.stat().st_uid == 1000
    if args.mode == 'history':
        history(root, project)
    else:
        summary(root, project)


if __name__ == '__main__':
    main()
