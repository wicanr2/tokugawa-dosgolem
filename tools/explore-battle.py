"""有界正常戰場研究。commands.json 只接受公開 X11 輸入，不寫 guest 狀態。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time
from PIL import Image

assert Path('/.dockerenv').exists() and os.getuid() == 1000
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000
env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
parser = argparse.ArgumentParser()
parser.add_argument('--timeout', type=int, default=3600)
parser.add_argument('--prefix-step', type=int, default=741000000)
parser.add_argument('--harness-only', action='store_true')
parser.add_argument('--desktop-speed', type=int, default=60)
parser.add_argument('--mouse-file', default='/project/tools/player-path.json')
parser.add_argument('--input-manifest', type=Path)
parser.add_argument('--graphics-dir', type=Path)
parser.add_argument('--unit-dir', type=Path)
parser.add_argument('--keep-pointer-outside', action='store_true')
args = parser.parse_args()
assert 30 <= args.timeout <= 7200 and 0 <= args.prefix_step <= 1000000000
assert 1 <= args.desktop_speed <= 60
children, captures, commands = [], [], []
process = None
expected_input = json.loads(Path(args.mouse_file).read_text())
data_origin = None
deadline = time.monotonic() + args.timeout


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def run(*args, **kwargs):
    return subprocess.run(args, env=env, check=True, timeout=15, **kwargs)


def status():
    try:
        return json.loads((root / 'desktop-status.json').read_text())
    except FileNotFoundError:
        return {}


def wait(test, limit=90):
    end = min(deadline, time.monotonic() + limit)
    while time.monotonic() < end:
        s = status()
        if test(s):
            return s
        if process.poll() is not None:
            raise RuntimeError('original process stopped; inspect receipt.json and stderr.log')
        time.sleep(.05)
    raise TimeoutError(str(status())[:2000])


def key(name, test):
    run('xdotool', 'keydown', '--clearmodifiers', name)
    try:
        s = wait(test)
    finally:
        run('xdotool', 'keyup', name)
    time.sleep(.1)
    return s


def paused(value):
    if status().get('paused') != value:
        key('F7', lambda s: s.get('paused') == value)


def capture(label):
    paused(True)
    count = len(status().get('events') or [])
    s = key('F9', lambda s: len(s.get('events') or []) > count)
    event = s['events'][-1]
    assert event['kind'] == 'capture' and event['paused']
    geometry = run('xdotool', 'getwindowgeometry', '--shell', window, stdout=subprocess.PIPE).stdout.decode()
    g = dict(line.split('=', 1) for line in geometry.splitlines())
    path = root / (label + '.png')
    run('ffmpeg', '-v', 'error', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab', '-draw_mouse', '0',
        '-video_size', g['WIDTH'] + 'x' + g['HEIGHT'], '-i', ':98+' + g['X'] + ',' + g['Y'], '-frames:v', '1', str(path))
    actual, expected = Image.open(path).convert('RGBA'), Image.open(event['frame']['path']).convert('RGBA')
    if s['bar_visible']:
        actual = actual.crop((0, 64, 1280, 864))
    assert actual.size == expected.size == (1280, 800) and actual.tobytes() == expected.tobytes()
    record = dict(label=label, actual=str(path), sha256=sha(path), event=event, window_pixels_equal=True,
                  composite=s.get('composite_observation'), mouse=s.get('desktop_mouse'))
    captures.append(record)
    (root / ('state-' + label + '.json')).write_text(json.dumps(s, ensure_ascii=False, indent=2) + '\n')
    return record


def report(phase, error=None):
    d = dict(phase=phase, captures=captures, commands=commands, step=status().get('step'), error=error,
             binary_sha256=sha(root / 'tokugawa-desktop-prototype'), tool_sha256=sha(Path(__file__)),
             harness_only=args.harness_only, timeout_seconds=args.timeout,
             input_data=data_origin, input_mouse_sha256=sha(Path(args.mouse_file)), desktop_speed=args.desktop_speed,
             graphics_dir=str(args.graphics_dir) if args.graphics_dir else None, keep_pointer_outside=args.keep_pointer_outside)
    pending = root / 'exploration-status.pending'
    pending.write_text(json.dumps(d, ensure_ascii=False, indent=2) + '\n')
    pending.replace(root / 'exploration-status.json')


def close_original():
    if process is None or process.poll() is not None:
        return
    run('xdotool', 'keydown', 'Shift_L')
    try:
        run('xdotool', 'keydown', 'Escape')
        process.wait(timeout=30)
    finally:
        run('xdotool', 'keyup', 'Escape', 'Shift_L')
    assert process.returncode == 0
    receipt = json.loads((root / 'receipt.json').read_text())
    assert receipt['integration_input_sha256'] == sources
    assert receipt['mouse_input_schedule'] == expected_input and receipt['desktop_script_handoff']


try:
    sources = json.loads((root / 'build-sources.json').read_text())
    assert len(sources) == 106 and all(sha(project / n) == h for n, h in sources.items())
    assert sha(project / 'tools/player-path.json') == 'f621bd1f66e0d4bb492570317ec82b898ec95d3a1f4139a6ee5c451011b30395'
    dat_sha = sha(Path('/original/DX5.DAT'))
    if args.input_manifest:
        manifest = json.loads(args.input_manifest.read_text())
        assert manifest['classification'] == 'original program data plus original-game save; not downloaded baseline DAT'
        assert manifest['saved_dat_sha256'] == dat_sha
        assert all(sha(Path('/original') / f['path']) == f['sha256'] for f in manifest['files'])
        data_origin = dict(kind='original-game save', manifest=str(args.input_manifest),
                           manifest_sha256=sha(args.input_manifest), dat_sha256=dat_sha)
    else:
        assert dat_sha == 'de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37'
        data_origin = dict(kind='downloaded baseline', dat_sha256=dat_sha)
    with (root / 'xvfb.log').open('wb') as log:
        xvfb = subprocess.Popen(['Xvfb', ':98', '-screen', '0', '1600x1200x24', '-nolisten', 'tcp', '-ac'],
                                env=env, stdout=log, stderr=subprocess.STDOUT)
    children.append(xvfb)
    end = time.monotonic() + 10
    while not Path('/tmp/.X11-unix/X98').exists():
        assert xvfb.poll() is None and time.monotonic() < end
        time.sleep(.05)
    run('xdotool', 'mousemove', '1590', '1190')
    command = [str(root / 'tokugawa-desktop-prototype'), '-desktop', '-publisher-intro=false', '-startup-display', 'original',
               '-desktop-speed', str(args.desktop_speed), '-steps', '2000000000', '-mouse-file', args.mouse_file,
               '-desktop-script-handoff', '-hd-composite-trace', '-hd-graphics-trace']
    if args.graphics_dir:
        command += ['-hd-graphics-dir', str(args.graphics_dir)]
    if args.unit_dir:
        command += ['-hd-terrain-unit-dir', str(args.unit_dir)]
    with (root / 'receipt.json').open('wb') as out, (root / 'stderr.log').open('wb') as err:
        process = subprocess.Popen(command, env=env, stdout=out, stderr=err)
    children.append(process)
    wait(lambda s: s.get('program') == 'game', 90)
    window = run('xdotool', 'search', '--onlyvisible', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
    run('xdotool', 'windowfocus', '--sync', window)
    if args.keep_pointer_outside:
        run('xdotool', 'mousemove', '1590', '1190')
    wait(lambda s: s.get('step', 0) >= args.prefix_step and
         (args.harness_only or not s.get('scripted_input_pending')), 900)
    paused(True)
    capture('harness-start' if args.harness_only else ('saved-checkpoint' if args.input_manifest else 'first-battle'))
    report('ready')
    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError('original process stopped while awaiting X11 commands')
        try:
            pending = json.loads((root / 'commands.json').read_text())
        except FileNotFoundError:
            time.sleep(.1)
            continue
        assert isinstance(pending, list) and len(pending) <= 256
        if len(commands) >= len(pending):
            time.sleep(.1)
            continue
        command = pending[len(commands)]
        assert command['id'] == len(commands) + 1
        kind = command['kind']
        if kind == 'quit':
            close_original()
            commands.append(command)
            receipt = json.loads((root / 'receipt.json').read_text())
            report('finished')
            print('finished ordinary battle exploration', receipt['successful_steps'], receipt['stop'])
            break
        elif kind == 'capture':
            pass
        elif kind == 'toggle':
            enabled = not status()['enabled']
            key('F6', lambda s: s['enabled'] == enabled)
        elif kind in ['wait_graphics', 'wait_restore']:
            begin = status()['step']
            n = command.get('steps', 50000000)
            assert isinstance(n, int) and 100000 <= n <= 200000000 and begin + n < 2000000000
            resource = command.get('resource')
            target = command.get('rect', [216, 100, 424, 260])
            if kind == 'wait_graphics':
                assert status()['enabled'] and isinstance(resource, str) and resource.endswith('.OZM') and len(resource) <= 64
            else:
                assert len(target) == 4 and all(isinstance(v, int) for v in target)
            def reached(s):
                if s.get('step', 0) > begin + n:
                    raise TimeoutError('ordinary graphics wait exceeded instruction limit')
                if kind == 'wait_graphics':
                    return any(report.get('resource') == resource and report.get('mode') == 'hd' for report in s.get('graphics') or [])
                for event in (s.get('window_layers') or {}).get('events') or []:
                    rect = event['logical_rect']
                    bounds = [rect['Min']['X'], rect['Min']['Y'], rect['Max']['X'], rect['Max']['Y']]
                    if event['kind'] == 'restore_return' and event['verified'] and event['step'] >= begin and bounds == target:
                        return True
                return False
            paused(False)
            wait(reached)
            paused(True)
            assert reached(status()), 'graphics stage ended before the ordinary pause'
        elif kind in ['click', 'drag', 'run']:
            begin = status()['step']
            n = command.get('steps', 5000000)
            assert isinstance(n, int) and 100000 <= n <= 200000000 and begin + n < 2000000000
            hold_steps = command.get('hold_steps', 100000)
            assert isinstance(hold_steps, int) and 100000 <= hold_steps <= n
            paused(False)
            if kind in ['click', 'drag']:
                x, y, button = command['x'], command['y'], command.get('button', 1)
                assert 0 <= x < 640 and 0 <= y < 400 and button in [1, 3]
                run('xdotool', 'mousemove', '--window', window, str(x * 2 + 1), str(y * 2 + 1))
                wait(lambda s: (s.get('desktop_mouse') or [])[:2] == [x, y])
                run('xdotool', 'mousedown', str(button))
                try:
                    wait(lambda s: s.get('desktop_mouse', [0, 0, 0])[2] == (1 if button == 1 else 2))
                    if kind == 'drag':
                        tx, ty = command['to_x'], command['to_y']
                        assert 0 <= tx < 640 and 0 <= ty < 400
                        run('xdotool', 'mousemove', '--window', window, str(tx * 2 + 1), str(ty * 2 + 1))
                        wait(lambda s: (s.get('desktop_mouse') or [])[:2] == [tx, ty])
                    held_from = status()['step']
                    wait(lambda s: s.get('step', 0) >= held_from + hold_steps)
                finally:
                    run('xdotool', 'mouseup', str(button))
                wait(lambda s: s.get('desktop_mouse', [0, 0, 1])[2] == 0)
            wait(lambda s: s.get('step', 0) >= begin + n)
            paused(True)
        else:
            raise ValueError('unsupported X11 research command: ' + str(kind))
        commands.append(command)
        capture('action-' + str(command['id']))
        report('ready')
    else:
        close_original()
        report('finished_deadline')
        print('bounded exploration closed normally; complete receipt preserved')
except BaseException as error:
    cleanup = None
    try:
        close_original()
    except BaseException as close_error:
        cleanup = repr(close_error)
    report('failed', repr(error) + ('; normal close failed: ' + cleanup if cleanup else ''))
    raise
finally:
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)
