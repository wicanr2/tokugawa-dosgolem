"""真實 X11 的 MARK 顯示、HD／缺圖、功能列及自然退出驗證。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time
from PIL import Image

assert Path('/.dockerenv').exists() and os.getuid()==1000
p=argparse.ArgumentParser();p.add_argument('--mode',choices=['hd','missing'],required=True)
mode=p.parse_args().mode
root=Path('/output');assert root.stat().st_uid==1000
env=dict(os.environ,DISPLAY=':98',LIBGL_ALWAYS_SOFTWARE='1',LP_NUM_THREADS='2')
children=[]

def run(*args,**kw):return subprocess.run(args,env=env,check=True,timeout=15,**kw)
def status():
    try:return json.loads((root/'desktop-status.json').read_text())
    except FileNotFoundError:return {}
def wait(test,limit=30):
    end=time.monotonic()+limit
    while time.monotonic()<end:
        s=status()
        if test(s):return s
        assert process.poll() is None,(process.returncode,(root/'stderr.log').read_text()[-2000:])
        time.sleep(.05)
    raise TimeoutError(str(status())[:3000])
def key(name,test):
    run('xdotool','keydown','--clearmodifiers',name)
    try:s=wait(test)
    finally:run('xdotool','keyup',name)
    time.sleep(.15);return s
def click(x,y,test):
    run('xdotool','mousemove','--window',window,str(x),str(y));run('xdotool','mousedown','1')
    try:s=wait(test)
    finally:run('xdotool','mouseup','1')
    time.sleep(.15);return s
def capture(label):
    count=len(status().get('events') or [])
    s=key('F9',lambda s:len(s.get('events') or [])>count)
    event=s['events'][-1];assert event['kind']=='capture' and event['paused']
    geometry=run('xdotool','getwindowgeometry','--shell',window,stdout=subprocess.PIPE).stdout.decode()
    g=dict(line.split('=',1) for line in geometry.splitlines())
    path=root/(label+'.png')
    run('ffmpeg','-v','error','-threads','1','-filter_threads','1','-f','x11grab','-draw_mouse','0','-video_size',g['WIDTH']+'x'+g['HEIGHT'],
        '-i',':98+'+g['X']+','+g['Y'],'-frames:v','1',str(path))
    actual=Image.open(path).convert('RGBA');presented=Image.open(event['frame']['path']).convert('RGBA')
    if s['bar_visible']:actual=actual.crop((0,64,1280,1024))
    assert actual.size==presented.size==(1280,960) and actual.tobytes()==presented.tobytes()
    return {'event':event,'actual':str(path),'sha256':hashlib.sha256(path.read_bytes()).hexdigest(),'window_pixels_equal':True}
def display(row):
    click(150,32,lambda s:s['bar_menu']==0)
    click(150,88+48*row,lambda s:s['bar_menu']==-1 and s['enabled']==(row==1))

try:
    with (root/'xvfb.log').open('wb') as log:
        xvfb=subprocess.Popen(['Xvfb',':98','-screen','0','1600x1200x24','-nolisten','tcp','-ac'],env=env,stdout=log,stderr=subprocess.STDOUT)
    children.append(xvfb);end=time.monotonic()+10
    while not Path('/tmp/.X11-unix/X98').exists():
        assert xvfb.poll() is None and time.monotonic()<end;time.sleep(.05)
    run('xdotool','mousemove','700','500')
    command=[str(root/'tokugawa-desktop-prototype'),'-program','mark','-desktop','-desktop-speed','4','-steps','75000000']
    if mode=='missing':command+=['-hd-publisher-asset','/output/no-publisher.png']
    with (root/'receipt.json').open('wb') as out,(root/'stderr.log').open('wb') as err:
        process=subprocess.Popen(command,env=env,stdout=out,stderr=err)
    children.append(process)
    wait(lambda s:s.get('startup_pending') and s.get('program')=='mark',60)
    window=run('xdotool','search','--name','德川家康',stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
    run('xdotool','windowfocus','--sync',window)
    time.sleep(.5);assert status()['step']==status()['mouse_events']==0
    if mode=='hd':key('Return',lambda s:not s['startup_pending'])
    else:click(896,416,lambda s:not s['startup_pending'] and s['startup_fence'])
    wait(lambda s:s['step']>=9700000 and s['canvas_height']==960,160)
    key('F7',lambda s:s['paused'])
    assert status()['step']<11721794,'missed natural visible publisher interval'
    run('xdotool','mousemove','--window',window,'700','500');time.sleep(.2)
    key('Escape',lambda s:s['bar_visible'] and s['bar_pinned'] and s['window_size']==[1280,1024])
    frozen=status()['step'];captures=[]
    if mode=='hd':
        for row in [0,1,0]:
            display(row);captures.append(capture('publisher-'+str(row)+'-'+str(len(captures))))
        assert captures[1]['event']['frame']['mode']=='hd'
        a,b,c=[Image.open(x['event']['frame']['path']).convert('RGBA') for x in captures]
        assert a.tobytes()==c.tobytes() and a.tobytes()!=b.tobytes()
        assert a.crop((0,898,1280,960)).tobytes()==b.crop((0,898,1280,960)).tobytes()
    else:
        captures.append(capture('publisher-missing-0'))
        for enabled in [False,True]:
            key('F6',lambda s:s['enabled']==enabled);captures.append(capture('publisher-missing-'+str(len(captures))))
        assert len({x['event']['frame']['rgba_sha256'] for x in captures})==1
    assert all(x['event']['proof']==captures[0]['event']['proof'] for x in captures)
    assert status()['step']==frozen
    assert all(e['guest_unchanged'] for e in status()['events'] if e['kind']=='toggle')
    key('Escape',lambda s:not s['bar_visible'])
    key('F7',lambda s:not s['paused'])
    process.wait(timeout=180);assert process.returncode==0
    receipt=json.loads((root/'receipt.json').read_text())
    baseline=json.loads(Path('/project/workplace/hd-prototype/catalog/remaining/consumers/mark-probe/r8/startup-probe/receipt-75m.json').read_text())
    fields=['successful_steps','instruction_budget','stop','stop_instruction_linear_eip','trace_tail','interrupt_counts','dpmi_calls','dpmi_unimplemented',
            'frame','cd_pcm','mounted_disc','real_mode_history','real_mode_last']
    differences=[k for k in fields if receipt[k]!=baseline[k]]
    assert not differences,differences
    assert receipt['input_file']=='MARK.EXE.BAK' and receipt['exit_code']==receipt['fd2_startup_calls']==0
    assert all(e['buttons']==0 for e in receipt['mouse_input_events'])
    ram=hashlib.sha256((root/'mark-startup-memory.bin').read_bytes()).hexdigest()
    oldram=hashlib.sha256(Path('/project/workplace/hd-prototype/catalog/remaining/consumers/mark-probe/r8/startup-memory.bin').read_bytes()).hexdigest()
    assert ram==oldram
    report={'result':'PASS','mode':mode,'binary_sha256':hashlib.sha256((root/'tokugawa-desktop-prototype').read_bytes()).hexdigest(),
            'verifier_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),'ram_sha256':ram,'captures':captures,'fields_equal':fields,
            'receipt_sha256':hashlib.sha256((root/'receipt.json').read_bytes()).hexdigest(),'scope':'standalone native MARK desktop; full game launcher still separate'}
    (root/'verification.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print('PASS publisher',mode)
finally:
    for child in reversed(children):
        if child.poll() is None:
            child.terminate()
            try:child.wait(timeout=5)
            except subprocess.TimeoutExpired:child.kill();child.wait(timeout=5)
