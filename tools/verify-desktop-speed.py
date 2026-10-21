"""以原版存檔、普通讀檔及交鋒輸入驗證速度選單與 Linux 真實視窗。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time
from PIL import Image

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--binary', required=True)
args = p.parse_args()
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == root.stat().st_gid == 1000
manifest_path = project / 'workplace/hd-prototype/catalog/remaining/consumers/battle/r11/save-turn3/ready-r2/save-import/manifest.json'
manifest = json.loads(manifest_path.read_text())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


for entry in manifest['files']:
    assert sha(Path('/original') / entry['path']) == entry['sha256'], entry['path']
prefix = project / 'workplace/hd-prototype/catalog/remaining/consumers/battle/r20/desktop-animation/ready/player-path.json'
inputs = json.loads(prefix.read_text())
assert len(inputs) == 19 and inputs[-1]['step'] == 171100000
state = root / 'state'
state.mkdir()
(state / 'player-path.json').write_text(json.dumps(inputs))
runtime = root / 'pulse'
runtime.mkdir(mode=0o700)
config = root / 'pulse.pa'
config.write_text('load-module module-null-sink sink_name=tokugawa rate=44100 channels=2\n'
                  f'load-module module-native-protocol-unix socket={runtime}/socket auth-anonymous=1\n'
                  'set-default-sink tokugawa\n')
env = dict(os.environ, DISPLAY=':96', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2',
           XDG_RUNTIME_DIR=str(runtime), PULSE_SERVER='unix:' + str(runtime / 'socket'))
children = []
process = None


def command(*values, **kw):
    return subprocess.run(values, env=env, check=True, timeout=20, **kw)


def status():
    try:
        return json.loads((state / 'desktop-status.json').read_text())
    except (FileNotFoundError, json.JSONDecodeError):
        return {}


def wait(test, seconds=180):
    end = time.monotonic() + seconds
    while time.monotonic() < end:
        s = status()
        if test(s):
            return s
        assert process.poll() is None, (process.returncode, (root / 'stderr.log').read_text()[-2000:])
        time.sleep(.02)
    raise TimeoutError(str(status())[:2000])


def key(name, test):
    command('xdotool', 'keydown', '--clearmodifiers', name)
    try:
        result = wait(test, 30)
    finally:
        command('xdotool', 'keyup', name)
    time.sleep(.1)
    return result


def click(x, y, test):
    command('xdotool', 'mousemove', '--window', window, str(x), str(y))
    command('xdotool', 'mousedown', '1')
    try:
        result = wait(test, 30)
    finally:
        command('xdotool', 'mouseup', '1')
    time.sleep(.1)
    return result


def grab(label):
    geometry = command('xdotool', 'getwindowgeometry', '--shell', window, stdout=subprocess.PIPE).stdout.decode()
    g = dict(line.split('=', 1) for line in geometry.splitlines())
    path = root / (label + '.png')
    serial = 2
    while path.exists():
        path = root / (label + '-' + str(serial) + '.png')
        serial += 1
    command('ffmpeg', '-v', 'error', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab', '-draw_mouse', '0',
            '-video_size', g['WIDTH'] + 'x' + g['HEIGHT'], '-i', ':96+' + g['X'] + ',' + g['Y'], '-frames:v', '1', str(path))
    return path


def capture(label):
    count = len(status().get('events') or [])
    s = key('F9', lambda s: len(s.get('events') or []) > count)
    event = s['events'][-1]
    assert event['kind'] == 'capture' and event['paused']
    path = grab(label)
    actual = Image.open(path).convert('RGBA')
    if s['bar_visible']:
        actual = actual.crop((0, 64, 1280, 864))
    expected = Image.open(event['frame']['path']).convert('RGBA')
    assert actual.size == expected.size and actual.tobytes() == expected.tobytes()
    return event


def choose_speed(speed):
    row = [1, 2, 4, 8, 16, 32].index(speed) + 2
    key('Escape', lambda s: s.get('bar_visible'))
    click(550, 32, lambda s: s.get('bar_menu') == 1)
    grab('speed-menu-' + str(speed))
    click(550, (32 + row * 24 + 12) * 2,
          lambda s: s.get('bar_menu') == -1 and (s.get('bar_events') or [{}])[-1].get('speed') == speed)
    command('xdotool', 'mousemove', '1590', '1190')
    key('Escape', lambda s: not s.get('bar_visible', True))


def graphic(s, name):
    return any(g.get('resource') == name and g.get('mode') == 'hd' for g in s.get('graphics') or [])


try:
    for cmd, label in [(['Xvfb', ':96', '-screen', '0', '1600x1200x24', '-ac'], 'xvfb'),
                       (['pulseaudio', '-n', '--daemonize=no', '--use-pid-file=no', '--exit-idle-time=-1', '-F', str(config)], 'pulse')]:
        log = (root / (label + '.log')).open('wb')
        child = subprocess.Popen(cmd, env=env, stdout=log, stderr=log)
        children.append(child)
    end = time.monotonic() + 15
    while not (runtime / 'socket').exists() or not Path('/tmp/.X11-unix/X96').exists():
        assert time.monotonic() < end and all(c.poll() is None for c in children)
        time.sleep(.1)
    command('xdotool', 'mousemove', '1590', '1190')
    cmd = [args.binary, '-play', '-diagnostics', '-publisher-intro=false', '-startup-display', 'hd',
           '-desktop-speed', '60', '-steps', '500000000', '-mouse-file', str(state / 'player-path.json'),
           '-resource-dir', '/project', '-game-dir', '/original', '-state-dir', str(state),
           '-cpu-source', '/project/workplace/desktop-prototype/package/r4/engine/internal/cpu386/cpu.go']
    with (root / 'receipt.json').open('wb') as out, (root / 'stderr.log').open('wb') as log:
        process = subprocess.Popen(cmd, env=env, stdout=out, stderr=log)
        children.append(process)
        wait(lambda s: s.get('step', 0) >= 190000000, 300)
        found = command('xdotool', 'search', '--onlyvisible', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()
        assert len(found) == 1
        window = found[0]
        command('xdotool', 'windowfocus', '--sync', window)
        key('F7', lambda s: s.get('paused'))
        choose_speed(4)
        before_combat = capture('before-combat')
        key('F7', lambda s: not s.get('paused', True))
        battle = wait(lambda s: graphic(s, 'BATTLE02.OZM'))
        battle_time = time.monotonic()
        animation = wait(lambda s: graphic(s, 'ANIME08.OZM'))
        animation_time = time.monotonic()
        grab('animation-live')
        def restored(s):
            for e in (s.get('window_layers') or {}).get('events') or []:
                rect = e.get('logical_rect') or {}
                if (e.get('kind') == 'restore_return' and e.get('verified') and e.get('step', 0) > animation['step']
                    and rect == {'Min': {'X': 216, 'Y': 100}, 'Max': {'X': 424, 'Y': 260}}):
                    return not graphic(s, 'ANIME08.OZM') and graphic(s, 'BATTLE02.OZM')
            return False
        restored_state = wait(restored)
        restored_time = time.monotonic()
        (root / 'combat-timing.json').write_text(json.dumps({
            'battle_begin_step': battle['step'], 'animation_step': animation['step'],
            'restored_step': restored_state['step'], 'speed': 4,
            'battle_to_restore_seconds': restored_time - battle_time,
            'animation_to_restore_seconds': restored_time - animation_time}, indent=2) + '\n')
        key('F7', lambda s: s.get('paused'))
        hd = capture('after-combat-hd')
        key('F6', lambda s: not s.get('enabled', True))
        original = capture('after-combat-original')
        key('F6', lambda s: s.get('enabled'))
        returned = capture('after-combat-restored')
        assert hd['proof'] == original['proof'] == returned['proof']
        for speed in [1, 2, 8, 16, 32, 4]:
            choose_speed(speed)
        final = status()
        assert all(e['guest_unchanged'] for e in final['bar_events'] if e['kind'] == 'option')
        assert final['paused'] and final['enabled']
        command('xdotool', 'keydown', 'Shift_L')
        command('xdotool', 'keydown', 'Escape')
        assert process.wait(timeout=45) == 0
        command('xdotool', 'keyup', 'Escape')
        command('xdotool', 'keyup', 'Shift_L')
    receipt = json.loads((root / 'receipt.json').read_text())
    assert receipt['stop'] == 'desktop_closed' and receipt['mouse_input_schedule'] == inputs
    for entry in manifest['files']:
        assert sha(Path('/original') / entry['path']) == entry['sha256']
    result = {'result': 'PASS', 'platform': 'Linux X11', 'classification': 'normal original-game save load and combat',
              'input_manifest_sha256': sha(manifest_path), 'mouse_inputs': len(inputs), 'combat_speed': 4,
              'ordinary_battle_begin_step': battle['step'], 'animation_step': animation['step'],
              'restored_step': restored_state['step'], 'battle_to_restore_seconds': restored_time - battle_time,
              'animation_to_restore_seconds': restored_time - animation_time, 'actual_X11_verified': True,
              'HD_original_restore_same_guest_state': True, 'all_six_speed_settings_guest_unchanged': True,
              'audio': receipt['desktop']['audio_output'], 'original_files_unchanged': True, 'close_normal': True}
    (root / 'verification-summary.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(result, ensure_ascii=False), flush=True)
finally:
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try:
                child.wait(timeout=10)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)
