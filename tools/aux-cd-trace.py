#!/usr/bin/env python3
"""唯讀查證原版 DOS/4GW 的 CD 配置；輸出僅作 DOSBox-X 輔助證據。"""
import os
import hashlib
import pathlib
import pty
import json
import re
import select
import shutil
import signal
import struct
import subprocess
import termios
import time
import tty

assert pathlib.Path('/.dockerenv').exists()
expected = {
    'DX5.EXE': '2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a',
    'DOS4GW.EXE': 'b8265123ac8a189637448618409ef3ecd2e9f3e1a47062c685a02240f688dec1',
    'DX5.cue': 'd9c007c1ab565cf86303e10e05a4d701b7acf501273d79818f10acc7f1943083',
    'DX5.bin': '3f511d4c2fb819f950d881697f4e771d37ccfa2b0fd9eca6c4bb48fff6be2448',
}
for name, digest in expected.items():
    assert hashlib.file_digest(open('/original/' + name, 'rb'), 'sha256').hexdigest() == digest, name
out = pathlib.Path('/output/aux-debug')
out.mkdir(exist_ok=True)
assert out.stat().st_uid == os.getuid() == 1000
for path in out.rglob('*'):
    assert path.is_symlink() or path.stat().st_uid == 1000, path
game = out / 'game'
game.mkdir(exist_ok=True)
for src in pathlib.Path('/original').iterdir():
    if src.is_file() and src.suffix.lower() not in {'.bin', '.cue'}:
        dst = game / src.name
        if dst.exists():
            assert dst.stat().st_uid == 1000
            dst.chmod(0o644)
        shutil.copyfile(src, dst)
(game / 'DX5.cue').write_bytes(pathlib.Path('/original/DX5.cue').read_bytes())
link = game / 'DX5.BIN'
if not link.exists():
    link.symlink_to('/original/DX5.bin')
conf = out / 'probe.conf'
conf.write_text('''[sdl]
output=surface
fullscreen=false
[dosbox]
machine=svga_s3
memsize=16
[cpu]
core=normal
cycles=fixed 30000
[mixer]
nosound=true
[log]
logfile=/output/aux-debug/debug.log
debuggerrun=debugger
[autoexec]
mount c /output/aux-debug/game
c:
imgmount d DX5.cue -t iso
DX5.EXE
''')
env = dict(os.environ, DISPLAY=':93', SDL_AUDIODRIVER='dummy', TERM='xterm', HOME='/tmp', ESCDELAY='25')
xlog = (out / 'xvfb.log').open('wb')
xvfb = subprocess.Popen(['Xvfb', ':93', '-screen', '0', '1024x768x24'], stdout=xlog, stderr=subprocess.STDOUT)
master, slave = pty.openpty()
tty.setraw(slave)
fcntl = __import__('fcntl')
fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', 60, 180, 0, 0))
log = (out / 'terminal.raw').open('wb')
proc = None

def stop(signum, frame):
    raise SystemExit(128 + signum)

signal.signal(signal.SIGTERM, stop)
signal.signal(signal.SIGINT, stop)

def drain(seconds):
    stop = time.monotonic() + seconds
    data = bytearray()
    while time.monotonic() < stop:
        ready, _, _ = select.select([master], [], [], min(0.1, max(0, stop-time.monotonic())))
        if ready:
            try:
                chunk = os.read(master, 65536)
            except OSError:
                break
            log.write(chunk)
            log.flush()
            data.extend(chunk)
    return bytes(data)

def command(text, wait=0.4):
    for char in text.encode('ascii'):
        os.write(master, bytes([char]))
        drain(0.1)
    os.write(master, b'\n')
    return drain(wait)

def ev(label):
    command('EV EAX EBX ECX EDX CS EIP DS ES EDI', 0.5)
    text = (out / 'debug.log').read_text(errors='replace')
    rows = re.findall(r"EV of '[^']+' is:\n([^\n]+)", text)
    values = [int(x, 16) for x in rows[-1].split()]
    return dict(label=label, **dict(zip(['EAX', 'EBX', 'ECX', 'EDX', 'CS', 'EIP', 'DS', 'ES', 'EDI'], values)))

