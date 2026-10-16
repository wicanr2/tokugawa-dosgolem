"""錄製正式 Linux 封包的普通啟動、首戰與動態原圖／HD 切換，僅供本機影片。"""
import argparse
import ctypes
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--bundle', type=Path, default=Path('/bundle'))
p.add_argument('--finalize-existing', action='store_true', help='核對已完成的遊戲收據與 FFmpeg 正常封存錄影，不重新執行遊戲')
args = p.parse_args()
root, state = Path('/output'), Path('/output/session')
assert root.stat().st_uid == 1000
if not args.finalize_existing:
    assert not state.exists()
    state.mkdir()
inputs = json.loads(Path('/project/tools/player-path.json').read_text())
mouse = state / 'player-path.json'
if args.finalize_existing:
    assert json.loads(mouse.read_text()) == inputs
else:
    mouse.write_text(json.dumps(inputs))
manifest = json.loads((args.bundle / 'manifest.json').read_text())
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
for name, expected in manifest['files'].items():
    assert (args.bundle / name).stat().st_size == expected['size'] and sha(args.bundle / name) == expected['sha256'], name
env = dict(os.environ, DISPLAY=':95', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
children, timeline = [], []
process = None
start = None


def finalize(stop_method):
    receipt = json.loads((root / 'game-receipt.json').read_text())
    assert receipt['stop'] == 'instruction_budget' and receipt['successful_steps'] == 760000000
    assert receipt['publisher_prelude']['receipt']['stop'] == 'dos_exit'
    events = receipt['desktop']['events']
    assert all(e['guest_unchanged'] for e in events if e['kind'] == 'toggle')
    saved_timeline = json.loads((root / 'timeline.json').read_text())
    captures = [r['capture'] for r in saved_timeline if r['label'] in ['battle-hd', 'battle-original', 'battle-restored']]
    assert len(captures) == 3 and captures[0]['proof'] == captures[1]['proof'] == captures[2]['proof']
    probe = json.loads(subprocess.check_output(['ffprobe', '-v', 'error', '-show_format', '-show_streams', '-of', 'json', str(root / 'raw.mkv')]))
    video = probe['streams'][0]
    assert [video['width'], video['height']] == [1280, 1000] and video['avg_frame_rate'] == '20/1'
    assert float(probe['format']['duration']) > saved_timeline[-1]['recording_seconds']
    for name, expected in manifest['files'].items():
        assert sha(args.bundle / name) == expected['sha256'], name
    result = {'result': 'PASS', 'version': manifest['version'], 'binary_commit': manifest['commit'],
              'binary_sha256': sha(args.bundle / 'tokugawa-dosgolem'), 'input_sha256': sha(mouse),
              'recording': 'raw.mkv', 'recording_sha256': sha(root / 'raw.mkv'), 'FPS': 20,
              'recording_canvas': [1280, 1000], 'desktop_speed': 60, 'publisher_speed': 4,
              'source': 'actual packaged application, ordinary LE entry and scripted mouse input',
              'guest_state_unchanged_by_switch': True, 'original_package_files_unchanged': True,
              'normal_playthrough_verified': False, 'rights': 'local-only original game recording',
              'recorder_stop': stop_method, 'ffprobe': probe, 'timeline': saved_timeline,
              'recording_tool_sha256': sha(root / 'record-promo.py'), 'finalizer_tool_sha256': sha(Path(__file__))}
    (root / 'recording.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')


if args.finalize_existing:
    finalize('first recording stopped by SIGINT; ffmpeg exit 255 expected; old assertion wrongly required zero; original capture retained')
    raise SystemExit(0)


def run(*command, **kwargs):
    return subprocess.run(command, env=env, check=True, timeout=20, **kwargs)


def status():
    try:
        return json.loads((state / 'desktop-status.json').read_text())
    except (FileNotFoundError, json.JSONDecodeError):
        return {}


def wait(predicate, seconds=180):
    end = time.monotonic() + seconds
    while time.monotonic() < end:
        s = status()
        if predicate(s):
            return s
        assert process.poll() is None, (process.returncode, (root / 'game-stderr.log').read_text()[-1800:])
        time.sleep(.03)
    raise TimeoutError(str(status())[:1500])


def mark(label):
    s = status()
    row = {'label': label, 'recording_seconds': time.monotonic() - start,
           'state': {k: s.get(k) for k in ['program', 'step', 'enabled', 'mode', 'portrait_resource', 'paused',
                    'bar_visible', 'startup_pending', 'window_size', 'terrain_hd_count', 'terrain_scene']}}
    timeline.append(row)
    (root / 'timeline.json').write_text(json.dumps(timeline, ensure_ascii=False, indent=2) + '\n')
    print(label, s.get('step'), round(row['recording_seconds'], 2), flush=True)
    return row


def window():
    ids = run('xdotool', 'search', '--onlyvisible', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()
    assert len(ids) == 1, ids
    run('xdotool', 'windowmove', ids[0], '0', '0')
    run('xdotool', 'windowfocus', '--sync', ids[0])
    run('xdotool', 'mousemove', '--window', ids[0], '700', '500')
    return ids[0]


def key(name, predicate):
    run('xdotool', 'keydown', '--clearmodifiers', name)
    try:
        s = wait(predicate, 20)
    finally:
        run('xdotool', 'keyup', name)
    time.sleep(.2)
    return s


def click(x, y, predicate):
    run('xdotool', 'mousemove', '--window', win, str(x), str(y))
    run('xdotool', 'mousedown', '1')
    try:
        s = wait(predicate, 20)
    finally:
        run('xdotool', 'mouseup', '1')
    time.sleep(.2)
    return s


def shot(label):
    count = len(status().get('events') or [])
    key('F9', lambda s: len(s.get('events') or []) > count)
    event = status()['events'][-1]
    assert event['kind'] == 'capture' and event['paused']
    mark(label)['capture'] = event
    run('ffmpeg', '-v', 'error', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab',
        '-draw_mouse', '0', '-video_size', '1280x1000', '-i', ':95+0,0', '-frames:v', '1', str(root / (label + '.png')))
    return event


try:
    with (root / 'xvfb.log').open('wb') as log:
        xvfb = subprocess.Popen(['Xvfb', ':95', '-screen', '0', '1280x1000x24', '-ac', '-nolisten', 'tcp'], env=env, stdout=log, stderr=log)
    children.append(xvfb)
    end = time.monotonic() + 10
    while not Path('/tmp/.X11-unix/X95').exists():
        assert xvfb.poll() is None and time.monotonic() < end
        time.sleep(.05)
    with (root / 'recording.log').open('wb') as log:
        recorder = subprocess.Popen(['ffmpeg', '-v', 'warning', '-threads', '2', '-filter_threads', '1',
              '-f', 'x11grab', '-draw_mouse', '0', '-framerate', '20', '-video_size', '1280x1000', '-i', ':95+0,0',
              '-t', '1000', '-c:v', 'libx264', '-preset', 'veryfast', '-crf', '18', '-pix_fmt', 'yuv420p',
              '-threads', '2', str(root / 'raw.mkv')], env=env, stdin=subprocess.PIPE, stdout=log, stderr=log)
    children.append(recorder)
    start = time.monotonic()
    command = [str(args.bundle / 'tokugawa-dosgolem'), '-diagnostics', '-state-dir', str(state),
               '-desktop-audio=false', '-desktop-speed', '60', '-publisher-speed', '4',
               '-steps', '760000000', '-mouse-file', str(mouse)]
    with (root / 'game-receipt.json').open('wb') as out, (root / 'game-stderr.log').open('wb') as err:
        process = subprocess.Popen(command, env=env, stdout=out, stderr=err)
        children.append(process)
        wait(lambda s: s.get('program') == 'mark' and s.get('startup_pending'))
        win = window()
        assert status()['step'] == 0
        mark('startup-choice')
        time.sleep(2)
        click(900, 416, lambda s: not s.get('startup_pending', True))
        wait(lambda s: s.get('program') == 'mark' and s.get('mode') == 'hd' and s.get('step', 0) > 5000000)
        mark('publisher-hd')
        wait(lambda s: s.get('program') == 'game' and s.get('step', 0) > 1000000)
        win = window()
        mark('game-opened')
        for milestone in [80000000, 100000000, 130000000, 160000000, 200000000, 250000000,
                          300000000, 375000000, 450000000, 550000000, 650000000, 730000000]:
            wait(lambda s: s.get('step', 0) >= milestone, 150)
            mark('game-' + str(milestone))
        key('F7', lambda s: s.get('paused'))
        assert status()['terrain_scene']['floor_hd_count'] == 67
        hd = shot('battle-hd')
        time.sleep(3)
        key('F6', lambda s: not s.get('enabled', True))
        original = shot('battle-original')
        time.sleep(3)
        key('F6', lambda s: s.get('enabled'))
        restored = shot('battle-restored')
        assert hd['proof'] == original['proof'] == restored['proof']
        time.sleep(2)
        key('Escape', lambda s: s.get('bar_visible'))
        mark('bar-open')
        time.sleep(2)
        click(150, 32, lambda s: s.get('bar_menu') == 0)
        mark('bar-display-menu')
        time.sleep(2)
        click(150, 88, lambda s: not s.get('enabled', True) and s.get('bar_menu') == -1)
        mark('bar-original')
        time.sleep(2)
        click(150, 32, lambda s: s.get('bar_menu') == 0)
        click(150, 136, lambda s: s.get('enabled') and s.get('bar_menu') == -1)
        mark('bar-hd')
        time.sleep(2)
        key('Escape', lambda s: not s.get('bar_visible', True))
        key('F7', lambda s: not s.get('paused', True))
        assert process.wait(timeout=120) == 0
    recorder.stdin.write(b'q\n')
    recorder.stdin.flush()
    assert recorder.wait(timeout=30) == 0
    finalize('stdin q; clean ffmpeg exit')
finally:
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try:
                child.wait(timeout=10)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)
