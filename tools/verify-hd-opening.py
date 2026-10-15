"""真實 X11 的普通歷史簡介、完整開場圖及 OP-FULL 個別缺圖驗證。"""
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


def palette_effect(active, contract):
    base = contract['base_palette_rgb8']
    peak = max(max(rgb) for rgb in active)
    maximum = max(max(rgb) for rgb in base)
    if active == [[min(c, peak) for c in rgb] for rgb in base]:
        assert peak <= maximum and ((peak >> 2) << 2 | (peak >> 2) >> 4) == peak
        return {'kind': 'clamp', 'native_limit': peak,
                'hd_limit': peak * 255 // maximum, 'cycle_steps': 0}
    for count in range(22):
        values = [rgb.copy() for rgb in contract['target_palette_dac6']]
        for _ in range(count):
            values = [[c + (c != 21) for c in rgb] for rgb in values]
        expected = [[((c & 63) << 2) | ((c & 63) >> 4) for c in rgb] for rgb in values]
        if active == expected:
            return {'kind': 'cycle', 'native_limit': 0, 'hd_limit': 0, 'cycle_steps': count}
    raise AssertionError(('unrecognized native palette', active))


def apply_effect(painting, effect):
    if effect['kind'] == 'clamp':
        table = [min(c, effect['hd_limit']) for c in range(256)]
    else:
        table = []
        for c in range(256):
            value = 0 if c == 0 else min(63, (c >> 2) + 1)
            for _ in range(effect['cycle_steps']):
                if value != 21:
                    value += 1
            value &= 63
            table.append((value << 2) | (value >> 4))
    return painting.point(table * 3)


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', choices=['opening', 'opening-missing', 'opening-effects', 'opening-palette', 'opening-palette-late'], required=True)
    parser.add_argument('--revision', choices=['palette'], default='palette')
    args = parser.parse_args()
    missing = args.mode == 'opening-missing'
    palette_probe = args.mode.startswith('opening-palette')
    effects = args.mode == 'opening-effects'
    root, project = Path('/output'), Path('/project')
    verifier_source = root / 'verifier-source.txt'
    assert not verifier_source.exists(), 'use a fresh verification directory'
    verifier_source.write_bytes(Path(__file__).read_bytes())
    history = project / 'workplace/hd-prototype/catalog/remaining/history'
    source_proof = json.loads((history / 'source-verification.json').read_text())
    static = json.loads((project / 'workplace/hd-prototype/catalog/graphics/static/profiles.json').read_text())
    op4 = next(record for record in static['records'] if record['name'] == 'OP4.OZM')
    full = json.loads((history / 'art-v2/profiles.json').read_text())['record']
    effect_contract = json.loads((history / 'palette-effects.json').read_text())
    assert effect_contract['verified']
    stages = [('OP4.OZM', op4['asset'], op4['paint_rect'], 245000000),
              ('OP-FULL.OZM', full['asset'], [0, 0, 640, 400], 539000000)]
    if palette_probe or effects:
        stages = [('OP4.OZM', op4['asset'], op4['paint_rect'], step) for step in [240000000, 245000000, 249000000]]
        stages += [('OP-FULL.OZM', full['asset'], [0, 0, 640, 400], step) for step in [534000000, 539000000, 543000000]]
    if effects:
        stages += [('OP-FULL.OZM', full['asset'], [0, 0, 640, 400], step) for step in [546000000, 548000000]]
    if args.mode == 'opening-palette-late':
        stages = [('OP-FULL.OZM', full['asset'], [0, 0, 640, 400], step) for step in [546000000, 548000000]]
    reference_steps = {}
    if missing:
        normal = json.loads((history / 'desktop' / args.revision / 'verification-opening.json').read_text())
        assert normal['verified']
        reference_steps = {name: proof['step'] for name, proof in normal['stages'].items()}
    (root / 'startup-probe').mkdir(exist_ok=True)
    command = [str(root / 'tokugawa-desktop-prototype'), '-desktop', '-desktop-speed', '60',
               '-steps', '600000000', '-mouse-file',
               '/project/workplace/hd-prototype/catalog/remaining/source-boundary/menu-history/mouse.json',
               '-hd-battle-frame-proof']
    if missing:
        directory = root / 'missing-graphics'
        directory.mkdir()
        archive = project / 'assets/hd-b/art'
        for source in archive.rglob('*.png'):
            relative = source.relative_to(archive)
            if str(relative) == full['asset']:
                continue
            target = directory / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            target.symlink_to(source)
        command += ['-hd-graphics-dir', str(directory)]
    env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
    children = []

    def run(*command, **kwargs):
        return subprocess.run(command, env=env, timeout=15, check=True, **kwargs)

    def status():
        try:
            return json.loads((root / 'desktop-status.json').read_text())
        except (FileNotFoundError, json.JSONDecodeError):
            return {}

    def wait(predicate, seconds=30):
        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            current = status()
            if predicate(current):
                return current
            assert process.poll() is None, f'desktop exited: {process.returncode}'
            time.sleep(.03)
        raise TimeoutError(f'opening wait at step {status().get("step")}')

    def key(name):
        before = status()
        run('xdotool', 'keydown', '--clearmodifiers', name)
        try:
            if name == 'F6':
                wait(lambda s: s.get('enabled') != before.get('enabled'), 15)
            elif name == 'F7':
                wait(lambda s: s.get('paused') != before.get('paused'), 15)
            else:
                wait(lambda s: len(s.get('events') or []) > len(before.get('events') or []), 15)
        finally:
            run('xdotool', 'keyup', name)
        time.sleep(.12)

    def capture(label):
        key('F9')
        current = status()
        event = current['events'][-1]
        assert event['kind'] == 'capture' and event['paused']
        geometry = dict(line.split('=', 1) for line in run('xdotool', 'getwindowgeometry', '--shell', window,
                                                        stdout=subprocess.PIPE).stdout.decode().splitlines())
        path = root / f'gui-{args.mode}-{label}.png'
        run('ffmpeg', '-v', 'error', '-y', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab',
            '-video_size', f'{geometry["WIDTH"]}x{geometry["HEIGHT"]}',
            '-i', f':98+{geometry["X"]},{geometry["Y"]}', '-frames:v', '1', str(path), stdout=subprocess.DEVNULL)
        assert Image.open(path).convert('RGBA').tobytes() == Image.open(event['frame']['path']).convert('RGBA').tobytes()
        return {'actual_png': path.name, 'sha256': sha(path), 'event': event,
                'window_pixels_equal': True, 'graphics': current['graphics']}

    try:
        with (root / 'xvfb.log').open('wb') as stdout:
            xvfb = subprocess.Popen(['Xvfb', ':98', '-screen', '0', '1600x1000x24', '-nolisten', 'tcp', '-ac'],
                                    env=env, stdout=stdout, stderr=subprocess.STDOUT)
        children.append(xvfb)
        deadline = time.monotonic() + 10
        while not Path('/tmp/.X11-unix/X98').exists():
            assert xvfb.poll() is None and time.monotonic() < deadline
            time.sleep(.05)
        receipt_path = root / f'desktop-{args.mode}.json'
        proofs = {}
        with receipt_path.open('wb') as stdout, (root / 'desktop.stderr.log').open('wb') as stderr:
            process = subprocess.Popen(command, env=env, stdout=stdout, stderr=stderr)
            children.append(process)
            wait(lambda s: s.get('step', 0) > 1000000, 60)
            window = run('xdotool', 'search', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
            run('xdotool', 'windowfocus', '--sync', window)
            run('xdotool', 'mousemove', '--window', window, '700', '500')
            key('F6')
            for name, asset_name, rect, probe_minimum in stages:
                expected_mode = 'fallback_missing_asset' if missing and name == 'OP-FULL.OZM' else 'hd'
                minimum = reference_steps.get(name, probe_minimum)

                def ready(s):
                    return s.get('step', 0) >= minimum and any(
                        record['resource'] == name and (palette_probe or record['mode'] == expected_mode)
                        for record in s.get('graphics') or [])

                wait(ready, 600)
                key('F7')
                step = status()['step']
                label = name.replace('.OZM', '') + (f'-{minimum}' if palette_probe or effects else '')
                enhanced = capture(label + '-enabled')
                key('F6')
                original = capture(label + '-original')
                key('F6')
                restored = capture(label + '-restored')
                assert enhanced['event']['proof'] == original['event']['proof'] == restored['event']['proof']
                assert enhanced['event']['frame']['png_sha256'] == restored['event']['frame']['png_sha256']
                reports = enhanced['graphics']
                assert len(reports) == 1 and reports[0]['resource'] == name
                if palette_probe:
                    expected_mode = reports[0]['mode']
                assert reports[0]['mode'] == expected_mode
                source = next(record for record in source_proof['full_copy_records'] if record['source'] == name)
                native = enhanced['event']['frame']['native_indices']
                assert sha(root / native['path']) == native['sha256'] == source['indexed_sha256']
                active_palette = [native['palette'][i] for i in native['dac']]
                effect = palette_effect(active_palette, effect_contract['records'][name])
                before = Image.open(root / original['actual_png']).convert('RGB')
                after = Image.open(root / enhanced['actual_png']).convert('RGB')
                painting = Image.open(project / 'assets/hd-b/art' / asset_name).convert('RGB')
                assert before.size == after.size == painting.size == (1280, 800)
                expected = before.copy()
                physical = tuple(value * 2 for value in rect)
                painted = (rect[2] - rect[0]) * (rect[3] - rect[1])
                if expected_mode == 'hd':
                    assert reports[0]['palette_effect'] == effect
                    expected.paste(apply_effect(painting, effect).crop(physical), physical[:2])
                assert expected.tobytes() == after.tobytes(), (name, 'HD sampling or pixels outside region')
                changed = sum(a != b for a, b in zip(before.getdata(), after.getdata()))
                assert (changed > 0) == (expected_mode == 'hd')
                assert reports[0]['painted_native_pixels'] == (painted if expected_mode == 'hd' else 0)
                stage_key = f'{name}@{minimum}' if palette_probe or effects else name
                proofs[stage_key] = {'step': step, 'mode': expected_mode, 'paint_rect': rect,
                                'checked_pixels': painted * 4, 'changed_pixels': changed,
                                'pictures': [enhanced, original, restored], 'source_indices_sha256': native['sha256'],
                                'active_palette': active_palette, 'palette_effect': effect}
                key('F7')
            process.wait(timeout=180)
            assert process.returncode == 0
        receipt = json.loads(receipt_path.read_text())
        baseline_root = history.parent / 'source-boundary/menu-history/rcr'
        baseline = json.loads((baseline_root / 'receipt.json').read_text())
        fields = source_proof['compared_fields']
        for field in fields:
            assert receipt[field] == baseline[field], field
        assert sha(root / 'startup-memory.bin') == source_proof['complete_RAM_sha256']
        toggles = [event for event in receipt['desktop']['events'] if event['kind'] == 'toggle']
        assert len(toggles) == 1 + 2 * len(stages) and all(event['guest_unchanged'] and event['before'] == event['after'] for event in toggles)
        for name, digest in receipt['integration_input_sha256'].items():
            assert sha(root / 'source-snapshot' / name) == digest == sha(project / name), name
        loaded = receipt['desktop']['assets']['graphics']
        assert len(loaded) == 29
        for name, info in loaded.items():
            assert info['validation'] == ('fallback_missing_asset' if missing and name == 'OP-FULL.OZM' else 'ready'), name
        result = {'verified': True, 'mode': args.mode, 'receipt_sha256': sha(receipt_path),
                  'classification': 'native palette sampling; not normal HD completion' if palette_probe else 'normal HD complete-copy verification',
                  'tool_sha256': sha(verifier_source), 'binary_sha256': sha(root / 'tokugawa-desktop-prototype'),
                  'source_proof_sha256': sha(history / 'source-verification.json'), 'stages': proofs,
                  'palette_contract_sha256': sha(history / 'palette-effects.json'),
                  'art_profiles_sha256': sha(history / 'art-v2/profiles.json'),
                  'compared_fields': fields, 'complete_RAM_sha256': sha(root / 'startup-memory.bin'),
                  'limitations': ['partial copies and scrolling wrappers remain outside this HD verification',
                                  'all-game HD and cross-platform packaging remain incomplete']}
        (root / f'verification-{args.mode}.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
        print(json.dumps({'mode': args.mode, 'verified': True, 'stages': list(proofs)}))
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
