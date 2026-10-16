"""從實際 ZIP 解包後，用 X11 驗證啟動選擇、MARK、標題與動態 HD 切換。"""
import argparse
import ctypes
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time
import zipfile
from PIL import Image

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--archive', type=Path, required=True)
p.add_argument('--platform', choices=['linux', 'windows'], required=True)
args = p.parse_args()
root = Path('/output')
assert root.stat().st_uid == 1000
unpacked = root / 'unpacked'
unpacked.mkdir()
with zipfile.ZipFile(args.archive) as z:
    assert z.testzip() is None
    for info in z.infolist():
        path = Path(info.filename)
        assert not path.is_absolute() and '..' not in path.parts
        z.extract(info, unpacked)
        (unpacked / path).chmod((info.external_attr >> 16) & 0o777 or 0o644)
dirs = list(unpacked.iterdir())
assert len(dirs) == 1 and dirs[0].is_dir()
package = dirs[0]
manifest = json.loads((package / 'manifest.json').read_text())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


for name, expected in manifest['files'].items():
    assert sha(package / name) == expected['sha256'] and (package / name).stat().st_size == expected['size'], name
windows = args.platform == 'windows'
state = root / 'user-data'
state.mkdir()
env = dict(os.environ, DISPLAY=':94', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2', LANG='C.UTF-8', LC_ALL='C.UTF-8',
           WINEPREFIX=str(root / 'wine-prefix'), WINEDEBUG='-all')
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
        value = status()
        if test(value):
            return value
        assert process.poll() is None, (process.returncode, (root / 'stderr.log').read_text()[-3500:])
        time.sleep(.05)
    raise TimeoutError(str(status())[:2000])


def key(name, test):
    command('xdotool', 'keydown', '--clearmodifiers', name)
    try:
        value = wait(test, 30)
    finally:
        command('xdotool', 'keyup', name)
    time.sleep(.2)
    return value


