"""普通歷史簡介的局部 HD、個別缺圖與匹配來源彙整。"""
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


def rect(value):
    return [value['Min']['X'], value['Min']['Y'], value['Max']['X'], value['Max']['Y']]


def crop(data, width, area):
    return b''.join(data[y * width + area[0]:y * width + area[2]] for y in range(area[1], area[3]))


def stages():
    result = []
    for i in range(3):
        result.append({'key': f'duel-{i}', 'source': 'OP2_P.OZM', 'src': [i * 128, 0, (i + 1) * 128, 119], 'dst': [248, 50, 376, 169]})
    for name in ['6.OZM', '11.OZM', '19.OZM', '28.OZM', '43.OZM']:
        result.append({'key': name, 'source': name, 'src': [0, 0, 320, 150], 'dst': [160, 50, 480, 200], 'story': True})
    for i in range(4):
        result.append({'key': f'eye-{i}', 'source': 'OP9.OZM', 'src': [416, 200 + i * 27, 488, 227 + i * 27], 'dst': [216, 98, 288, 125]})
    for i in range(5):
        x, y = (i % 3 * 208, 0) if i < 3 else ((i - 3) * 208, 200)
        result.append({'key': f'portrait-{i}', 'source': 'OP9.OZM', 'src': [x, y, x + 208, y + 200], 'dst': [216, 30, 424, 230]})
    return result


