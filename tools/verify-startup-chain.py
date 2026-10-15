"""預設桌面啟動鏈、最後模式沿用、實際視窗及關窗驗證。只在固定 GUI Docker 執行。"""
import argparse
import ctypes
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time
from PIL import Image

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--mode', choices=['original', 'hd', 'cancel'], required=True)
mode = p.parse_args().mode
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000
env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
children = []


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def run(*args, **kw):
    return subprocess.run(args, env=env, check=True, timeout=15, **kw)


def status():
    try:
        return read(root / 'desktop-status.json')
    except FileNotFoundError:
        return {}


def wait(test, limit=30):
    end = time.monotonic() + limit
    while time.monotonic() < end:
        s = status()
        if test(s):
            return s
        assert process.poll() is None, (process.returncode, (root / 'stderr.log').read_text()[-3000:])
        time.sleep(.05)
    raise TimeoutError(str(status())[:2500])


def key(name, test):
    run('xdotool', 'keydown', '--clearmodifiers', name)
    try:
        s = wait(test)
    finally:
        run('xdotool', 'keyup', name)
    time.sleep(.15)
    return s


def click(x, y, test):
    run('xdotool', 'mousemove', '--window', window, str(x), str(y))
    run('xdotool', 'mousedown', '1')
    try:
        s = wait(test)
    finally:
        run('xdotool', 'mouseup', '1')
    time.sleep(.15)
    return s


def screenshot(label):
    geometry = run('xdotool', 'getwindowgeometry', '--shell', window, stdout=subprocess.PIPE).stdout.decode()
    g = dict(line.split('=', 1) for line in geometry.splitlines())
    path = root / (label + '.png')
    run('ffmpeg', '-v', 'error', '-threads', '1', '-filter_threads', '1', '-f', 'x11grab', '-draw_mouse', '0',
        '-video_size', g['WIDTH'] + 'x' + g['HEIGHT'], '-i', ':98+' + g['X'] + ',' + g['Y'], '-frames:v', '1', str(path))
    return path


def capture(label):
    count = len(status().get('events') or [])
    s = key('F9', lambda s: len(s.get('events') or []) > count)
    event = s['events'][-1]
    assert event['kind'] == 'capture' and event['paused']
    path = screenshot(label)
    actual = Image.open(path).convert('RGBA')
    presented = Image.open(event['frame']['path']).convert('RGBA')
    height = s['canvas_height']
    if s['bar_visible']:
        actual = actual.crop((0, 64, 1280, height + 64))
    assert actual.size == presented.size == (1280, height)
    assert actual.tobytes() == presented.tobytes()
    return dict(event=event, actual=str(path), sha256=sha(path), window_pixels_equal=True)


def display(enabled):
    click(150, 32, lambda s: s['bar_menu'] == 0)
    click(150, 88 + 48 * int(enabled), lambda s: s['bar_menu'] == -1 and s['enabled'] == enabled)


def find_window():
    result = run('xdotool', 'search', '--onlyvisible', '--name', '德川家康', stdout=subprocess.PIPE).stdout.decode().splitlines()
    assert len(result) == 1, result
    run('xdotool', 'windowfocus', '--sync', result[0])
    return result[0]


