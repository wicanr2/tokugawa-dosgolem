"""將真正桌面錄影剪成 72 秒介紹，配樂只取已核對的原版 CDDA。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import wave
from PIL import Image, ImageDraw, ImageFont

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--input', type=Path, default=Path('/input'))
p.add_argument('--work', type=Path, default=Path('/output'))
p.add_argument('--delivery', type=Path, default=Path('/delivery'))
p.add_argument('--reuse-manifest', type=Path)
args = p.parse_args()
source, work, delivery = args.input, args.work, args.delivery
assert all(p.is_dir() for p in [source, work, delivery])
assert work.stat().st_uid == delivery.stat().st_uid == 1000
record = json.loads((source / 'recording.json').read_text())
assert record['result'] == 'PASS'
version = record['version']
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
assert sha(source / 'raw.mkv') == record['recording_sha256']
font_records = json.loads((source / 'font-sources.json').read_text())
assert all(sha(source / r['file']) == r['sha256'] for r in font_records)
audio = json.loads((source / 'audio-source.json').read_text())
assert audio['generated_or_synthesized'] is False and audio['track'] == 10
with wave.open(str(source / 'original-track10.wav'), 'rb') as w:
    assert w.getparams()[:3] == (2, 2, 44100)
    assert hashlib.sha256(w.readframes(w.getnframes())).hexdigest() == audio['pcm_sha256']
timeline = {r['label']: r['recording_seconds'] for r in record['timeline']}
previous = json.loads(args.reuse_manifest.read_text()) if args.reuse_manifest else None
if previous:
    assert previous['source_recording_sha256'] == record['recording_sha256']
    assert previous['fonts'] == font_records
    assert previous['resolution'] == [1920, 1080] and previous['FPS'] == 30
previous_scenes = {s['name']: s for s in previous['scenes']} if previous else {}
W, H, FPS = 1920, 1080, 30
BG, GOLD, INK = (23, 33, 38), (195, 157, 87), (239, 232, 215)
font_paths = [source / 'fonts/NotoSerifCJK-Regular.ttc', source / 'fonts/NotoSerifCJK-Bold.ttc']
boxes = []


def font(size, bold=False):
    return ImageFont.truetype(str(font_paths[int(bold)]), size, index=3)


def text(image, value, center, size, bold=False, color=INK, safe=(60, 10, 1860, 1060)):
    d = ImageDraw.Draw(image)
    f = font(size, bold)
    x, y = center
    bounds = d.textbbox((x, y), value, font=f, anchor='mm')
    assert bounds[0] >= safe[0] and bounds[1] >= safe[1] and bounds[2] <= safe[2] and bounds[3] <= safe[3], (value, bounds)
    d.text((x, y), value, font=f, anchor='mm', fill=color, stroke_width=0)
    boxes.append({'text': value, 'bounds': list(bounds), 'safe': list(safe)})


def card(name, title, subtitle, small=''):
    image = Image.new('RGB', (W, H), BG)
    d = ImageDraw.Draw(image)
    d.rectangle((64, 64, W-65, H-65), outline=GOLD, width=2)
    d.line((640, 330, 1280, 330), fill=GOLD, width=2)
    text(image, title, (960, 470), 128, True, GOLD)
    text(image, subtitle, (960, 640), 44)
    if small:
        text(image, small, (960, 790), 27)
    path = work / (name + '.png')
    image.save(path)
    return path


def run(command, label):
    with (work / (label + '.log')).open('wb') as log:
        subprocess.run(command, check=True, stdout=log, stderr=log, timeout=180)


def span(a, b, duration, name, caption, height=800):
    return {'name': name, 'source_start': timeline[a], 'source_duration': timeline[b] - timeline[a],
            'duration': duration, 'caption': caption, 'height': height, 'kind': 'actual_recording'}


scenes = [
    {'name': '00-title', 'duration': 4, 'kind': 'title_card', 'card': card('00-title', '德川家康', 'DOS 原版 × 戰國手繪 HD', '三方原的危局，天下人的漫長道路')},
    {'name': '01-publisher', 'source_start': timeline['publisher-hd'] - .4, 'source_duration': 5,
     'duration': 5, 'caption': '熟悉的開場，換上戰國手繪新貌', 'height': 960, 'kind': 'actual_recording'},
    {'name': '02-choice', 'source_start': timeline['startup-choice'] + .1, 'source_duration': 1.7,
     'duration': 4, 'caption': '原版與 HD，啟動時自由選擇', 'height': 800, 'kind': 'actual_recording'},
    span('game-80000000', 'game-130000000', 8, '03-menu', '原版故事與玩法，由 dosgolem 執行'),
    span('game-130000000', 'game-200000000', 10, '04-portraits', '軍議、人物與抉擇，走進家康的戰國'),
    span('game-250000000', 'game-300000000', 10, '05-story', '武將、場景與捲軸介面，逐項換上 HD'),
    {'name': '06-battle', 'source_start': timeline['battle-hd'] + .1, 'source_duration': 3.6,
     'duration': 7, 'caption': '手繪地形與部隊，重返三方原戰場', 'height': 800, 'kind': 'actual_recording'},
    {'name': '07-compare', 'source_start': timeline['battle-hd'],
     'source_duration': timeline['battle-restored'] - timeline['battle-hd'] + 1.5, 'duration': 8,
     'caption': '原版與 HD，遊戲中隨時切換', 'height': 800, 'kind': 'actual_recording'},
    {'name': '08-toolbar', 'source_start': timeline['bar-open'],
     'source_duration': timeline['bar-hd'] - timeline['bar-open'] + 1.5, 'duration': 8,
     'caption': '把滑鼠移到頂端，展開隱藏功能列', 'height': 864, 'kind': 'actual_recording'},
    {'name': '09-platforms', 'duration': 4, 'kind': 'title_card', 'card': card('09-platforms', '桌面完整版', 'Windows・Linux・macOS', 'macOS 已完成編譯，尚待真機試跑')},
    {'name': '10-end', 'duration': 4, 'kind': 'title_card', 'card': card('10-end', '德川家康', '原版遊戲運作，HD 圖像隨時切換', 'wicanr2/tokugawa-dosgolem  ·  '+version)},
]
assert sum(s['duration'] for s in scenes) == 72
base_args = ['ffmpeg', '-v', 'warning', '-nostdin', '-threads', '2', '-filter_threads', '1', '-filter_complex_threads', '1']
clips = []
elapsed = 0
for scene in scenes:
    path = work / (scene['name'] + '.mp4')
    frames = int(scene['duration'] * FPS)
    command = base_args[:]
    if scene['kind'] == 'title_card':
        command += ['-loop', '1', '-framerate', str(FPS), '-i', str(scene['card']), '-vf', 'format=yuv420p']
        scene['intentional_static'] = True
    else:
        h = scene['height']
        display_h = 880
        display_w = round(1280 * display_h / h / 2) * 2
        left, top = (W-display_w)//2, 62
        overlay = Image.new('RGBA', (W, H), (0, 0, 0, 0))
        d = ImageDraw.Draw(overlay)
        d.rectangle((left-2, top-2, left+display_w+1, top+display_h+1), outline=GOLD+(255,), width=2)
        d.line((120, 970, 1800, 970), fill=GOLD+(255,), width=1)
        text(overlay, '德川家康 × dosgolem', (960, 27), 26, color=GOLD+(255,), safe=(80, 0, 1840, 55))
        text(overlay, scene['caption'], (960, 1015), 38, safe=(80, 978, 1840, 1060))
        overlay_path = work / (scene['name'] + '-overlay.png')
        overlay.save(overlay_path)
        factor = scene['duration'] / scene['source_duration']
        chain = (f'[0:v]setpts=(PTS-STARTPTS)*{factor:.12f},crop=1280:{h}:0:0,'
                 f'scale={display_w}:{display_h}:flags=lanczos,pad={W}:{H}:{left}:{top}:color=0x172126,'
                 f'setsar=1,fps={FPS},tpad=stop_mode=clone:stop_duration=1[base];'
                 '[base][1:v]overlay=0:0:format=auto,format=yuv420p[v]')
        command += ['-ss', str(scene['source_start']), '-t', str(scene['source_duration']), '-i', str(source / 'raw.mkv'),
                    '-i', str(overlay_path), '-filter_complex', chain, '-map', '[v]']
        scene['playback_speed'] = scene['source_duration'] / scene['duration']
        scene['crop'] = [0, 0, 1280, h]
        scene['display_rectangle'] = [left, top, display_w, display_h]
        scene['intentional_static'] = scene['name'] in ['02-choice', '06-battle', '07-compare', '08-toolbar']
    command += ['-an', '-frames:v', str(frames), '-c:v', 'libx264', '-preset', 'veryfast', '-crf', '19',
                '-pix_fmt', 'yuv420p', '-r', str(FPS), '-threads', '2', '-movflags', '+faststart', str(path)]
    if path.exists():
        old = previous_scenes[scene['name']]
        spec_keys = ['kind', 'duration', 'source_start', 'source_duration', 'caption', 'height',
                     'intentional_static', 'crop', 'display_rectangle', 'playback_speed']
        assert all(old.get(k) == scene.get(k) for k in spec_keys), scene['name']
        assert sha(path) == old['clip_sha256'], path
        print('REUSED_VERIFIED', scene['name'], flush=True)
    else:
        run(command, scene['name'])
    scene.update(output_start=elapsed, output_end=elapsed+scene['duration'], clip_sha256=sha(path))
    elapsed += scene['duration']
    clips.append(path)
    print('COMPOSED', scene['name'], flush=True)
concat = work / 'concat.txt'
concat.write_text(''.join("file '"+str(p)+"'\n" for p in clips))
output = delivery / f'tokugawa-dosgolem-{version}-promo.mp4'
assert not output.exists()
run(['ffmpeg', '-v', 'warning', '-nostdin', '-f', 'concat', '-safe', '0', '-i', str(concat),
     '-i', str(source / 'original-track10.wav'), '-map', '0:v:0', '-map', '1:a:0', '-t', '72',
     '-c:v', 'copy', '-af', 'volume=0.7,afade=t=in:st=0:d=0.5,afade=t=out:st=70.5:d=1.5',
     '-c:a', 'aac', '-b:a', '192k', '-ar', '44100', '-ac', '2', '-movflags', '+faststart',
     '-metadata', 'title=德川家康 × dosgolem', '-metadata', 'comment=Local-only original game footage and CDDA music', str(output)], 'final-mux')
captions = []
for s in scenes:
    if s['kind'] == 'actual_recording':
        i = len(captions) + 1
        stamp = lambda v: f'{int(v)//3600:02}:{int(v)//60%60:02}:{int(v)%60:02},000'
        captions.append(f'{i}\n{stamp(s["output_start"])} --> {stamp(s["output_end"])}\n{s["caption"]}\n')
(delivery / 'captions.srt').write_text('\n'.join(captions))
clean_scenes = [{k: str(v) if isinstance(v, Path) else v for k, v in s.items()} for s in scenes]
metadata = {'version': version, 'binary_commit': record['binary_commit'], 'duration_seconds': 72,
            'resolution': [W, H], 'FPS': FPS, 'file': output.name, 'sha256': sha(output),
            'source_recording_sha256': record['recording_sha256'], 'scenes': clean_scenes,
            'audio': audio, 'audio_gain': 0.7, 'audio_fades_seconds': [0.5, 1.5],
            'fonts': font_records, 'text_rectangles': boxes,
            'rights': 'local-only; contains original game footage, original CDDA music and generated HD assets; not uploaded',
            'normal_playthrough_verified': False, 'normal_playthrough_required': False,
            'sources_are_actual_application_recording': True, 'generated_music': False,
            'original_game_colors_regraded': False, 'editing': 'cuts, cropping, proportional scaling, captions, soundtrack; accelerated source intervals declared'}
(delivery / 'media-manifest.json').write_text(json.dumps(metadata, ensure_ascii=False, indent=2) + '\n')
print('OUTPUT', output, flush=True)