def key(seq):
    os.write(master, seq)
    drain(1)

def dump(address, filename):
    path = out / 'MEMDUMP.BIN'
    if path.exists():
        path.unlink()
    command('MEMDUMPBIN ' + address + ' 40', 0.5)
    if not path.exists():
        path = out / 'memdump.bin'
    assert path.exists(), address
    path.replace(out / filename)

try:
    time.sleep(0.5)
    proc = subprocess.Popen(['dosbox-x', '-conf', str(conf), '-break-start', '-fastlaunch', '-nomenu', '-nogui'], stdin=slave, stdout=slave, stderr=slave, env=env, cwd=out, start_new_session=True)
    os.close(slave)
    drain(2)
    command('BPINT 31 01 00')
    command('BPLIST')
    os.write(master, b'\x1b[15~')
    drain(5)
    rows = [ev('query-allocation-before')]
    assert rows[-1]['EAX'] == 0x100 and rows[-1]['EBX'] == 1
    key(b'\x1b[21~')
    rows.append(ev('query-allocation-after'))
    query_ax = rows[-1]['EAX']
    command('BPINT 31 03 00')
    key(b'\x1b[15~')
    rows.append(ev('request-allocation-before'))
    assert rows[-1]['EAX'] == 0x100 and rows[-1]['EBX'] == 2
    key(b'\x1b[21~')
    rows.append(ev('request-allocation-after'))
    request_ax = rows[-1]['EAX']
    key(b'\x1b[15~')
    rows.append(ev('simulate-int2f-before'))
    assert rows[-1]['EAX'] == 0x300 and rows[-1]['EBX'] == 0x2f
    dump('ES:EDI', 'real-packet-before.bin')
    dump('DS:' + format(query_ax, 'X'), 'query-at-segment-before.bin')
    dump('DS:' + format(request_ax, 'X'), 'request-at-segment-before.bin')
    dump('DS:' + format(query_ax << 4, 'X'), 'query-at-base-before.bin')
    key(b'\x1b[21~')
    rows.append(ev('simulate-int2f-after'))
    dump('ES:EDI', 'real-packet-after.bin')
    dump('DS:' + format(query_ax, 'X'), 'query-at-segment-after.bin')
    dump('DS:' + format(request_ax, 'X'), 'request-at-segment-after.bin')
    (out / 'registers.json').write_text(json.dumps(rows, indent=2))
    command('DOS MCBS', 0.5)
    print(json.dumps(rows, indent=2))
    print((out / 'debug.log').read_text(errors='replace')[-2500:])
finally:
    if proc is not None:
        os.killpg(proc.pid, signal.SIGTERM)
        try:
            proc.wait(timeout=3)
        except subprocess.TimeoutExpired:
            os.killpg(proc.pid, signal.SIGKILL)
            proc.wait()
    xvfb.terminate()
    xvfb.wait(timeout=3)
    log.close()
    xlog.close()
    outputs = {p.name: hashlib.file_digest(p.open('rb'), 'sha256').hexdigest()
               for p in out.iterdir() if p.is_file() and p.name != 'manifest.json'}
    manifest = {
        'classification': 'DOSBox-X auxiliary evidence; not a dosgolem receipt',
        'image': 'psychicwar-dosboxx@sha256:79a3e2df79738a5821a7a168d15e73894ad3ce7e7c6e2d4cdd53c09550316d46',
        'executable_sha256': hashlib.file_digest(open('/usr/local/bin/dosbox-x', 'rb'), 'sha256').hexdigest(),
        'script_sha256': hashlib.file_digest(open(__file__, 'rb'), 'sha256').hexdigest(),
        'input_sha256': expected,
        'output_sha256': outputs,
    }
    (out / 'manifest.json').write_text(json.dumps(manifest, indent=2))
