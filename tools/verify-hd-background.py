"""核對靜態圖、原版留白、部分遮擋及真實開場字標；正常來源分項記錄。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

from PIL import Image, ImageChops


FIELDS = ['successful_steps','instruction_budget','stop','stop_instruction_linear_eip','trace_tail',
          'interrupt_counts','dpmi_calls','dpmi_unimplemented','frame','cd_pcm','mounted_disc',
          'mouse_input_events','input_boundary_frames','real_mode_history','real_mode_last']


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def logo_gui(root, project):
    assert root.stat().st_uid == 1000
    (root/'startup-probe').mkdir(exist_ok=True)
    (root/'idle-mouse.json').write_text('[]')
    env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
    children = []

    def run(*command, **kwargs):
        return subprocess.run(command,env=env,timeout=15,check=True,**kwargs)

    def status():
        try:
            return json.loads((root/'desktop-status.json').read_text())
        except FileNotFoundError:
            return {}

    def wait_for(predicate, limit=30):
        end = time.monotonic()+limit
        while time.monotonic() < end:
            state = status()
            if predicate(state):
                return state
            assert process.poll() is None, f'early desktop exit {process.returncode}'
            time.sleep(0.05)
        raise TimeoutError(f'logo wait: {status()}')

    def key(name):
        before = status()
        run('xdotool','keydown','--clearmodifiers',name)
        try:
            if name == 'F6':
                wait_for(lambda s:s.get('enabled') != before.get('enabled'),15)
            elif name == 'F7':
                wait_for(lambda s:s.get('paused') != before.get('paused'),15)
            else:
                wait_for(lambda s:len(s.get('events') or []) > len(before.get('events') or []),15)
        finally:
            run('xdotool','keyup',name)
        time.sleep(0.15)

    def capture(label):
        key('F9')
        state = status();event = state['events'][-1]
        assert event['kind']=='capture' and event['paused']
        geometry = dict(line.split('=',1) for line in run('xdotool','getwindowgeometry','--shell',window,stdout=subprocess.PIPE).stdout.decode().splitlines())
        path = root/f'gui-logo-{label}.png'
        run('ffmpeg','-v','error','-y','-threads','1','-filter_threads','1','-f','x11grab',
            '-video_size',f'{geometry["WIDTH"]}x{geometry["HEIGHT"]}',
            '-i',f':98+{geometry["X"]},{geometry["Y"]}','-frames:v','1',str(path),stdout=subprocess.DEVNULL)
        assert Image.open(path).convert('RGBA').tobytes()==Image.open(Path(event['frame']['path'])).convert('RGBA').tobytes()
        return {'actual_png':path.name,'sha256':sha(path),'presented':event,'window_pixels_equal':True}

    try:
        with (root/'xvfb.log').open('wb') as log:
            xvfb = subprocess.Popen(['Xvfb',':98','-screen','0','1600x1000x24','-nolisten','tcp','-ac'],env=env,stdout=log,stderr=subprocess.STDOUT)
        children.append(xvfb)
        deadline = time.monotonic()+10
        while not Path('/tmp/.X11-unix/X98').exists():
            assert xvfb.poll() is None and time.monotonic()<deadline
            time.sleep(0.05)
        receipt_path = root/'desktop-logo.json'
        command = [str(root/'tokugawa-desktop-prototype'),'-desktop', "-publisher-intro=false",'-desktop-speed','60','-steps','200000000',
                   '-mouse-file',str(root/'idle-mouse.json'),'-hd-battle-frame-proof']
        with receipt_path.open('wb') as stdout,(root/'desktop.stderr.log').open('wb') as stderr:
            process = subprocess.Popen(command,env=env,stdout=stdout,stderr=stderr)
            children.append(process)
            wait_for(lambda s:s.get('step',0)>1000000,60)
            window = run('xdotool','search','--name','德川家康',stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
            run('xdotool','windowfocus','--sync',window)
            run('xdotool','mousemove','--window',window,'700','500')
            wait_for(lambda s:s.get('step',0)>=12000000,60)
            key('F6')
            wait_for(lambda s:any(r['resource']=='OP1.OZM' and r['mode']=='hd' for r in s.get('graphics') or []),60)
            key('F7')
            key('F6')
            original = capture('original')
            key('F6')
            enhanced = capture('enabled')
            state = status()
            reports = [r for r in state.get('graphics') or [] if r['resource']=='OP1.OZM']
            assert len(reports)==1 and reports[0]['mode']=='hd', reports
            key('F6')
            again = capture('original-again')
            key('F6')
            restored = capture('restored')
            assert original['presented']['proof']==enhanced['presented']['proof']==again['presented']['proof']==restored['presented']['proof']
            assert original['presented']['frame']['png_sha256']==again['presented']['frame']['png_sha256']
            assert enhanced['presented']['frame']['png_sha256']==restored['presented']['frame']['png_sha256']
            logo = json.loads((project/'workplace/hd-prototype/catalog/graphics/static/logo/profiles.json').read_text())['record']
            bounds = tuple(v*2 for v in logo['paint_rect'])
            assert reports[0]['painted_native_pixels']==(bounds[2]-bounds[0])*(bounds[3]-bounds[1])//4
            a = Image.open(root/enhanced['actual_png']).convert('RGB')
            b = Image.open(root/original['actual_png']).convert('RGB')
            art = Image.open(project/'assets/hd-b/art'/logo['asset']).convert('RGB')
            raw = original['presented']['frame']['native_indices']
            native_path = root/raw['path'];assert sha(native_path)==raw['sha256']
            native = native_path.read_bytes();assert len(native)==640*400
            palette = [raw['palette'][i] for i in raw['dac']]
            expected = Image.frombytes('P',(640,400),native)
            expected.putpalette([v for c in palette for v in c[:3]]+[0]*(768-48))
            expected = expected.convert('RGB').resize((1280,800),Image.Resampling.NEAREST)
            diff = ImageChops.difference(b,expected)
            channels = diff.split()
            overlay = ImageChops.lighter(ImageChops.lighter(channels[0],channels[1]),channels[2]).point(lambda v:255 if v else 0)
            cursor = overlay.getbbox()
            ignored = None
            if cursor:
                assert cursor[2]-cursor[0] <= 32 and cursor[3]-cursor[1] <= 32
                ignored = (max(0,cursor[0]-32),max(0,cursor[1]-32),min(1280,cursor[2]+32),min(800,cursor[3]+32))
            changed = verified = 0
            for y in range(800):
                for x in range(1280):
                    inside = bounds[0] <= x < bounds[2] and bounds[1] <= y < bounds[3]
                    if not inside:
                        assert a.getpixel((x,y))==b.getpixel((x,y)), ('outside logo',x,y)
                        continue
                    changed += a.getpixel((x,y))!=b.getpixel((x,y))
                    if ignored and ignored[0] <= x < ignored[2] and ignored[1] <= y < ignored[3]:
                        continue
                    assert a.getpixel((x,y))==art.getpixel((x,y)), ('logo sample',x,y)
                    verified += 1
            assert changed>1000 and verified>30000
            key('F7')
            process.wait(timeout=300)
            assert process.returncode==0
        receipt = json.loads(receipt_path.read_text())
        baseline_path = project/'workplace/hd-prototype/catalog/graphics/static/normal-intro/ready/receipt.json'
        baseline = json.loads(baseline_path.read_text())
        for name in FIELDS:
            assert receipt[name]==baseline[name], name
        original_root = baseline_path.parent
        assert sha(root/'startup-memory.bin')==sha(original_root/'startup-memory.bin')
        assert receipt['desktop']['physical_mouse_events']==0 and receipt['desktop']['scripted_mouse'] and not receipt['mouse_input_events']
        toggles = [e for e in receipt['desktop']['events'] if e['kind']=='toggle']
        assert len(toggles)==5 and all(e['guest_unchanged'] and e['before']==e['after'] for e in toggles)
        for name,digest in receipt['integration_input_sha256'].items():
            assert sha(root/'source-snapshot'/name)==digest
        result = {'verified':True,'classification':'normal idle opening OP1 HD, explicit empty input timeline, real X11 and original 200M state comparison',
                  'tool_sha256':sha(Path(__file__)),'receipt_sha256':sha(receipt_path),'baseline_sha256':sha(baseline_path),
                  'pictures':[original,enhanced,again,restored],'logo_report':reports[0],'changed_pixels':changed,
                  'formula_verified_pixels':verified,'original_overlay_bounds':cursor,'formula_excluded_bounds':ignored,
                  'complete_RAM_sha256':sha(root/'startup-memory.bin'),'compared_fields':FIELDS}
        (root/'verification-logo.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
        print(json.dumps({'verified':True,'mode':'logo','changed_pixels':changed,'formula_verified_pixels':verified}))
    finally:
        for child in reversed(children):
            if child.poll() is None:
                child.terminate()
                try:
                    child.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    child.kill();child.wait(timeout=5)


def summary(root, project):
    palette_path = root/'palette-probe/palette-captures.json'
    assert sha(palette_path)=='cbe61f72fc33ce5a3e2ed5873ac5d06fa9fdba96d9688c478ce579c1f1f73936'
    captures = json.loads(palette_path.read_text())
    assert len(captures)==7 and (root/'palette-probe/exit.txt').read_text().strip()=='0'
    observed = [[r['frame']['native_indices']['palette'][i] for i in r['frame']['native_indices']['dac']] for r in captures]
    assert observed[4]==observed[5] and all(p!=observed[4] for p in observed[:4]+observed[6:])
    profiles_path = root/'profiles.json'
    profiles = json.loads(profiles_path.read_text())
    assert profiles['tool_sha256']==sha(project/'tools/hd-background-prepare.py')==sha(root/'prepare-source.txt')
    assert profiles['review_sha256']==sha(root/'review.json')
    logo_path = root/'logo/profiles.json';logo = json.loads(logo_path.read_text())
    assert logo['tool_sha256']==sha(project/'tools/hd-logo-prepare.py')==sha(root/'logo/prepare-source.txt')
    records = profiles['records']+[logo['record']]
    assert len(records)==6
    for r in records:
        path = project/'assets/hd-b/art'/r['asset']
        assert sha(path)==r['asset_sha256']
        idx_path = project/'workplace/hd-prototype/catalog/decoded'/(r['name']+'.idx')
        assert sha(idx_path)==r['source']['indices_sha256']
        im = Image.open(path).convert('RGB')
        assert im.size==(r['source']['width']*2,r['source']['height']*2)
        if r['name'].startswith('BATTLE'):
            data = idx_path.read_bytes()
            assert data.count(0)==r['padding_pixels']==148302
            for i,value in enumerate(data):
                if value==0:
                    x,y=i%640*2,i//640*2
                    assert all(im.getpixel((x+dx,y+dy))==(0,0,0) for dx in [0,1] for dy in [0,1])
        if r['name']=='OP2.OZM':
            for y in range(im.height):
                for x in range(im.width):
                    if x>=960 or y>=300:
                        assert im.getpixel((x,y))==(20,20,227)
    folder = root/'desktop/palette'
    receipt_path,proof_path = folder/'desktop-weather.json',folder/'verification-weather.json'
    receipt,proof = json.loads(receipt_path.read_text()),json.loads(proof_path.read_text())
    assert proof['verified'] and proof['receipt_sha256']==sha(receipt_path)
    assert proof['tool_sha256']==sha(project/'tools/verify-hd-weather.py')
    assert receipt['successful_steps']==750000000 and len(receipt['mouse_input_events'])==390
    assert sha(folder/'startup-memory.bin')=='4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name,digest in receipt['integration_input_sha256'].items():
        assert sha(project/name)==digest==sha(folder/'source-snapshot'/name)
    loaded = receipt['desktop']['assets']['graphics']
    for r in records:
        info = loaded[r['name']]
        assert info['validation']=='ready' and info['hd_sha256']==r['asset_sha256']
    prior = project/'workplace/hd-prototype/catalog/terrain/objects/walls/desktop'
    for picture in proof['pictures']:
        path = folder/picture['actual_png']
        assert picture['window_pixels_equal'] and Image.open(path).convert('RGBA').tobytes()==Image.open(prior/picture['actual_png']).convert('RGBA').tobytes()
    logo_folder = root/'desktop-logo/palette/ready'
    logo_proof_path = logo_folder/'verification-logo.json'
    logo_proof = json.loads(logo_proof_path.read_text())
    assert logo_proof['verified'] and logo_proof['tool_sha256']==sha(Path(__file__))
    assert logo_proof['receipt_sha256']==sha(logo_folder/'desktop-logo.json')
    assert sha(logo_folder/'tokugawa-desktop-prototype')==sha(folder/'tokugawa-desktop-prototype')
    log_path = root/'runtime/palette/startup-probe/platform-test.log'
    log = log_path.read_text();assert 'FAIL' not in log and len([l for l in log.splitlines() if l.startswith('ok')])==4
    result = {'verified':True,'classification':'six calibrated static sources, visible opaque fragments and normal idle publisher logo',
              'tool_sha256':sha(Path(__file__)),'profiles_sha256':sha(profiles_path),'logo_profiles_sha256':sha(logo_path),
              'original_palette_evidence_sha256':sha(palette_path),'original_palette_samples':7,'stable_palette_samples':2,
              'reviewed_sources':[r['name'] for r in records],'normal_new_source_names':['OP1.OZM'],
              'normal_weather_source_names':['WEATHER1.OZM'],'remaining_five_sources_normal_not_sampled':True,
              'receipt_sha256':sha(receipt_path),'GUI_verification_sha256':sha(proof_path),'logo_GUI_verification_sha256':sha(logo_proof_path),
              'binary_sha256':sha(folder/'tokugawa-desktop-prototype'),'platform_test_log_sha256':sha(log_path),
              'original_first_battle_images_unchanged':True,
              'limitations':['five new non-logo sources not individually sampled in normal player paths',
                             'unknown direct VRAM updates still fall back for the affected visible fragment',
                             'remaining OZM, animation and PCX art incomplete; no platform packages yet']}
    target = root/'verification-summary.json';assert not target.exists()
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'static_sources':6,'normal_logo':True,'sha256':sha(target)}))


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000
    parser = argparse.ArgumentParser();parser.add_argument('--mode',choices=['logo','summary'],default='summary')
    args = parser.parse_args()
    root,project=Path('/output'),Path('/project')
    if args.mode=='logo':
        logo_gui(root,project)
    else:
        summary(root,project)


if __name__=='__main__':
    main()
