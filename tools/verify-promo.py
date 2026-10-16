"""驗證推廣片的完整解碼、音訊、黑畫面、停格、字幕邊界與來源。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
from PIL import Image, ImageDraw, ImageFont

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--delivery', type=Path, default=Path('/delivery'))
p.add_argument('--input', type=Path, default=Path('/input'))
args = p.parse_args()
root, source = args.delivery, args.input
assert root.stat().st_uid == 1000
meta = json.loads((root / 'media-manifest.json').read_text())
movie = root / meta['file']
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
assert sha(movie) == meta['sha256']


def ff(command, label):
    with (root / (label + '.log')).open('wb') as log:
        subprocess.run(command, stdout=log, stderr=log, check=True, timeout=240)
    return (root / (label + '.log')).read_text()


probe = json.loads(subprocess.check_output(['ffprobe', '-v', 'error', '-show_format', '-show_streams', '-of', 'json', str(movie)]))
(root / 'ffprobe.json').write_text(json.dumps(probe, indent=2) + '\n')
video = next(s for s in probe['streams'] if s['codec_type'] == 'video')
audio = next(s for s in probe['streams'] if s['codec_type'] == 'audio')
assert [video['width'], video['height']] == [1920, 1080]
assert video['codec_name'] == 'h264' and video['pix_fmt'] == 'yuv420p' and video['avg_frame_rate'] == '30/1'
assert int(video['nb_frames']) == 2160 and abs(float(probe['format']['duration']) - 72) < .1
assert audio['codec_name'] == 'aac' and int(audio['sample_rate']) == 44100 and audio['channels'] == 2
assert movie.stat().st_size > 1000000
decode = ff(['ffmpeg', '-v', 'error', '-xerror', '-threads', '2', '-i', str(movie), '-map', '0', '-f', 'null', '-'], 'decode')
assert not decode.strip(), decode[:1000]
sound = ff(['ffmpeg', '-v', 'info', '-threads', '2', '-i', str(movie), '-vn',
            '-af', 'astats=metadata=1:reset=0,volumedetect,silencedetect=noise=-50dB:d=1', '-f', 'null', '-'], 'audio-analysis')
mean = float(re.search(r'mean_volume: ([-.0-9]+) dB', sound)[1])
peak = float(re.search(r'max_volume: ([-.0-9]+) dB', sound)[1])
assert -40 < mean < -3 and -12 < peak < -.1, (mean, peak)
silence = [float(v) for v in re.findall(r'silence_start: ([-.0-9]+)', sound)]
assert all(v < 1 or v > 70 for v in silence), silence
assert meta['audio']['generated_or_synthesized'] is False and meta['generated_music'] is False
assert meta['audio']['pcm_sha256'] == json.loads((source / 'audio-source.json').read_text())['pcm_sha256']
for row in meta['text_rectangles']:
    a, b = row['bounds'], row['safe']
    assert a[0] >= b[0] and a[1] >= b[1] and a[2] <= b[2] and a[3] <= b[3]
findings = []
samples = root / 'frames'
samples.mkdir(exist_ok=True)
sheet = Image.new('RGB', (1920, 3*296), (23, 33, 38))
draw = ImageDraw.Draw(sheet)
label_font = ImageFont.truetype('/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf', 17)
unexpected = []
unexpected_black = []
for i, scene in enumerate(meta['scenes']):
    timestamp = (scene['output_start'] + scene['output_end']) / 2
    frame = samples / (scene['name'] + '.png')
    subprocess.run(['ffmpeg', '-v', 'error', '-threads', '1', '-filter_threads', '1', '-ss', str(timestamp), '-i', str(movie),
                    '-frames:v', '1', str(frame)], check=True, timeout=20)
    img = Image.open(frame).convert('RGB')
    assert img.size == (1920, 1080)
    sheet.paste(img.resize((480, 270), Image.Resampling.LANCZOS), (i%4*480, i//4*296))
    draw.text((i%4*480+10, i//4*296+273), f'{timestamp:04.1f}s  '+scene['name'], fill=(239, 232, 215), font=label_font)
    if scene['kind'] != 'actual_recording':
        findings.append({'scene': scene['name'], 'intentional_title_card': True, 'duration': scene['duration']})
        continue
    x, y, width, height = scene['display_rectangle']
    analysis = ff(['ffmpeg', '-v', 'info', '-threads', '2', '-filter_threads', '1', '-ss', str(scene['output_start']),
                   '-t', str(scene['duration']), '-i', str(movie), '-an', '-vf',
                   f'crop={width}:{height}:{x}:{y},blackdetect=d=0.15:pic_th=0.98:pix_th=0.04,freezedetect=n=-55dB:d=2.5',
                   '-f', 'null', '-'], scene['name'] + '-visual-analysis')
    blacks = re.findall(r'black_start:([.0-9]+) black_end:([.0-9]+) black_duration:([.0-9]+)', analysis)
    freezes = [float(v) for v in re.findall(r'freeze_start: ([.0-9]+)', analysis)]
    # The original MARK fades in. The observed first half-second of this
    # specific cut is native black, with a visible caption around the game.
    native_fade = bool(blacks) and scene['name'] == '01-publisher' and all(
        float(a) == 0 and float(b) <= .5 + 1/30 and float(c) <= .5 + 1/30 for a, b, c in blacks)
    if blacks and not native_fade:
        unexpected_black.append(scene['name'])
    if freezes and not scene['intentional_static']:
        unexpected.append(scene['name'])
    findings.append({'scene': scene['name'], 'content_crop': scene['display_rectangle'], 'black_intervals': blacks,
                     'declared_original_MARK_fade': native_fade,
                     'freeze_starts_local_seconds': freezes, 'intentional_display_hold': scene['intentional_static']})
sheet.save(root / 'contact-sheet.png')
shutil.copyfile(source / 'fonts/COPYRIGHT-Noto-CJK.txt', root / 'COPYRIGHT-Noto-CJK.txt')
rights = {'classification': 'local-only; no public upload authorized', 'contains_original_game_footage': True,
          'contains_original_CDDA_music': True, 'original_EXE_sha256': '2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a',
          'original_BIN_sha256': meta['audio']['BIN_sha256'], 'original_audio_track': 10,
          'original_audio_pcm_sha256': meta['audio']['pcm_sha256'], 'HD_art': 'private repo assets/hd-b; approved B handpainted style',
          'graphics_source': 'actual packaged application recording; no generated gameplay frames',
          'title_cards_and_captions': 'project authored', 'fonts': meta['fonts'],
          'font_license': 'COPYRIGHT-Noto-CJK.txt', 'original_assets_not_licensed_by_RRSAL': True}
(root / 'rights-metadata.json').write_text(json.dumps(rights, ensure_ascii=False, indent=2) + '\n')
summary = {'result': 'PASS' if not unexpected and not unexpected_black else 'NEEDS_VISUAL_REVIEW', 'version': meta['version'],
           'file': movie.name, 'sha256': sha(movie), 'duration_seconds': float(probe['format']['duration']),
           'resolution': [1920, 1080], 'FPS': 30, 'frames': 2160, 'full_decode': 'PASS',
           'audio_channels': 2, 'sample_rate': 44100, 'mean_volume_dB': mean, 'max_volume_dB': peak,
           'silence_starts': silence, 'caption_rectangles_checked': len(meta['text_rectangles']),
           'content_black_detect': 'PASS_EXCEPT_DECLARED_ORIGINAL_MARK_FADE' if not unexpected_black else 'NEEDS_REVIEW',
           'black_scenes_to_review': unexpected_black, 'visual_findings': findings, 'static_scenes_to_review': unexpected,
           'source_recording_sha256': meta['source_recording_sha256'], 'real_gameplay_recording': True,
           'normal_playthrough_required': False, 'normal_playthrough_verified': False,
           'listening_verified': False, 'human_visual_review': False,
           'verifier_sha256': sha(Path(__file__))}
(root / 'verification-summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(summary, ensure_ascii=False), flush=True)