def close_window():
    # Send the window manager's normal close request, without destroying X resources.
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
    connection = x.XOpenDisplay(b':98')
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
    with (root / 'xvfb.log').open('wb') as log:
        xvfb = subprocess.Popen(['Xvfb', ':98', '-screen', '0', '1600x1200x24', '-nolisten', 'tcp', '-ac'],
                                env=env, stdout=log, stderr=subprocess.STDOUT)
    children.append(xvfb)
    end = time.monotonic() + 10
    while not Path('/tmp/.X11-unix/X98').exists():
        assert xvfb.poll() is None and time.monotonic() < end
        time.sleep(.05)
    run('xdotool', 'mousemove', '700', '500')
    command = [str(root / 'tokugawa-desktop-prototype'), '-desktop', '-desktop-speed', '60', '-publisher-speed', '4',
               '-steps', '750000000', '-mouse-file', '/project/tools/player-path.json']
    with (root / 'receipt.json').open('wb') as out, (root / 'stderr.log').open('wb') as err:
        process = subprocess.Popen(command, env=env, stdout=out, stderr=err)
    children.append(process)
    wait(lambda s: s.get('program') == 'mark' and s.get('startup_pending'), 90)
    window = find_window()
    time.sleep(.5)
    assert status()['step'] == status()['mouse_events'] == 0
    initial_picture = screenshot('startup-choice')
    assert Image.open(initial_picture).convert('RGBA').tobytes() == Image.open(
        project / 'workplace/desktop-prototype/startup-choice/preview/startup-choice.png').convert('RGBA').tobytes()
    mark_captures, game_captures = [], []
    if mode == 'cancel':
        close_window()
        process.wait(timeout=60)
        assert process.returncode == 0
        receipt = read(root / 'receipt.json')
        child = receipt['publisher_prelude']['receipt']
        assert receipt['stop'] == 'desktop_closed' and receipt['game_started'] is False
        assert child['program'] == 'mark' and child['successful_steps'] == 0 and child['stop'] == 'desktop_closed'
        assert child['desktop']['startup_selection_events'] in [[], None]
        assert not child['mouse_input_events'] and not (root / 'startup-memory.bin').exists()
        assert not receipt['publisher_prelude']['continued']
    else:
        # Deliberately change the first selection in MARK, to verify the last setting is carried.
        if mode == 'hd':
            key('Return', lambda s: not s['startup_pending'])
        else:
            click(896, 416, lambda s: not s['startup_pending'] and s['startup_fence'])
        wait(lambda s: s.get('step', 0) >= 9700000 and s['canvas_height'] == 960, 180)
        key('F7', lambda s: s['paused'])
        assert status()['step'] < 11721794 and status()['enabled'] == (mode == 'original')
        key('Escape', lambda s: s['bar_visible'] and s['bar_pinned'])
        mark_captures.append(capture('mark-initial'))
        display(mode == 'hd')
        mark_captures.append(capture('mark-last'))
        a, b = [Image.open(c['event']['frame']['path']).convert('RGBA') for c in mark_captures]
        assert a.tobytes() != b.tobytes()
        assert a.crop((0, 898, 1280, 960)).tobytes() == b.crop((0, 898, 1280, 960)).tobytes()
        assert mark_captures[0]['event']['proof'] == mark_captures[1]['event']['proof']
        key('Escape', lambda s: not s['bar_visible'])
        key('F7', lambda s: not s['paused'])
        wait(lambda s: s.get('program') == 'game', 180)
        window = find_window()
        first_game = status()
        assert not first_game['startup_pending'] and not first_game['startup_events']
        assert first_game['startup_selection'] == mode and first_game['enabled'] == (mode == 'hd')
        (root / 'first-game-status.json').write_text(json.dumps(first_game, ensure_ascii=False, indent=2) + '\n')
        wait(lambda s: s.get('step', 0) >= 740000000 and len(s.get('terrain_history') or []) == 134, 660)
        key('F7', lambda s: s['paused'])
        key('Escape', lambda s: s['bar_visible'] and s['bar_pinned'])
        assert status()['enabled'] == (mode == 'hd')
        game_captures.append(capture('game-carried'))
        display(mode != 'hd')
        game_captures.append(capture('game-other'))
        display(mode == 'hd')
        game_captures.append(capture('game-restored'))
        assert game_captures[0]['event']['proof'] == game_captures[1]['event']['proof'] == game_captures[2]['event']['proof']
        assert game_captures[0]['event']['frame']['rgba_sha256'] == game_captures[2]['event']['frame']['rgba_sha256']
        assert game_captures[0]['event']['frame']['rgba_sha256'] != game_captures[1]['event']['frame']['rgba_sha256']
        key('Escape', lambda s: not s['bar_visible'])
        key('F7', lambda s: not s['paused'])
        process.wait(timeout=90)
        assert process.returncode == 0
        receipt = read(root / 'receipt.json')
        prelude, child = receipt['publisher_prelude'], receipt['publisher_prelude']['receipt']
        assert prelude['continued'] and prelude['next_display'] == mode and receipt['game_started']
        assert sha(Path(prelude['receipt_path'])) == prelude['receipt_sha256']
        assert read(Path(prelude['receipt_path'])) == child
        baseline = read(project / 'workplace/desktop-prototype/headless/startup-probe/latest.json')
        fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip', 'trace_tail',
                  'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame', 'cd_pcm', 'mounted_disc',
                  'mouse_input_events', 'input_boundary_frames', 'real_mode_history', 'real_mode_last']
        assert all(receipt[k] == baseline[k] for k in fields), [k for k in fields if receipt[k] != baseline[k]]
        native_mark = read(project / 'workplace/hd-prototype/catalog/remaining/consumers/mark-probe/r8/startup-probe/receipt-75m.json')
        mark_fields = [k for k in fields if k not in ['mouse_input_events', 'input_boundary_frames']]
        assert all(child[k] == native_mark[k] for k in mark_fields), [k for k in mark_fields if child[k] != native_mark[k]]
        assert child['successful_steps'] == 15989208 and child['exit_code'] == child['fd2_startup_calls'] == 0
        assert all(e['buttons'] == 0 for e in child['mouse_input_events'])
        assert sha(root / 'mark-startup-memory.bin') == '62da54cc0cef42e9a55c64cb1ac958b05b851f19c12247023f6feddd023fa085'
        assert sha(root / 'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
        assert receipt['desktop']['startup_selection'] == mode and not receipt['desktop']['startup_selection_events']
        assert all(e['guest_unchanged'] for r in [child, receipt] for e in r['desktop']['events'] if e['kind'] == 'toggle')
    sources = read(root / 'build-sources.json')
    assert len(sources) == 103 and child['integration_input_sha256'] == sources
    if mode != 'cancel':
        assert receipt['integration_input_sha256'] == sources
    assert all(sha(project / n) == v for n, v in sources.items())
    report = dict(result='PASS', mode=mode, binary_sha256=sha(root / 'tokugawa-desktop-prototype'),
                  source_count=len(sources), receipt_sha256=sha(root / 'receipt.json'), verifier_sha256=sha(Path(__file__)),
                  startup_picture=dict(path=str(initial_picture), sha256=sha(initial_picture)),
                  mark_captures=mark_captures, game_captures=game_captures,
                  scope='default desktop startup selection, original MARK, last display setting, DX5 first battle, or normal window close')
    (root / 'verification.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print('PASS startup chain', mode)
finally:
    for child_process in reversed(children):
        if child_process.poll() is None:
            child_process.terminate()
            try:
                child_process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child_process.kill()
                child_process.wait(timeout=5)
