"""核對新來源註冊與正常標題；靜態測試不冒充其他來源的正常流程。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time

from PIL import Image


FIELDS=['successful_steps','instruction_budget','stop','stop_instruction_linear_eip','trace_tail',
        'interrupt_counts','dpmi_calls','dpmi_unimplemented','frame','cd_pcm','mounted_disc',
        'mouse_input_events','input_boundary_frames','real_mode_history','real_mode_last']


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def coordinates(rect):
    return [rect['Min']['X'],rect['Min']['Y'],rect['Max']['X'],rect['Max']['Y']]


def title(root,project,missing):
    mode='title-missing' if missing else 'title'
    assert root.stat().st_uid==1000
    (root/'startup-probe').mkdir(exist_ok=True)
    (root/'idle-mouse.json').write_text('[]')
    metadata=json.loads((project/'workplace/hd-prototype/catalog/remaining/runtime-profiles-v1.json').read_text())
    source=next(r for r in metadata['records'] if r['name']=='OP5.OZM')
    profiles=json.loads((project/'workplace/hd-prototype/catalog/remaining/profiles.json').read_text())
    art_record=next(r for r in profiles['records'] if r['name']=='OP5.OZM')
    command=[str(root/'tokugawa-desktop-prototype'),'-desktop', "-publisher-intro=false",'-desktop-speed','60','-steps','200000000',
             '-mouse-file',str(root/'idle-mouse.json'),'-hd-battle-frame-proof']
    if missing:
        folder=root/'missing-graphics'
        folder.mkdir()
        # 隔離整個原有圖目錄，只刪除新標題；其他既有及新來源維持可用。
        archive=project/'assets/hd-b/art'
        for path in archive.rglob('*.png'):
            relative=path.relative_to(archive)
            if str(relative)==source['asset']:continue
            target=folder/relative;target.parent.mkdir(parents=True,exist_ok=True)
            shutil.copy2(path,target)
        command+=['-hd-graphics-dir',str(folder)]
    env=dict(os.environ,DISPLAY=':98',LIBGL_ALWAYS_SOFTWARE='1',LP_NUM_THREADS='2')
    children=[]

    def run(*command,**kwargs):
        return subprocess.run(command,env=env,timeout=15,check=True,**kwargs)

    def state():
        try:return json.loads((root/'desktop-status.json').read_text())
        except (FileNotFoundError,json.JSONDecodeError):return {}

    def wait(predicate,seconds=30):
        end=time.monotonic()+seconds
        while time.monotonic()<end:
            current=state()
            if predicate(current):return current
            assert process.poll() is None,f'desktop exited early: {process.returncode}'
            time.sleep(.05)
        raise TimeoutError(f'title wait: step={state().get("step")}')

    def key(name):
        before=state()
        run('xdotool','keydown','--clearmodifiers',name)
        try:
            if name=='F6':wait(lambda s:s.get('enabled')!=before.get('enabled'),15)
            elif name=='F7':wait(lambda s:s.get('paused')!=before.get('paused'),15)
            else:wait(lambda s:len(s.get('events') or [])>len(before.get('events') or []),15)
        finally:run('xdotool','keyup',name)
        time.sleep(.15)

    def capture(label):
        key('F9');current=state();event=current['events'][-1]
        assert event['kind']=='capture' and event['paused']
        geometry=dict(line.split('=',1) for line in run('xdotool','getwindowgeometry','--shell',window,stdout=subprocess.PIPE).stdout.decode().splitlines())
        path=root/f'gui-{mode}-{label}.png'
        run('ffmpeg','-v','error','-y','-threads','1','-filter_threads','1','-f','x11grab',
            '-video_size',f'{geometry["WIDTH"]}x{geometry["HEIGHT"]}',
            '-i',f':98+{geometry["X"]},{geometry["Y"]}','-frames:v','1',str(path),stdout=subprocess.DEVNULL)
        assert Image.open(path).convert('RGBA').tobytes()==Image.open(event['frame']['path']).convert('RGBA').tobytes()
        return {'actual_png':path.name,'sha256':sha(path),'event':event,'window_pixels_equal':True,
                'graphics':current['graphics'],'windows':current['window_draws'],'window_modes':current['window_modes']}

    try:
        with (root/'xvfb.log').open('wb') as stdout:
            xvfb=subprocess.Popen(['Xvfb',':98','-screen','0','1600x1000x24','-nolisten','tcp','-ac'],env=env,stdout=stdout,stderr=subprocess.STDOUT)
        children.append(xvfb)
        end=time.monotonic()+10
        while not Path('/tmp/.X11-unix/X98').exists():
            assert xvfb.poll() is None and time.monotonic()<end
            time.sleep(.05)
        receipt_path=root/f'desktop-{mode}.json'
        with receipt_path.open('wb') as stdout,(root/'desktop.stderr.log').open('wb') as stderr:
            process=subprocess.Popen(command,env=env,stdout=stdout,stderr=stderr);children.append(process)
            wait(lambda s:s.get('step',0)>1000000,60)
            window=run('xdotool','search','--name','德川家康',stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
            run('xdotool','windowfocus','--sync',window)
            run('xdotool','mousemove','--window',window,'700','500')
            wait(lambda s:s.get('step',0)>=80000000,60)
            key('F7')
            original=capture('original');key('F6');enhanced=capture('enabled')
            key('F6');again=capture('original-again');key('F6');restored=capture('restored')
            pictures=[original,enhanced,again,restored]
            assert all(p['event']['proof']==original['event']['proof'] for p in pictures)
            assert original['event']['frame']['png_sha256']==again['event']['frame']['png_sha256']
            assert enhanced['event']['frame']['png_sha256']==restored['event']['frame']['png_sha256']
            windows=enhanced['windows'];assert len(windows)==1 and coordinates(windows[0]['logical_rect'])==[232,150,408,246]
            assert windows[0]['completed_step']==46797968 and enhanced['window_modes']==['hd']
            reports=[r for r in enhanced['graphics'] if r['resource']=='OP5.OZM'];assert len(reports)==1
            report=reports[0];expected_mode='fallback_missing_asset' if missing else 'hd'
            assert report['mode']==expected_mode,report
            paint=source['paint_rect'];window_rect=coordinates(windows[0]['logical_rect'])
            before=Image.open(root/original['actual_png']).convert('RGB')
            after=Image.open(root/enhanced['actual_png']).convert('RGB')
            art=Image.open(project/'assets/hd-b/art'/source['asset']).convert('RGB')
            checked=changed=0
            visible=[coordinates(r) for r in report.get('visible_rectangles',[])]
            for y in range(800):
                for x in range(1280):
                    nx,ny=x//2,y//2
                    if window_rect[0]<=nx<window_rect[2] and window_rect[1]<=ny<window_rect[3]:continue
                    inside=paint[0]<=nx<paint[2] and paint[1]<=ny<paint[3]
                    if inside and not missing:
                        assert sum(r[0]<=nx<r[2] and r[1]<=ny<r[3] for r in visible)==1
                        assert after.getpixel((x,y))==art.getpixel((x,y)),('title formula',x,y)
                        checked+=1
                    else:assert after.getpixel((x,y))==before.getpixel((x,y)),('outside title or missing asset',x,y)
                    changed+=after.getpixel((x,y))!=before.getpixel((x,y))
            if missing:assert checked==changed==0 and report['painted_native_pixels']==0
            else:assert checked==571560 and report['painted_native_pixels']==142890 and changed>10000
            key('F7');process.wait(timeout=240);assert process.returncode==0
        receipt=json.loads(receipt_path.read_text())
        base_path=project/'workplace/hd-prototype/catalog/graphics/static/normal-intro/ready/receipt.json'
        base=json.loads(base_path.read_text())
        for name in FIELDS:assert receipt[name]==base[name],name
        assert sha(root/'startup-memory.bin')==sha(base_path.parent/'startup-memory.bin')
        assert receipt['desktop']['physical_mouse_events']==0 and receipt['desktop']['scripted_mouse'] and not receipt['mouse_input_events']
        toggles=[e for e in receipt['desktop']['events'] if e['kind']=='toggle']
        assert len(toggles)==3 and all(e['guest_unchanged'] and e['before']==e['after'] for e in toggles)
        for name,digest in receipt['integration_input_sha256'].items():
            assert sha(project/name)==digest==sha(root/'source-snapshot'/name)
        result={'verified':True,'mode':mode,'classification':'normal original entry, native title/menu and HD/individual-missing toggles',
                'tool_sha256':sha(Path(__file__)),'receipt_sha256':sha(receipt_path),'baseline_sha256':sha(base_path),
                'pictures':pictures,'title_report':report,'checked_HD_pixels':checked,'changed_title_pixels':changed,
                'complete_RAM_sha256':sha(root/'startup-memory.bin'),'compared_fields':FIELDS}
        (root/f'verification-{mode}.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
        print(json.dumps({'verified':True,'mode':mode,'checked_HD_pixels':checked,'changed_title_pixels':changed}))
    finally:
        for child in reversed(children):
            if child.poll() is None:
                child.terminate()
                try:child.wait(timeout=5)
                except subprocess.TimeoutExpired:child.kill();child.wait(timeout=5)


def summary(root,project):
    metadata_path=root/'runtime-profiles-v1.json';metadata=json.loads(metadata_path.read_text())
    assert len(metadata['registered_source_names'])==20
    assert metadata['go_profiles_sha256']==sha(project/'tools/hd-graphics-remaining-profiles.go')
    assert metadata['tool_sha256']==sha(project/'tools/hd-remaining-runtime.py')==sha(root/'runtime-profile-source.txt')
    assert metadata['art_profiles_sha256']==sha(root/'profiles.json') and metadata['ready_sha256']==sha(root/'runtime-ready-v1.json')
    proofs={}
    for mode,folder in [('title','desktop-title'),('title-missing','desktop-title-missing'),('weather','desktop-regression')]:
        path=root/folder/f'verification-{mode}.json';proof=json.loads(path.read_text())
        receipt_path=root/folder/f'desktop-{mode}.json';receipt=json.loads(receipt_path.read_text())
        assert proof['verified'] and proof['receipt_sha256']==sha(receipt_path)
        assert proof['tool_sha256']==sha(project/'tools'/('verify-hd-weather.py' if mode=='weather' else 'verify-hd-remaining-runtime.py'))
        for name,digest in receipt['integration_input_sha256'].items():assert sha(project/name)==digest==sha(root/folder/'source-snapshot'/name)
        loaded=receipt['desktop']['assets']['graphics'];assert len(loaded)==28
        for source in metadata['records']:
            info=loaded[source['name']]
            if mode=='title-missing' and source['name']=='OP5.OZM':assert info['validation']=='fallback_missing_asset'
            else:assert info['validation']=='ready' and info['hd_sha256']==source['asset_sha256']
        proofs[mode]=proof
    assert proofs['title']['checked_HD_pixels']==571560 and proofs['title-missing']['changed_title_pixels']==0
    for mode,folder in [('title','desktop-title'),('title-missing','desktop-title-missing')]:
        path=root/folder/proofs[mode]['pictures'][1]['actual_png']
        crop=Image.open(path).convert('RGB').crop((464,300,816,492))
        if mode=='title':normal_window=crop.tobytes()
        else:assert crop.tobytes()==normal_window
    previous=project/'workplace/hd-prototype/catalog/graphics/static/desktop/palette'
    for picture in proofs['weather']['pictures']:
        assert Image.open(root/'desktop-regression'/picture['actual_png']).convert('RGBA').tobytes()==Image.open(previous/picture['actual_png']).convert('RGBA').tobytes()
    log_path=root/'runtime/startup-probe/platform-test.log';log=log_path.read_text()
    assert 'FAIL' not in log and len([l for l in log.splitlines() if l.startswith('ok')])==4
    result={'verified':True,'classification':'twenty source profiles, fixed original foreground, observed occlusion and normal title/menu',
            'tool_sha256':sha(Path(__file__)),'runtime_profiles_sha256':sha(metadata_path),
            'registered_sources':metadata['registered_source_names'],'normal_new_sources':['OP5.OZM'],
            'source_and_shared_consumer_tests':{'new_sources':20,'animation_frames':48,'sky_frames':8,'duel_frames':3,'eye_strips':4},
            'normal_title_HD_pixels':571560,'normal_menu_excluded_pixels':67584,
            'title_proof_sha256':sha(root/'desktop-title/verification-title.json'),
            'title_missing_proof_sha256':sha(root/'desktop-title-missing/verification-title-missing.json'),
            'regression_proof_sha256':sha(root/'desktop-regression/verification-weather.json'),
            'platform_test_log_sha256':sha(log_path),'binary_sha256':sha(root/'runtime/tokugawa-desktop-prototype'),
            'limitations':['other nineteen new sources not individually sampled in normal paths',
                           'static foreground preservation does not accept unknown dynamic writers',
                           'five unsupported OZM, PCX, whole-game HD and platform packaging remain incomplete']}
    target=root/'runtime-verification-summary.json';assert not target.exists()
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'registered_new_sources':20,'normal_new_sources':['OP5.OZM'],'summary_sha256':sha(target)}))


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000
    parser=argparse.ArgumentParser();parser.add_argument('--mode',choices=['title','title-missing','summary'],required=True)
    args=parser.parse_args();root,project=Path('/output'),Path('/project')
    if args.mode=='summary':summary(root,project)
    else:title(root,project,args.mode=='title-missing')


if __name__=='__main__':main()
