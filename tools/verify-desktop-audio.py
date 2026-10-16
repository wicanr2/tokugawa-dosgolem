"""在既有 PulseAudio Docker image 驗證實際桌面輸出及原版狀態不變。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import struct
import subprocess
import time

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--binary', type=Path, required=True)
p.add_argument('--resource-dir', required=True)
p.add_argument('--game-dir', required=True)
p.add_argument('--cpu-source', required=True)
p.add_argument('--verify-only', action='store_true', help='核對已保存的同一執行收據，不重新播放')
args = p.parse_args()
root = Path('/output')
assert root.stat().st_uid == 1000
result = root / 'audio-device'
result.mkdir(exist_ok=args.verify_only)
runtime = result / 'pulse-runtime'
runtime.mkdir(mode=0o700, exist_ok=args.verify_only)
env = dict(os.environ, DISPLAY=':93', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2',
           XDG_RUNTIME_DIR=str(runtime), XDG_CONFIG_HOME=str(result / 'config'),
           PULSE_SERVER='unix:' + str(runtime / 'socket'))
config = result / 'pulse.pa'
config.write_text('load-module module-null-sink sink_name=tokugawa rate=44100 channels=2\n'
                  f'load-module module-native-protocol-unix socket={runtime}/socket auth-anonymous=1\n'
                  'set-default-sink tokugawa\n')
children = []


def launch(command, label, output=None):
    log = (result / (label + '.log')).open('wb')
    proc = subprocess.Popen(command, env=env, stdout=output or log, stderr=log)
    children.append((proc, log))
    return proc


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


try:
    if args.verify_only:
        receipts = {label: json.loads((result / (label + ".json")).read_text()) for label in ("disabled", "enabled")}
    else:
        xvfb = launch(['Xvfb', ':93', '-screen', '0', '1600x1200x24', '-nolisten', 'tcp', '-ac'], 'xvfb')
        pulse = launch(['pulseaudio', '-n', '--daemonize=no', '--use-pid-file=no', '--exit-idle-time=-1', '-F', str(config)], 'pulse')
        limit = time.monotonic() + 15
        while not (runtime / 'socket').exists() or not Path('/tmp/.X11-unix/X93').exists():
            assert pulse.poll() is None and xvfb.poll() is None and time.monotonic() < limit
            time.sleep(.05)
        empty = result / 'empty-mouse.json'
        empty.write_text('[]\n')
        receipts = {}
        with (result / 'monitor.pcm').open('wb') as raw:
            monitor = launch(['parec', '--device=tokugawa.monitor', '--format=s16le', '--rate=44100', '--channels=2'], 'monitor', raw)
            for enabled in (False, True):
                label = 'enabled' if enabled else 'disabled'
                state = result / label
                state.mkdir()
                state_mouse = state / 'empty-mouse.json'
                # First original PLAY AUDIO is at 136777435 in this established
                # ordinary-input path. Keep only events within this sample.
                inputs = json.loads(Path('/project/tools/player-path.json').read_text())
                state_mouse.write_text(json.dumps([event for event in inputs if event['step'] < 150000000]))
                command = [str(args.binary), '-play', '-desktop=' + str(enabled).lower(), '-desktop-audio=' + str(enabled).lower(),
                           '-publisher-intro=false', '-startup-display', 'hd', '-steps', '150000000', '-desktop-speed', '60', '-diagnostics',
                           '-resource-dir', args.resource_dir, '-game-dir', args.game_dir, '-state-dir', str(state),
                           '-cpu-source', args.cpu_source, '-mouse-file', str(state_mouse)]
                with (result / (label + '.json')).open('wb') as receipt:
                    process = launch(command, label, receipt)
                assert process.wait(timeout=300) == 0, label
                receipts[label] = json.loads((result / (label + '.json')).read_text())
            time.sleep(.4)
            monitor.terminate()
            monitor.wait(timeout=10)
    a, b = receipts['disabled'], receipts['enabled']
    state_fields = ['successful_steps', 'stop', 'stop_instruction_linear_eip', 'interrupt_counts', 'interrupts_tail',
                    'dpmi_calls', 'dpmi_unimplemented', 'real_mode_history', 'real_mode_last', 'trace_tail',
                    'mouse_cursor', 'mouse_calls', 'mouse_input_events', 'virtual_clock']
    assert a['successful_steps'] == 150000000
    differences = [k for k in state_fields if a[k] != b[k]]
    assert not differences, differences
    assert {k: v for k, v in a['cd_pcm'].items() if k != 'path'} == {k: v for k, v in b['cd_pcm'].items() if k != 'path'}
    ram_paths = [result / 'disabled/startup-memory.bin']
    other_ram = [result / 'enabled/startup-memory.bin']
    assert len(ram_paths) == len(other_ram) == 1, (ram_paths, other_ram)
    assert sha(ram_paths[0]) == sha(other_ram[0])
    data = (result / 'monitor.pcm').read_bytes()
    assert len(data) % 4 == 0
    peak = max((abs(x[0]) for x in struct.iter_unpack('<h', data)), default=0)
    assert peak > 100, peak
    device = b['desktop']['audio_output']
    assert device['enabled'] and device['device_position_ns'] > 0
    summary = {'result': 'PASS', 'scope': 'PulseAudio null-sink loopback; technical playback only',
               'steps': 150000000, 'desktop_speed': 60, 'equal_original_fields': state_fields, 'complete_RAM_equal': True,
               'RAM_sha256': sha(ram_paths[0]), 'monitor_bytes': len(data), 'monitor_peak_s16': peak,
               'monitor_sha256': sha(result / 'monitor.pcm'), 'device': device, 'listening_verified': False}
    (result / 'verification-summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(summary, ensure_ascii=False), flush=True)
finally:
    for process, log in reversed(children):
        if process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
        log.close()
