"""原版正常入口至首戰，核對真實視窗的天氣 HD、切換及逐項缺圖。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import time

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', choices=['weather', 'weather-missing'], required=True)
    args = parser.parse_args()
    root = Path('/output')
    assert root.stat().st_uid == 1000
    (root/'startup-probe').mkdir(exist_ok=True)
    env = dict(os.environ, DISPLAY=':98', LIBGL_ALWAYS_SOFTWARE='1', LP_NUM_THREADS='2')
    children = []

    def run(*command, **kwargs):
        return subprocess.run(command, env=env, timeout=15, check=True, **kwargs)

    def status():
        try:
            return json.loads((root/'desktop-status.json').read_text())
        except FileNotFoundError:
            return {}

    def wait_for(predicate, limit=30):
        deadline = time.monotonic()+limit
        while time.monotonic() < deadline:
            state = status()
            if predicate(state):
                return state
            assert process.poll() is None, f'desktop exited early: {process.returncode}'
            time.sleep(0.05)
        raise TimeoutError(f'desktop timeout: {status().get("step")}')

    def key(name):
        before = status()
        run('xdotool', 'keydown', '--clearmodifiers', name)
        try:
            if name == 'F6':
                wait_for(lambda s:s.get('enabled') != before.get('enabled'),15)
            elif name == 'F7':
                wait_for(lambda s:s.get('paused') != before.get('paused'),15)
            elif name == 'F9':
                wait_for(lambda s:len(s.get('events') or []) > len(before.get('events') or []),15)
        finally:
            run('xdotool', 'keyup', name)
        time.sleep(0.15)

    def capture(label):
        key('F9')
        state = status()
        event = state['events'][-1]
        assert event['kind'] == 'capture' and event['paused']
        geometry = dict(line.split('=',1) for line in run('xdotool','getwindowgeometry','--shell',window,stdout=subprocess.PIPE).stdout.decode().splitlines())
        actual = root/f'gui-{args.mode}-{label}.png'
        run('ffmpeg','-v','error','-y','-threads','1','-filter_threads','1','-f','x11grab',
            '-video_size',f'{geometry["WIDTH"]}x{geometry["HEIGHT"]}',
            '-i',f':98+{geometry["X"]},{geometry["Y"]}','-frames:v','1',str(actual),stdout=subprocess.DEVNULL)
        presented = Path(event['frame']['path'])
        assert Image.open(actual).convert('RGBA').tobytes() == Image.open(presented).convert('RGBA').tobytes()
        return {'actual_png':actual.name,'sha256':sha(actual),'presented':event,'window_pixels_equal':True}

    try:
        with (root/'xvfb.log').open('wb') as xvfb_log:
            xvfb = subprocess.Popen(['Xvfb',':98','-screen','0','1600x1000x24','-nolisten','tcp','-ac'],env=env,stdout=xvfb_log,stderr=subprocess.STDOUT)
        children.append(xvfb)
        deadline = time.monotonic()+10
        while not Path('/tmp/.X11-unix/X98').exists():
            assert xvfb.poll() is None and time.monotonic() < deadline
            time.sleep(0.05)
        for name in ['desktop-status.json','desktop-status.pending']:
            (root/name).unlink(missing_ok=True)
        command = [str(root/'tokugawa-desktop-prototype'),'-desktop', "-publisher-intro=false",'-desktop-speed','60',
                   '-steps','750000000','-mouse-file','/project/tools/player-path.json','-hd-battle-frame-proof']
        if args.mode == 'weather-missing':
            command += ['-hd-graphics-dir','/output/no-weather']
        receipt_path = root/f'desktop-{args.mode}.json'
        with receipt_path.open('wb') as stdout, (root/'desktop.stderr.log').open('wb') as stderr:
            process = subprocess.Popen(command,env=env,stdout=stdout,stderr=stderr)
            children.append(process)
            wait_for(lambda s:s.get('step',0)>1000000,60)
            window = run('xdotool','search','--name','德川家康',stdout=subprocess.PIPE).stdout.decode().splitlines()[0]
            run('xdotool','windowfocus','--sync',window)
            run('xdotool','mousemove','--window',window,'700','500')
            wait_for(lambda s:s.get('step',0)>=740000000 and len(s.get('terrain_history') or []) == 134,600)
            key('F7')
            key('F6')
            enhanced = capture('enabled')
            snapshot = status()
            key('F6')
            original = capture('original')
            key('F6')
            restored = capture('restored')
            assert enhanced['presented']['proof'] == original['presented']['proof'] == restored['presented']['proof']
            assert enhanced['presented']['frame']['png_sha256'] == restored['presented']['frame']['png_sha256']
            reports = snapshot['graphics']
            assert len(reports) == 1 and reports[0]['resource'] == 'WEATHER1.OZM'
            expected_mode = 'fallback_missing_asset' if args.mode == 'weather-missing' else 'hd'
            assert reports[0]['mode'] == expected_mode, reports
            assert reports[0]['painted_native_pixels'] == (0 if args.mode == 'weather-missing' else 144*74)
            assert snapshot['battle_frame']['mode'] == 'hd' and snapshot['terrain_scene']['floor_hd_count'] == 67
            raw = enhanced['presented']['frame']['native_indices']
            native_path = root/raw['path']
            assert sha(native_path) == raw['sha256']
            source = Path('/project/workplace/hd-prototype/catalog/decoded/WEATHER1.OZM.idx').read_bytes()
            native = native_path.read_bytes()
            crop = b''.join(native[y*640+464:y*640+608] for y in range(291,365))
            assert source == crop
            a = Image.open(root/enhanced['actual_png']).convert('RGBA')
            b = Image.open(root/original['actual_png']).convert('RGBA')
            art = Image.open('/project/assets/hd-b/art/WEATHER1-hd-b-v2.png').convert('RGBA')
            changed = verified = 0
            for y in range(148):
                for x in range(288):
                    point = (928+x,582+y)
                    expected = b.getpixel(point) if args.mode == 'weather-missing' else art.getpixel((x*art.width//288,y*art.height//148))
                    assert a.getpixel(point) == expected, ('weather pixel',point)
                    changed += a.getpixel(point) != b.getpixel(point)
                    verified += 1
            assert (changed>0) == (args.mode == 'weather')
            key('F7')
            process.wait(timeout=180)
            assert process.returncode == 0
        receipt = json.loads(receipt_path.read_text())
        baseline = json.loads(Path('/project/workplace/desktop-prototype/headless/startup-probe/latest.json').read_text())
        fields = ['successful_steps','instruction_budget','stop','stop_instruction_linear_eip','trace_tail',
                  'interrupt_counts','dpmi_calls','dpmi_unimplemented','frame','cd_pcm','mounted_disc',
                  'mouse_input_events','input_boundary_frames','real_mode_history','real_mode_last']
        for name in fields:
            assert receipt[name] == baseline[name], name
        assert sha(root/'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
        toggles = [e for e in receipt['desktop']['events'] if e['kind'] == 'toggle']
        assert len(toggles) == 3 and all(e['guest_unchanged'] and e['before'] == e['after'] for e in toggles)
        for name,digest in receipt['integration_input_sha256'].items():
            assert sha(root/'source-snapshot'/name) == digest
        result = {'verified':True,'mode':args.mode,'receipt_sha256':sha(receipt_path),'tool_sha256':sha(Path(__file__)),
                  'pictures':[enhanced,original,restored], 'weather_reference_pixels_verified':verified,
                  'weather_changed_pixels':changed,'graphics_observation':snapshot['graphics_observation'],
                  'weather_report':reports[0], 'compared_fields':fields,'complete_RAM_sha256':sha(root/'startup-memory.bin')}
        (root/f'verification-{args.mode}.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
        print(json.dumps({'mode':args.mode,'verified':True,'weather_changed_pixels':changed}))
    finally:
        for child in reversed(children):
            if child.poll() is None:
                child.terminate()
                try:
                    child.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    child.kill();child.wait(timeout=5)


if __name__ == '__main__':
    main()
