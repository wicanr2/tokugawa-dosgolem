"""使用隔離入口執行原版結局，擷取真正 X11 與狀態不變的 HD 切換。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time

from PIL import Image

assert Path('/.dockerenv').is_file() and os.getuid() == 1000
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000
parser = argparse.ArgumentParser()
parser.add_argument('--mode', choices=['hd', 'missing-end2'], required=True)
parser.add_argument('--timeout', type=int, default=1500)
args = parser.parse_args()
assert 60 <= args.timeout <= 2400
env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
sources = read(root/'build-sources.json')
assert len(sources) == 106 and all(sha(project/p) == h for p, h in sources.items())
manifest = read(root/'validation-manifest.json')
assert manifest['validation_only'] and not manifest['normal_playthrough_verified']
assert sources == manifest['controlled_compiled_sources']
assert sha(Path(__file__)) == sha(root/'source-snapshot/tools/verify-controlled-ending.py')
children, captures, stages = [], [], []
process = None
deadline = time.monotonic()+args.timeout
selected_sources = set()


def run(*command, **kwargs):
    return subprocess.run(command, env=env, check=True, timeout=15, **kwargs)


def status():
    try:
        return read(root/'desktop-status.json')
    except (FileNotFoundError, json.JSONDecodeError):
        return {}


def wait(predicate, seconds=30):
    until = min(deadline, time.monotonic()+seconds)
    while time.monotonic() < until:
        s = status()
        if predicate(s):
            return s
        assert process.poll() is None, 'controlled process stopped before expected desktop state'
        time.sleep(.05)
    raise TimeoutError(str(status())[:1800])


def key(name, predicate):
    run('xdotool', 'keydown', '--clearmodifiers', name)
    try:
        result = wait(predicate)
    finally:
        run('xdotool', 'keyup', name)
    time.sleep(.08)
    return result


def capture(stage, mode):
    count = len(status().get('events') or [])
    s = key('F9', lambda state: len(state.get('events') or []) > count)
    event = s['events'][-1]
    assert event['kind'] == 'capture' and event['paused']
    label = f"stage-{stage['id']:03d}-{mode}"
    geometry = run('xdotool', 'getwindowgeometry', '--shell', window, stdout=subprocess.PIPE).stdout.decode()
    g = dict(line.split('=', 1) for line in geometry.splitlines())
    actual_path = root/(label+'.png')
    run('ffmpeg', '-v', 'error', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab', '-draw_mouse', '0',
        '-video_size', g['WIDTH']+'x'+g['HEIGHT'], '-i', ':98+'+g['X']+','+g['Y'], '-frames:v', '1', str(actual_path))
    actual = Image.open(actual_path).convert('RGBA')
    if s['bar_visible']:
        actual = actual.crop((0, 64, 1280, 864))
    expected = Image.open(event['frame']['path']).convert('RGBA')
    assert actual.size == expected.size == (1280, 800) and actual.tobytes() == expected.tobytes()
    assert event['proof']['ram_sha256'] == stage['ram_sha256']
    assert event['proof']['step'] == stage['step'] and event['proof']['eip'] == stage['eip']
    assert event['proof']['raw_indexed_sha256'] == stage['indices_sha256']
    (root/(label+'.json')).write_text(json.dumps(s, ensure_ascii=False, indent=2)+'\n')
    captures.append(dict(label=label, stage_id=stage['id'], mode=mode, stage_label=stage['label'],
                         actual=str(actual_path), actual_sha256=sha(actual_path), event=event,
                         state_path=label+'.json', window_pixels_equal=True))


def selected(stage):
    label = stage['label']
    if label in ['END-2-shift-000', 'END-2-shift-120', 'END-2-shift-240', 'END-11B-final-copy']:
        return True
    if '.OZM-delay-' not in label:
        return False
    source, ret = label.split('.OZM-delay-')
    ret = int(ret, 16)
    key = source
    if source == 'END-2B':
        key = (source, json.dumps(stage['last_graphics_return']['SourceRect'], sort_keys=True))
    if not 0x1872d <= ret < 0x18db4 or key in selected_sources:
        return False
    selected_sources.add(key)
    return True


try:
    with (root/'xvfb.log').open('wb') as log:
        xvfb = subprocess.Popen(['Xvfb', ':98', '-screen', '0', '1600x1200x24', '-nolisten', 'tcp', '-ac'],
                                env=env, stdout=log, stderr=subprocess.STDOUT)
    children.append(xvfb)
    until = time.monotonic()+10
    while not Path('/tmp/.X11-unix/X98').exists():
        assert xvfb.poll() is None and time.monotonic()<until
        time.sleep(.05)
    run('xdotool', 'mousemove', '1590', '1190')
    command = [str(root/'tokugawa-desktop-prototype'), '-desktop', '-publisher-intro=false', '-startup-display', 'original',
               '-desktop-speed', '60', '-steps', '2000000000']
    if args.mode == 'missing-end2':
        graphics = root/'graphics'
        graphics.mkdir()
        excluded = Path('remaining/prepared/END-2.OZM-hd-v3.png')
        for path in (project/'assets/hd-b/art').rglob('*'):
            if path.is_file() and path.relative_to(project/'assets/hd-b/art') != excluded:
                dest = graphics/path.relative_to(project/'assets/hd-b/art')
                dest.parent.mkdir(parents=True, exist_ok=True)
                dest.symlink_to(path)
        command += ['-hd-graphics-dir', str(graphics)]
    with (root/'receipt.json').open('wb') as out, (root/'stderr.log').open('wb') as err:
        process = subprocess.Popen(command, env=env, stdout=out, stderr=err)
    children.append(process)
    wait(lambda s: s.get('program') == 'game', 90)
    window = run('xdotool', 'search', '--onlyvisible', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
    run('xdotool', 'windowfocus', '--sync', window)
    previous = 0
    while time.monotonic()<deadline:
        if process.poll() is not None:
            break
        path = root/'ending-stage.json'
        try:
            stage = read(path)
        except (FileNotFoundError, json.JSONDecodeError):
            time.sleep(.05)
            continue
        if stage['id'] == previous:
            time.sleep(.05)
            continue
        assert stage['id'] == previous+1 and stage['validation_only'] and not stage['normal_playthrough_verified']
        wait(lambda s: s.get('step', 0) == stage['step'])
        stages.append(stage)
        if selected(stage):
            if not status().get('paused'):
                key('F7', lambda s: s.get('paused'))
            capture(stage, 'original')
            key('F6', lambda s: s.get('enabled'))
            capture(stage, 'hd')
            key('F6', lambda s: not s.get('enabled'))
            capture(stage, 'restored')
        (root/f"ending-continue-{stage['id']:03d}").write_text('continue\n')
        if status().get('paused'):
            key('F7', lambda s: not s.get('paused'))
        previous = stage['id']
    if process.poll() is None:
        raise TimeoutError('controlled ending exceeded the bounded deadline')
    assert process.returncode == 0
    receipt = read(root/'receipt.json')
    assert receipt['stop'] == 'dos_exit' and receipt['exit_code'] == 0
    assert receipt['integration_input_sha256'] == sources and receipt['ending_validation_stages'] == stages
    report = dict(status='PASS_CAPTURE', validation_only=True, normal_playthrough_verified=False, mode=args.mode,
                  stages=stages, captures=captures, binary_sha256=sha(root/'tokugawa-desktop-prototype'),
                  tool_sha256=sha(Path(__file__)), source_manifest_sha256=sha(root/'validation-manifest.json'),
                  receipt_sha256=sha(root/'receipt.json'))
    (root/'controlled-ending-captures.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(f"captured {len(captures)} actual windows at {len(stages)} controlled stages; original exit 0")
finally:
    if process is not None and process.poll() is None:
        try:
            run('xdotool', 'keydown', 'Shift_L')
            run('xdotool', 'keydown', 'Escape')
            process.wait(timeout=30)
        except Exception:
            process.terminate()
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait()
