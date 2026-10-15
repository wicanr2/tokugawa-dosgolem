"""捲動開場配對 X11 與普通歷史完整狀態驗收。"""
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


def area(rect):
    return [rect['Min']['X'], rect['Min']['Y'], rect['Max']['X'], rect['Max']['Y']]


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--missing', action='store_true')
    args = parser.parse_args()
    root, project = Path('/output'), Path('/project')
    history = project/'workplace/hd-prototype/catalog/remaining/history'
    catalog = history.parents[1]
    assert root.stat().st_uid == root.stat().st_gid == 1000
    assert not (root/'verifier-source.txt').exists()
    (root/'verifier-source.txt').write_bytes(Path(__file__).read_bytes())
    (root/'startup-probe').mkdir(exist_ok=True)
    static = json.loads((project/'assets/hd-b/art/static/profiles.json').read_text())
    profiles = {r['name']: r for r in static['records']}
    mode = 'shift-missing' if args.missing else 'shift'
    command = [str(root/'tokugawa-desktop-prototype'), '-desktop', '-desktop-speed', '10', '-steps', '600000000',
               '-mouse-file', '/project/workplace/hd-prototype/catalog/remaining/source-boundary/menu-history/mouse.json',
               '-hd-battle-frame-proof']
    if args.missing:
        directory = root/'missing-graphics'
        directory.mkdir()
        excluded = {'static/OP2-hd-calibrated-v1.png', 'static/OP8-hd-calibrated-v1.png'}
        archive = project/'assets/hd-b/art'
        for source in archive.rglob('*.png'):
            relative = source.relative_to(archive)
            if str(relative) in excluded:
                continue
            target = directory/relative
            target.parent.mkdir(parents=True, exist_ok=True)
            target.symlink_to(source)
        command += ['-hd-graphics-dir', str(directory)]
    env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
    children = []

    def run(*command, **kwargs):
        return subprocess.run(command, env=env, check=True, timeout=15, **kwargs)

    def status():
        try:
            return json.loads((root/'desktop-status.json').read_text())
        except (FileNotFoundError, json.JSONDecodeError):
            return {}

    def wait(predicate, seconds):
        end = time.monotonic()+seconds
        while time.monotonic() < end:
            current = status()
            if predicate(current):
                return current
            assert process.poll() is None, f'desktop exited {process.returncode}'
            time.sleep(.01)
        raise TimeoutError(f'wait at {status().get("step")}')

    def key(name):
        previous = status()
        run('xdotool', 'keydown', '--clearmodifiers', name)
        try:
            if name == 'F6':
                wait(lambda s: s.get('enabled') != previous.get('enabled'), 15)
            elif name == 'F7':
                wait(lambda s: s.get('paused') != previous.get('paused'), 15)
            else:
                wait(lambda s: len(s.get('events') or []) > len(previous.get('events') or []), 15)
        finally:
            run('xdotool', 'keyup', name)
        time.sleep(.03)

    def capture(label):
        key('F9')
        state = status()
        event = state['events'][-1]
        assert event['kind'] == 'capture' and event['paused']
        geometry = dict(line.split('=', 1) for line in run('xdotool', 'getwindowgeometry', '--shell', window,
                                                        stdout=subprocess.PIPE).stdout.decode().splitlines())
        picture = root/f'gui-{mode}-{label}.png'
        run('ffmpeg', '-v', 'error', '-y', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab',
            '-video_size', f'{geometry["WIDTH"]}x{geometry["HEIGHT"]}', '-i', f':98+{geometry["X"]},{geometry["Y"]}',
            '-frames:v', '1', str(picture), stdout=subprocess.DEVNULL)
        presented = Path(event['frame']['path'])
        assert sha(presented) == event['frame']['png_sha256']
        assert Image.open(picture).convert('RGBA').tobytes() == Image.open(presented).convert('RGBA').tobytes()
        return {'actual_png': picture.name, 'sha256': sha(picture), 'event': event, 'state': state, 'window_pixels_equal': True}

    checks = []
    try:
        with (root/'xvfb.log').open('wb') as stdout:
            xvfb = subprocess.Popen(['Xvfb', ':98', '-screen', '0', '1600x1000x24', '-nolisten', 'tcp', '-ac'],
                                    env=env, stdout=stdout, stderr=subprocess.STDOUT)
        children.append(xvfb)
        end = time.monotonic()+10
        while not Path('/tmp/.X11-unix/X98').exists():
            assert xvfb.poll() is None and time.monotonic() < end
            time.sleep(.05)
        receipt_path = root/f'desktop-{mode}.json'
        with receipt_path.open('wb') as stdout, (root/'desktop.stderr.log').open('wb') as stderr:
            process = subprocess.Popen(command, env=env, stdout=stdout, stderr=stderr)
            children.append(process)
            wait(lambda s: s.get('step', 0) >= 1000000, 60)
            window = run('xdotool', 'search', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
            run('xdotool', 'windowfocus', '--sync', window)
            run('xdotool', 'mousemove', '--window', window, '700', '500')
            key('F6')
            for source, low, high in [('OP2.OZM', 5, 8), ('OP2.OZM', 127, 140),
                                      ('OP8.OZM', 1, 5), ('OP8.OZM', 127, 140), ('OP8.OZM', 255, 270)]:
                expected_mode = 'fallback_missing_asset' if args.missing else 'hd'

                def selected(s):
                    records = s.get('graphics_observation', {}).get('verified_draws') or []
                    return next((d for d in records if d['resource'] == source and d['kind'] == 'opening_shift'
                                 and low <= d.get('source_shift', 0) <= high), None)

                def ready(s):
                    return selected(s) and any(r['resource'] == source and r['mode'] == expected_mode for r in s.get('graphics') or [])

                wait(ready, 1000)
                key('F7')
                state = status()
                assert ready(state), (source, low, 'pause crossed selected draw', state.get('step'))
                record = selected(state)
                count = record.get('source_shift', 0)
                label = f'{source}-N{count}'
                enhanced = capture(label+'-enabled')
                key('F6')
                original = capture(label+'-original')
                key('F6')
                restored = capture(label+'-restored')
                assert enhanced['event']['proof'] == original['event']['proof'] == restored['event']['proof']
                assert enhanced['event']['frame']['png_sha256'] == restored['event']['frame']['png_sha256']
                raw = enhanced['event']['frame']['native_indices']
                indices_path = root/raw['path']
                assert sha(indices_path) == raw['sha256']
                indices = indices_path.read_bytes()
                source_path = catalog/'decoded'/(source+'.idx')
                source_indices = source_path.read_bytes()
                assert len(source_indices) == 128000
                dst = area(record['DestRect'])
                sx, width, rows = (240, 240, 150) if source == 'OP2.OZM' else (440, 208, 200)
                painting = Image.open(project/'assets/hd-b/art'/profiles[source]['asset']).convert('RGB')
                assert painting.size == (1280, 400)
                before = Image.open(root/original['actual_png']).convert('RGB')
                after = Image.open(root/enhanced['actual_png']).convert('RGB')
                want = before.copy()
                paint, target = painting.load(), want.load()
                unknown = []
                known_indices = bytearray()
                for y in range(rows):
                    for x in range(width):
                        position = y*640+sx+x
                        position = position if position < 16 else max(15, position-count)
                        known = position < len(source_indices)
                        known_indices.append(source_indices[position] if known else 255)
                        if not known:
                            unknown.append([dst[0]+x, dst[1]+y])
                            continue
                        assert indices[(dst[1]+y)*640+dst[0]+x] == source_indices[position]
                        if not args.missing:
                            for dy in range(2):
                                for dx in range(2):
                                    target[(dst[0]+x)*2+dx, (dst[1]+y)*2+dy] = paint[position%640*2+dx, position//640*2+dy]
                assert hashlib.sha256(known_indices).hexdigest() == record['known_source_sha256']
                assert after.tobytes() == want.tobytes(), (source, count, 'whole-window HD mapping and outside protection')
                changed = sum(a != b for a, b in zip(after.getdata(), before.getdata()))
                assert (changed > 0) != args.missing
                if source == 'OP8.OZM' and low == 1:
                    assert unknown and len(unknown) == 8-count
                checks.append({'source': source, 'shift_count': count, 'mode': expected_mode, 'step': enhanced['event']['step'],
                               'dest_rect': dst, 'unknown_native_pixels': unknown, 'changed_pixels': changed,
                               'known_source_sha256': record['known_source_sha256'], 'source_indices_sha256': sha(source_path),
                               'pictures': [enhanced, original, restored]})
                (root/'completed-checks.json').write_text(json.dumps(checks, ensure_ascii=False, indent=2)+'\n')
                key('F7')
            process.wait(timeout=800)
            assert process.returncode == 0
        receipt = json.loads(receipt_path.read_text())
        baseline_path = history.parent/'source-boundary/menu-history/rcr/receipt.json'
        baseline = json.loads(baseline_path.read_text())
        fields = json.loads((history/'source-verification.json').read_text())['compared_fields']
        for field in fields:
            assert receipt[field] == baseline[field], field
        assert sha(root/'startup-memory.bin') == 'c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2'
        assert len(receipt['integration_input_sha256']) == 86
        for name, digest in receipt['integration_input_sha256'].items():
            assert sha(project/name) == digest == sha(root/'source-snapshot'/name)
        observed = receipt['desktop']['hd_observation']['graphics']
        assert observed['completed_returns'] == 700, observed['completed_returns']
        toggles = [e for e in receipt['desktop']['events'] if e['kind'] == 'toggle']
        assert len(toggles) == 11 and all(e['guest_unchanged'] and e['before'] == e['after'] for e in toggles)
        for name, info in receipt['desktop']['assets']['graphics'].items():
            expected = 'fallback_missing_asset' if args.missing and name in ['OP2.OZM', 'OP8.OZM'] else 'ready'
            assert info['validation'] == expected, (name, info)
        result = {'verified': True, 'mode': mode, 'classification': 'sampled scrolling HD and complete ordinary history state',
                  'tool_sha256': sha(root/'verifier-source.txt'), 'receipt_sha256': sha(receipt_path), 'baseline_sha256': sha(baseline_path),
                  'binary_sha256': sha(root/'tokugawa-desktop-prototype'), 'complete_RAM_sha256': sha(root/'startup-memory.bin'),
                  'compared_fields': fields, 'source_snapshot_count': 86, 'graphics_completed_returns': observed['completed_returns'],
                  'source_proof_sha256': sha(history/'scrolling/source-verification.json'), 'checks': checks,
                  'limitations': ['five normal-path X11 samples per mode, other shifts covered by original fixtures and shared tests',
                                  'whole-game HD and packaging incomplete']}
        target = root/f'verification-{mode}.json'
        assert not target.exists()
        target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
        print(json.dumps({'verified': True, 'mode': mode, 'samples': len(checks), 'sha256': sha(target)}))
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