def verify(root, project, missing):
    mode = 'partial-missing' if missing else 'partial'
    history = project / 'workplace/hd-prototype/catalog/remaining/history'
    source_path = history / 'partial/source-verification.json'
    source_proof = json.loads(source_path.read_text())
    assert source_proof['verified']
    graphics = {r['name']: r for r in json.loads((history.parent / 'profiles.json').read_text())['records']}
    stories = {r['resource']: r for r in json.loads((history.parents[1] / 'story-expanded/profiles.json').read_text())['records']}
    verifier_path = root / 'verifier-source.txt'
    assert not verifier_path.exists()
    verifier_path.write_bytes(Path(__file__).read_bytes())
    (root / 'startup-probe').mkdir(exist_ok=True)
    command = [str(root / 'tokugawa-desktop-prototype'), '-desktop', '-desktop-speed', '1', '-steps', '600000000',
               '-mouse-file', '/project/workplace/hd-prototype/catalog/remaining/source-boundary/menu-history/mouse.json',
               '-hd-battle-frame-proof']
    if missing:
        omitted = {graphics[n]['asset'] for n in ['OP2_P.OZM', 'OP9.OZM']}
        for folder, excluded in [('missing-graphics', omitted), ('missing-story', {stories['11.OZM']['asset']})]:
            directory = root / folder
            directory.mkdir()
            archive = project / 'assets/hd-b/art'
            for source in archive.rglob('*.png'):
                relative = source.relative_to(archive)
                if str(relative) in excluded:
                    continue
                target = directory / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                target.symlink_to(source)
        command += ['-hd-graphics-dir', str(root / 'missing-graphics'), '-hd-portrait-dir', str(root / 'missing-story')]
    env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
    children = []

    def run(*args, **kwargs):
        return subprocess.run(args, env=env, check=True, timeout=15, **kwargs)

    def status():
        try:
            return json.loads((root / 'desktop-status.json').read_text())
        except (FileNotFoundError, json.JSONDecodeError):
            return {}

    def wait(predicate, seconds=45):
        end = time.monotonic() + seconds
        while time.monotonic() < end:
            current = status()
            if predicate(current):
                return current
            assert process.poll() is None, f'desktop exited {process.returncode}'
            time.sleep(.01)
        raise TimeoutError(f'partial wait at {status().get("step")}')

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
        path = root / f'gui-{mode}-{label}.png'
        run('ffmpeg', '-v', 'error', '-y', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab',
            '-video_size', f'{geometry["WIDTH"]}x{geometry["HEIGHT"]}', '-i', f':98+{geometry["X"]},{geometry["Y"]}',
            '-frames:v', '1', str(path), stdout=subprocess.DEVNULL)
        assert Image.open(path).convert('RGBA').tobytes() == Image.open(event['frame']['path']).convert('RGBA').tobytes()
        return {'actual_png': path.name, 'sha256': sha(path), 'event': event, 'state': state, 'window_pixels_equal': True}

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
        checks = []
        with receipt_path.open('wb') as stdout, (root / 'desktop.stderr.log').open('wb') as stderr:
            process = subprocess.Popen(command, env=env, stdout=stdout, stderr=stderr)
            children.append(process)
            wait(lambda s: s.get('step', 0) >= 1000000, 60)
            window = run('xdotool', 'search', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
            run('xdotool', 'windowfocus', '--sync', window)
            run('xdotool', 'mousemove', '--window', window, '700', '500')
            key('F6')
            for stage in stages():
                source, area = stage['source'], stage['dst']
                expected_mode = 'fallback_missing_asset' if missing and source in ['OP2_P.OZM', 'OP9.OZM', '11.OZM'] else 'hd'

                def ready(s):
                    if stage.get('story'):
                        return s.get('portrait_resource') == source and s.get('eligible') and s.get('portrait_mode') == expected_mode
                    drawn = any(d['resource'] == source and rect(d['SourceRect']) == stage['src']
                                for d in s.get('graphics_observation', {}).get('verified_draws') or [])
                    return drawn and any(r['resource'] == source and r['mode'] == expected_mode for r in s.get('graphics') or [])

                if status().get('paused'):
                    # Existing F7 controls advance only a few normal updates between short portraits.
                    pulses = 0
                    while not ready(status()):
                        assert pulses < 120, (stage['key'], 'bounded F7 sampling missed frame')
                        key('F7')
                        key('F7')
                        pulses += 1
                else:
                    wait(ready, 1500)
                    key('F7')
                assert ready(status()), (stage['key'], 'pause crossed the selected original frame', status().get('step'))
                enhanced = capture(stage['key'] + '-enabled')
                key('F6')
                original = capture(stage['key'] + '-original')
                key('F6')
                restored = capture(stage['key'] + '-restored')
                assert enhanced['event']['proof'] == original['event']['proof'] == restored['event']['proof']
                assert enhanced['event']['frame']['png_sha256'] == restored['event']['frame']['png_sha256']
                raw = enhanced['event']['frame']['native_indices']
                assert sha(root / raw['path']) == raw['sha256']
                indexed = (root / raw['path']).read_bytes()
                profile = stories[source] if stage.get('story') else graphics[source]
                width = 320 if stage.get('story') else profile['source']['width']
                idx_path = history.parents[1] / 'decoded' / (source + '.idx')
                assert crop(indexed, 640, area) == crop(idx_path.read_bytes(), width, stage['src'])
                before = Image.open(root / original['actual_png']).convert('RGB')
                after = Image.open(root / enhanced['actual_png']).convert('RGB')
                physical = [c * 2 for c in area]
                painting = Image.open(project / 'assets/hd-b/art' / profile['asset']).convert('RGB')
                if stage.get('story'):
                    assert not profile['label_rects']
                    selected = Image.new('RGB', (640, 300))
                    pixels, target = painting.load(), selected.load()
                    for y in range(300):
                        for x in range(640):
                            target[x, y] = pixels[x * painting.width // 640, y * painting.height // 300]
                else:
                    selected = painting.crop(tuple(v * 2 for v in stage['src']))
                want = selected if expected_mode == 'hd' else before.crop(physical)
                assert after.crop(physical).tobytes() == want.tobytes(), (stage['key'], 'HD crop formula')
                state = enhanced['state']
                # Existing scroll and upper frame remain enabled; their rectangles are separate validated layers.
                a, b = after.load(), before.load()
                for y in range(800):
                    for x in range(1280):
                        if physical[0] <= x < physical[2] and physical[1] <= y < physical[3]:
                            continue
                        central = 320 <= x < 960 and 100 <= y < 400
                        frame = state.get('frame_mode') == 'hd' and y < 500 and not central
                        scroll = state.get('scroll_mode') == 'hd' and y >= 500
                        if not frame and not scroll:
                            assert a[x, y] == b[x, y], (stage['key'], 'outside authorized HD layers', x, y)
                changed = sum(a != b for a, b in zip(before.crop(physical).getdata(), after.crop(physical).getdata()))
                assert (changed > 0) == (expected_mode == 'hd')
                checks.append({'key': stage['key'], 'source': source, 'source_rect': stage['src'], 'dest_rect': area,
                               'mode': expected_mode, 'step': enhanced['event']['step'], 'changed_crop_pixels': changed,
                               'native_indices_sha256': raw['sha256'], 'pictures': [enhanced, original, restored]})
                (root / 'completed-stages.json').write_text(json.dumps([c['key'] for c in checks]))
                if stage['key'] not in ['eye-3', 'portrait-0', 'portrait-1', 'portrait-2', 'portrait-3']:
                    key('F7')
            process.wait(timeout=600)
            assert process.returncode == 0
        receipt = json.loads(receipt_path.read_text())
        base_path = history.parent / 'source-boundary/menu-history/rcr/receipt.json'
        baseline = json.loads(base_path.read_text())
        fields = json.loads((history / 'source-verification.json').read_text())['compared_fields']
        for field in fields:
            assert receipt[field] == baseline[field], field
        assert sha(root / 'startup-memory.bin') == 'c854c975a6cc487629b7bbbc281d565e8a0ffa69e7fe510c7861f7603299d2d2'
        toggles = [e for e in receipt['desktop']['events'] if e['kind'] == 'toggle']
        assert len(toggles) == 35 and all(e['guest_unchanged'] and e['before'] == e['after'] for e in toggles)
        assert len(receipt['integration_input_sha256']) == 84
        for name, digest in receipt['integration_input_sha256'].items():
            assert sha(project / name) == digest == sha(root / 'source-snapshot' / name), name
        result = {'verified': True, 'mode': mode, 'classification': 'normal history-menu partial HD with original guest state unchanged',
                  'tool_sha256': sha(verifier_path), 'receipt_sha256': sha(receipt_path), 'source_proof_sha256': sha(source_path),
                  'binary_sha256': sha(root / 'tokugawa-desktop-prototype'), 'checks': checks, 'compared_fields': fields,
                  'complete_RAM_sha256': sha(root / 'startup-memory.bin'), 'limitations': ['scrolling wrappers and other animations not covered']}
        (root / f'verification-{mode}.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
        print(json.dumps({'verified': True, 'mode': mode, 'stages': len(checks)}))
    finally:
        for child in reversed(children):
            if child.poll() is None:
                child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', choices=['partial', 'partial-missing', 'summary'], required=True)
    args = parser.parse_args()
    root, project = Path('/output'), Path('/project')
    assert root.stat().st_uid == 1000
    if args.mode == 'summary':
        subprocess.run(['python3', '/project/tools/verify-hd-opening-partial-captures.py', '--mode', 'summary'], check=True)
    else:
        verify(root, project, args.mode == 'partial-missing')


if __name__ == '__main__':
    main()
