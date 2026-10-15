"""同一建置的普通歷史簡介完整狀態驗證；短畫格的 X11 擷取另核對。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--missing', action='store_true')
    args = parser.parse_args()
    root, project = Path('/output'), Path('/project')
    history = project / 'workplace/hd-prototype/catalog/remaining/history'
    partial = history / 'partial'
    assert root.stat().st_uid == 1000
    (root / 'startup-probe').mkdir(exist_ok=True)
    (root / 'verifier-source.txt').write_bytes(Path(__file__).read_bytes())
    mode = 'full-missing' if args.missing else 'full'
    command = [str(root / 'tokugawa-desktop-prototype'), '-desktop', '-desktop-speed', '60', '-steps', '600000000',
               '-mouse-file', '/project/workplace/hd-prototype/catalog/remaining/source-boundary/menu-history/mouse.json',
               '-hd-battle-frame-proof']
    if args.missing:
        command += ['-hd-graphics-dir', str(partial / 'desktop-missing/missing-graphics'),
                    '-hd-portrait-dir', str(partial / 'desktop-missing/missing-story')]
    env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
    children = []

    def state():
        try:
            return json.loads((root / 'desktop-status.json').read_text())
        except (FileNotFoundError, json.JSONDecodeError):
            return {}

    def wait(predicate, seconds):
        end = time.monotonic() + seconds
        while time.monotonic() < end:
            current = state()
            if predicate(current):
                return current
            assert process.poll() is None, f'desktop exited early {process.returncode}'
            time.sleep(.03)
        raise TimeoutError(f'full history at {state().get("step")}')

    try:
        with (root / 'xvfb.log').open('wb') as stdout:
            xvfb = subprocess.Popen(['Xvfb', ':98', '-screen', '0', '1600x1000x24', '-nolisten', 'tcp', '-ac'],
                                    env=env, stdout=stdout, stderr=subprocess.STDOUT)
        children.append(xvfb)
        end = time.monotonic() + 10
        while not Path('/tmp/.X11-unix/X98').exists():
            assert xvfb.poll() is None and time.monotonic() < end
            time.sleep(.05)
        receipt_path = root / f'desktop-{mode}.json'
        with receipt_path.open('wb') as stdout, (root / 'desktop.stderr.log').open('wb') as stderr:
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
            process.wait(timeout=700)
            assert process.returncode == 0
        receipt = json.loads(receipt_path.read_text())
        baseline_path = history.parent / 'source-boundary/menu-history/rcr/receipt.json'
        baseline = json.loads(baseline_path.read_text())
        fields = json.loads((history / 'source-verification.json').read_text())['compared_fields']
        for field in fields:
            assert receipt[field] == baseline[field], field
        assert sha(root / 'startup-memory.bin') == 'c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2'
        assert len(receipt['integration_input_sha256']) == 84
        for name, digest in receipt['integration_input_sha256'].items():
            assert sha(project / name) == digest == sha(root / 'source-snapshot' / name)
        toggles = [e for e in receipt['desktop']['events'] if e['kind'] == 'toggle']
        assert len(toggles) == 1 and toggles[0]['guest_unchanged'] and toggles[0]['before'] == toggles[0]['after']
        loaded = receipt['desktop']['assets']['graphics']
        assert len(loaded) == 29
        for name, info in loaded.items():
            expected = 'fallback_missing_asset' if args.missing and name in ['OP2_P.OZM', 'OP9.OZM'] else 'ready'
            assert info['validation'] == expected, (name, info)
        story = receipt['desktop']['assets']['story_portraits']['11.OZM']
        assert bool(story.get('hd_sha256')) != args.missing
        result = {'verified': True, 'mode': mode, 'classification': 'complete ordinary history with normal/missing HD assets; paired GUI samples separately verified',
                  'tool_sha256': sha(root / 'verifier-source.txt'), 'receipt_sha256': sha(receipt_path), 'baseline_sha256': sha(baseline_path),
                  'binary_sha256': sha(root / 'tokugawa-desktop-prototype'), 'complete_RAM_sha256': sha(root / 'startup-memory.bin'),
                  'compared_fields': fields, 'source_snapshot_count': 84}
        (root / f'verification-{mode}.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
        print(json.dumps({'verified': True, 'mode': mode, 'successful_steps': receipt['successful_steps']}))
    finally:
        for child in reversed(children):
            if child.poll() is None:
                child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)


if __name__ == '__main__':
    main()
