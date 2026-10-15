"""真實 X11 啟動選擇、確認手勢隔離與功能列動態切換。"""
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
p.add_argument('--mode', choices=['original', 'hd'], required=True)
p.add_argument('--startup-only', action='store_true')
args = p.parse_args()
mode = args.mode
root = Path('/output')
assert root.stat().st_uid == 1000
env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
children = []

def run(*args, **kw):
    return subprocess.run(args, env=env, check=True, timeout=15, **kw)

def status():
    try: return json.loads((root/'desktop-status.json').read_text())
    except FileNotFoundError: return {}

def wait(test, limit=30):
    end = time.monotonic()+limit
    while time.monotonic()<end:
        s=status()
        if test(s): return s
        assert process.poll() is None, (process.returncode, (root/'stderr.log').read_text()[-2000:])
        time.sleep(.05)
    raise TimeoutError(status())

def key(key, test):
    run('xdotool','keydown','--clearmodifiers',key)
    try: s=wait(test)
    finally: run('xdotool','keyup',key)
    time.sleep(.2)
    return s

def click(x,y,test):
    run('xdotool','mousemove','--window',window,str(x),str(y))
    run('xdotool','mousedown','1')
    try: s=wait(test)
    finally: run('xdotool','mouseup','1')
    time.sleep(.2)
    return s

def capture():
    count=len(status().get('events') or [])
    s=key('F9',lambda s:len(s.get('events') or [])>count)
    return s['events'][-1]

def screenshot(name):
    geometry=run('xdotool','getwindowgeometry','--shell',window,stdout=subprocess.PIPE).stdout.decode()
    g=dict(line.split('=',1) for line in geometry.splitlines())
    path=root/(name+'.png')
    for attempt in range(20):
        candidate = path if name!='startup-choice' else root/f'startup-attempt-{attempt}.png'
        run('ffmpeg','-v','error','-threads','1','-filter_threads','1','-f','x11grab','-draw_mouse','0',
            '-video_size',g['WIDTH']+'x'+g['HEIGHT'],'-i',':98+'+g['X']+','+g['Y'],'-frames:v','1',str(candidate))
        if name!='startup-choice': break
        im=Image.open(candidate).convert('RGB')
        if im.getpixel((0,0))==(24,29,38) and len(im.getcolors(im.width*im.height) or [])>=5:
            candidate.rename(path)
            break
        time.sleep(.2)
    else: raise AssertionError('startup screen was not presented in X11')
    return str(path)

try:
    with (root/'xvfb.log').open('wb') as log:
        xvfb=subprocess.Popen(['Xvfb',':98','-screen','0','1600x1000x24','-nolisten','tcp','-ac'],env=env,stdout=log,stderr=subprocess.STDOUT)
    children.append(xvfb)
    end=time.monotonic()+10
    while not Path('/tmp/.X11-unix/X98').exists():
        assert xvfb.poll() is None and time.monotonic()<end
        time.sleep(.05)
    run('xdotool','mousemove','700','500')
    with (root/'receipt.json').open('wb') as out, (root/'stderr.log').open('wb') as err:
        process=subprocess.Popen([str(root/'tokugawa-desktop-prototype'),'-desktop','-desktop-speed','30','-steps','200000000'],env=env,stdout=out,stderr=err)
    children.append(process)
    s=wait(lambda s:s.get('startup_pending') and s.get('mode')=='startup_choice',60)
    window=run('xdotool','search','--name','德川家康',stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
    run('xdotool','windowfocus','--sync',window)
    choice_picture=screenshot('startup-choice')
    time.sleep(1)
    assert status()['step']==0 and status()['mouse_events']==0 and not status().get('events')
    if args.startup_only:
        (root/'startup-preview.json').write_text(json.dumps({'result':'PASS','classification':'startup window only; game flow separately verified',
            'picture':choice_picture,'png_sha256':hashlib.sha256(Path(choice_picture).read_bytes()).hexdigest(),
            'binary_sha256':hashlib.sha256((root/'tokugawa-desktop-prototype').read_bytes()).hexdigest(),
            'verifier_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),'step':status()['step'],'mouse_events':status()['mouse_events']},indent=2)+'\n')
        print('PASS startup preview')
        raise SystemExit(0)
    run('xdotool','mousemove','--window',window,'700','500')
    if mode=='original':
        s=key('Return',lambda s:not s['startup_pending'])
    else:
        run('xdotool','mousemove','--window',window,'896','416')
        run('xdotool','mousedown','1')
        s=wait(lambda s:not s['startup_pending'] and s['startup_fence'])
        run('xdotool','mousemove','--window',window,'600','342')
        time.sleep(.5)
        assert status()['step']==0 and status()['mouse_events']==0 and status()['startup_fence']
        run('xdotool','mousemove','--window',window,'700','500')
        run('xdotool','mouseup','1')
    assert s['startup_selection']==mode and s['enabled']==(mode=='hd')
    e=s['startup_events'][0]
    assert e['guest_unchanged'] and e['before']==e['after'] and e['before']['step']==0
    wait(lambda s:s['step']>=80_000_000,180)
    key('F7',lambda s:s['paused'])
    # Move away from the reveal strip, then pin the bar with ordinary Escape.
    run('xdotool','mousemove','--window',window,'700','500')
    time.sleep(.2)
    key('Escape',lambda s:s['bar_visible'] and s['bar_pinned'])
    frozen=status()['step']
    captures=[]
    for row in [0,1,0]:
        click(150,32,lambda s:s['bar_menu']==0)
        click(150,88+48*row,lambda s:s['bar_menu']==-1 and s['enabled']==(row==1))
        captures.append(capture())
        assert status()['step']==frozen
    assert all(c['proof']==captures[0]['proof'] for c in captures)
    assert captures[0]['frame']['rgba_sha256']==captures[2]['frame']['rgba_sha256']
    assert captures[0]['frame']['rgba_sha256']!=captures[1]['frame']['rgba_sha256']
    assert all(e['guest_unchanged'] for e in status()['bar_events'] if e['kind']=='option')
    picture=screenshot('toolbar-after-startup')
    key('Escape',lambda s:not s['bar_visible'])
    key('F7',lambda s:not s['paused'])
    process.wait(timeout=180)
    assert process.returncode==0
    receipt=json.loads((root/'receipt.json').read_text())
    assert receipt['successful_steps']==200_000_000 and receipt['stop']=='instruction_budget'
    assert all(e['buttons']==0 for e in receipt['mouse_input_events']) and receipt['fd2_startup_calls']==0
    report={'result':'PASS','selection':mode,'startup_event':e,'captures':captures,'pictures':[choice_picture,picture],
            'verifier_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
            'binary_sha256':hashlib.sha256((root/'tokugawa-desktop-prototype').read_bytes()).hexdigest(),
            'ram_sha256':hashlib.sha256((root/'startup-memory.bin').read_bytes()).hexdigest(),
            'receipt_sha256':hashlib.sha256((root/'receipt.json').read_bytes()).hexdigest()}
    (root/'verification.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print('PASS',mode)
finally:
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try: child.wait(timeout=5)
            except subprocess.TimeoutExpired: child.kill(); child.wait(timeout=5)