def find_window():
    ids = command('xdotool', 'search', '--onlyvisible', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()
    assert len(ids) == 1, ids
    command('xdotool', 'windowfocus', '--sync', ids[0])
    return ids[0]


def capture(label):
    count = len(status().get('events') or [])
    value = key('F9', lambda s: len(s.get('events') or []) > count)
    event = value['events'][-1]
    assert event['kind'] == 'capture' and event['paused']
    path = event['frame']['path']
    if windows:
        assert path.startswith('Z:') or path.startswith('z:'), path
        path = path[2:].replace('\\', '/')
    presented = Path(path)
    geometry = command('xdotool', 'getwindowgeometry', '--shell', window, stdout=subprocess.PIPE).stdout.decode()
    g = dict(line.split('=', 1) for line in geometry.splitlines())
    actual_path = root / (label + '.png')
    command('ffmpeg', '-v', 'error', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab', '-draw_mouse', '0',
            '-video_size', g['WIDTH'] + 'x' + g['HEIGHT'], '-i', ':94+' + g['X'] + ',' + g['Y'], '-frames:v', '1', str(actual_path))
    actual = Image.open(actual_path).convert('RGBA')
    if value['bar_visible']:
        actual = actual.crop((0, 64, 1280, value['canvas_height'] + 64))
    expected = Image.open(presented).convert('RGBA')
    assert actual.size == expected.size and actual.tobytes() == expected.tobytes(), label
    return {'actual': str(actual_path), 'sha256': sha(actual_path), 'presented': str(presented),
            'pixels': hashlib.sha256(actual.tobytes()).hexdigest(), 'proof': event['proof']}


def click(x, y, test):
    command('xdotool', 'mousemove', '--window', window, str(x), str(y))
    command('xdotool', 'mousedown', '1')
    try:
        result = wait(test, 30)
    finally:
        command('xdotool', 'mouseup', '1')
    time.sleep(.2)
    return result


def close_window():
    class ClientMessage(ctypes.Structure):
        _fields_ = [('type', ctypes.c_int), ('serial', ctypes.c_ulong), ('send_event', ctypes.c_int),
                    ('display', ctypes.c_void_p), ('window', ctypes.c_ulong), ('message_type', ctypes.c_ulong),
                    ('format', ctypes.c_int), ('data', ctypes.c_long * 5)]
    class Event(ctypes.Union):
        _fields_ = [('message', ClientMessage), ('padding', ctypes.c_long * 24)]
    x = ctypes.CDLL('libX11.so.6')
    x.XOpenDisplay.argtypes, x.XOpenDisplay.restype = [ctypes.c_char_p], ctypes.c_void_p
    x.XInternAtom.argtypes, x.XInternAtom.restype = [ctypes.c_void_p, ctypes.c_char_p, ctypes.c_int], ctypes.c_ulong
    x.XSendEvent.argtypes = [ctypes.c_void_p, ctypes.c_ulong, ctypes.c_int, ctypes.c_long, ctypes.POINTER(Event)]
    x.XFlush.argtypes = x.XCloseDisplay.argtypes = [ctypes.c_void_p]
    connection = x.XOpenDisplay(b':94')
    assert connection
    try:
        event = Event()
        event.message = ClientMessage(type=33, display=connection, window=int(window),
                                      message_type=x.XInternAtom(connection, b'WM_PROTOCOLS', 0), format=32)
        event.message.data[0] = x.XInternAtom(connection, b'WM_DELETE_WINDOW', 0)
        assert x.XSendEvent(connection, int(window), 0, 0, ctypes.byref(event))
        x.XFlush(connection)
    finally:
        x.XCloseDisplay(connection)


try:
    xvfb_log = (root / 'xvfb.log').open('wb')
    xvfb = subprocess.Popen(['Xvfb', ':94', '-screen', '0', '1600x1200x24', '-ac'], env=env, stdout=xvfb_log, stderr=xvfb_log)
    children.append(xvfb)
    end = time.monotonic() + 10
    while not Path('/tmp/.X11-unix/X94').exists():
        assert xvfb.poll() is None and time.monotonic() < end
        time.sleep(.05)
    executable = package / ('tokugawa-dosgolem.exe' if windows else 'tokugawa-dosgolem')
    prefix = ['wine', str(executable)] if windows else [str(executable)]
    version = command(*prefix, '-version', stdout=subprocess.PIPE, stderr=subprocess.PIPE).stdout.decode().strip()
    assert version == manifest['version'] + ' ' + manifest['commit'], version
    state_arg = 'Z:' + str(state).replace('/', '\\') if windows else str(state)
    cmd = prefix + ['-diagnostics', '-state-dir', state_arg, '-desktop-audio=false', '-desktop-speed', '4',
                    '-publisher-speed', '4', '-steps', '200000000']
    with (root / 'receipt.json').open('wb') as out, (root / 'stderr.log').open('wb') as log:
        process = subprocess.Popen(cmd, env=env, stdout=out, stderr=log)
        children.append(process)
        initial = wait(lambda s: s.get('program') == 'mark' and s.get('startup_pending'))
        window = find_window()
        assert initial['step'] == initial['mouse_events'] == 0
        key('Tab', lambda s: s.get('startup_pending'))
        key('Return', lambda s: not s.get('startup_pending', True))
        wait(lambda s: s.get('program') == 'game' and s.get('step', 0) > 80000000)
        window = find_window()
        paused = key('F7', lambda s: s.get('paused'))
        assert paused['enabled'] and not paused['startup_pending']
        hd = capture('hd-title')
        key('Escape', lambda s: s.get('bar_visible'))
        click(150, 32, lambda s: s.get('bar_menu') == 0)
        click(150, 88, lambda s: s.get('bar_menu') == -1 and not s.get('enabled', True))
        original = capture('original-title')
        click(150, 32, lambda s: s.get('bar_menu') == 0)
        click(150, 136, lambda s: s.get('bar_menu') == -1 and s.get('enabled'))
        restored = capture('restored-hd-title')
        assert hd['proof'] == original['proof'] == restored['proof']
        assert hd['pixels'] == restored['pixels'] and hd['pixels'] != original['pixels']
        final_status = status()
        assert all(e['guest_unchanged'] for e in final_status['events'] if e['kind'] == 'toggle')
        close_window()
        assert process.wait(timeout=45) == 0
    receipt = json.loads((root / 'receipt.json').read_text())
    assert receipt['stop'] == 'desktop_closed' and receipt['build_version'] == manifest['version']
    mark = receipt['publisher_prelude']['receipt']
    assert mark['stop'] == 'dos_exit' and mark['exit_code'] == 0 and mark['successful_steps'] == 15989208
    assert receipt['publisher_prelude']['next_display'] == 'hd'
    assert receipt['initial_effective_DAT_sha256'] == sha(package / 'game/DX5.DAT')
    for name, expected in manifest['files'].items():
        assert sha(package / name) == expected['sha256'], name
    summary = {'result': 'PASS', 'platform': args.platform, 'runtime': 'Wine' if windows else 'Linux X11',
               'version': manifest['version'], 'commit': manifest['commit'], 'archive_sha256': sha(args.archive),
               'files_verified': len(manifest['files']), 'startup_choice_step_zero': True,
               'MARK_natural_exit_steps': mark['successful_steps'], 'MARK_HD_inherited': True,
               'ordinary_title_steps': receipt['successful_steps'], 'bar_toggles_guest_unchanged': True,
               'actual_X11_equals_presented': True, 'HD_original_differ': True, 'HD_restore_equal': True,
               'original_data_unchanged': True, 'close_normal': True, 'frames': [hd, original, restored],
               'native_Windows_verified': False, 'native_macOS_verified': False, 'normal_playthrough_required': False}
    (root / 'verification-summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({k: v for k, v in summary.items() if k != 'frames'}, ensure_ascii=False), flush=True)
finally:
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try:
                child.wait(timeout=10)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait(timeout=5)
    if windows:
        subprocess.run(['wineserver', '-k'], env=env, timeout=10, check=False)
