"""核對原版程式自己的存檔寫入及正常選單讀回，不解碼 SAV 語意。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import sys
from PIL import Image, ImageChops

assert Path('/.dockerenv').exists() and os.getuid() == 1000
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000
parser = argparse.ArgumentParser()
parser.add_argument('--source-root', type=Path, default=project)
parser.add_argument('--desktop', type=Path)
parser.add_argument('--imported', action='store_true')
args = parser.parse_args()
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
if args.imported:
    data = read(root / 'input-data.json')
    manifest_path = project / data['manifest_project_path']
    manifest = read(manifest_path)
    assert sha(manifest_path) == data['manifest_sha256']
    assert manifest['classification'] == 'original program data plus original-game save; not downloaded baseline DAT'
    assert manifest['saved_dat_sha256'] == data['saved_dat_sha256']
    assert manifest['unchanged_resources'] == 200
    assert len(manifest['changed_resources_from_download']) == 1
    assert manifest['changed_resources_from_download'][0]['resource'] == 'I003.SAV'
    for item in manifest['files']:
        assert sha(manifest_path.parent / 'game' / item['path']) == item['sha256']
    sources, receipt = read(root / 'build-sources.json'), read(root / 'receipt.json')
    assert receipt['integration_input_sha256'] == sources and len(sources) == 106
    assert all(sha(args.source_root / name) == digest for name, digest in sources.items())
    assert receipt['successful_steps'] == 140000000 and receipt['stop'] == 'instruction_budget'
    assert receipt['fd2_startup_calls'] == receipt['native_runtime_instructions_skipped'] == 0
    assert receipt['input_sha256'] == '2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a'
    schedule = read(root / 'player-path.json')
    assert len(schedule) == len(receipt['mouse_input_events']) == 9 and receipt['mouse_input_schedule'] == schedule
    assert all(event['actual_step'] == request['step'] for event, request in zip(receipt['mouse_input_events'], schedule))
    origin = manifest_path.parent.parent
    assert sha(origin / 'receipt.json') == manifest['save_origin_receipt_sha256']
    saved_frame = manifest['save_origin_frame']
    assert receipt['frame']['raw_indexed_sha256'] == saved_frame['raw_indexed_sha256']
    a = root / 'startup-probe' / Path(receipt['frame']['path']).name
    b = origin / 'startup-probe' / Path(saved_frame['path']).name
    assert sha(a) == receipt['frame']['png_sha256'] and sha(b) == saved_frame['png_sha256']
    loaded, saved = Image.open(a).convert('RGBA'), Image.open(b).convert('RGBA')
    assert loaded.size == saved.size == (640, 400)
    differences = sum(x != y for x, y in zip(loaded.getdata(), saved.getdata()))
    loaded_rgb, saved_rgb = loaded.convert('RGB'), saved.convert('RGB')
    report = dict(result='PASS' if differences == 0 else 'PASS_NATIVE_INDICES_ONLY',
                  scope='one imported original FILE 3 save/load sample via normal title and confirmation inputs',
                  source_count=len(sources), steps=140000000, ordinary_mouse_events=9,
                  saved_dat_sha256=data['saved_dat_sha256'], unchanged_resources=200,
                  raw_indexed_sha256=receipt['frame']['raw_indexed_sha256'],
                  rgba_equal=differences == 0, rgba_differences=differences,
                  rgb_differences=sum(x != y for x, y in zip(loaded_rgb.getdata(), saved_rgb.getdata())),
                  rgb_difference_bbox=ImageChops.difference(loaded_rgb, saved_rgb).getbbox(),
                  color_difference_cause='unknown; palette phase not reverse engineered',
                  receipt_sha256=sha(root / 'receipt.json'), ram_sha256=sha(root / 'startup-memory.bin'),
                  input_manifest_sha256=sha(manifest_path), tool_sha256=sha(Path(__file__)))
    assert args.desktop is None, '--imported desktop verification is not implemented'
    (root / 'verification.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({k: report[k] for k in ['result', 'raw_indexed_sha256', 'rgba_equal', 'rgb_differences']}))
    raise SystemExit(0)
before, after = root / 'save-before.dat', root / 'save-after.dat'
original = project / 'original/game/DX5.DAT'
assert sha(before) == sha(original) == 'de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37'
old, new = before.read_bytes(), after.read_bytes()
inventory = read(project / 'workplace/hd-prototype/asset-inventory.json')
assert inventory['input_sha256'] == sha(before) and inventory['record_count'] == 201
assert len(old) == len(new) and old[:inventory['candidate_table_end']] == new[:inventory['candidate_table_end']]
changed, unchanged = [], []
for record in inventory['records']:
    start, length = record['candidate_file_offset'], record['candidate_length']
    a, b = old[start:start + length], new[start:start + length]
    if a == b:
        unchanged.append(record['name'])
    else:
        changed.append(dict(resource=record['name'], file_offset=start, length=length,
                            changed_bytes=sum(x != y for x, y in zip(a, b))))
assert len(unchanged) == 200 and len(changed) == 1
assert changed[0]['resource'] == 'I003.SAV' and changed[0]['changed_bytes'] == 134
manifest_path = root / 'save-import/manifest.json'
manifest = read(manifest_path)
assert manifest['saved_dat_sha256'] == sha(after)
for record in manifest['files']:
    assert sha(root / 'save-import/game' / record['path']) == record['sha256']
case = root / 'load-check/r4'
r = read(case / 'receipt.json')
sources = read(root / 'build-sources.json')
assert len(sources) == 106 and r['integration_input_sha256'] == sources
assert all(sha(args.source_root / name) == digest for name, digest in sources.items())
assert r['input_sha256'] == '2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a'
assert r['successful_steps'] == 140000000 and r['stop'] == 'instruction_budget'
assert r['fd2_startup_calls'] == r['native_runtime_instructions_skipped'] == 0
schedule = read(case / 'continue-path.json')
assert len(schedule) == len(r['mouse_input_events']) == 9 and r['mouse_input_schedule'] == schedule
assert all(event['actual_step'] == request['step'] for event, request in zip(r['mouse_input_events'], schedule))
saved_state = read(root / 'state-action-3.json')
saved_raw = saved_state['events'][-1]['proof']['raw_indexed_sha256']
assert r['frame']['raw_indexed_sha256'] == saved_raw == 'eac8871b05cc4a25ded2be4fb5f25baf0ef0f51f291b68fd11412d4f207cfb82'
draws = {d['resource'] for d in r['graphics_draw_trace'].get('records') or [] if d['verified']}
assert {'OP1.OZM', 'OP5.OZM', 'MAPGAMEN.OZM', 'WEATHER1.OZM'} <= draws
frame_path = case / 'startup-probe' / Path(r['frame']['path']).name
assert sha(frame_path) == r['frame']['png_sha256']
loaded = Image.open(frame_path).convert('RGBA')
saved = Image.open(root / 'action-3.png').convert('RGBA')
assert loaded.size == (640, 400) and saved.size == (1280, 800)
rgba_differences = sum(loaded.getpixel((x, y)) != saved.getpixel((x * 2, y * 2))
                       for y in range(400) for x in range(640))
assert rgba_differences == 0, 'saved and loaded RGBA pixels differ'
report = dict(result='PASS', scope='one original-game FILE 3 save/load sample via normal title and confirmation inputs',
              save_format_decoded=False, other_slots_verified=False, full_ram_save_load_parity_claimed=False,
              source_count=len(sources), steps=r['successful_steps'], ordinary_mouse_events=len(schedule),
              original_dat_sha256=sha(before), saved_dat_sha256=sha(after), unchanged_resources=len(unchanged),
              changed_resources=changed, saved_and_loaded_native_indices_sha256=saved_raw,
              rgba_differences_including_cursor=rgba_differences,
              receipt_sha256=sha(case / 'receipt.json'), ram_sha256=sha(case / 'startup-memory.bin'),
              input_manifest_project_path='workplace/hd-prototype/catalog/remaining/consumers/battle/r8/save-import/manifest.json',
              input_manifest_sha256=sha(manifest_path), input_mouse_sha256=sha(case / 'continue-path.json'),
              python=sys.version.split()[0], tool_sha256=sha(Path(__file__)))
if args.desktop:
    desktop = args.desktop
    e, g = read(desktop / 'exploration-status.json'), read(desktop / 'receipt.json')
    assert e['phase'] in ['finished', 'finished_deadline'] and not e['harness_only']
    assert e['input_data']['kind'] == 'original-game save'
    assert e['input_data']['dat_sha256'] == sha(after) and e['input_data']['manifest_sha256'] == sha(manifest_path)
    assert g['stop'] == 'desktop_closed' and g['integration_input_sha256'] == sources
    assert g['mouse_input_schedule'] == schedule and g['mouse_input_events'][:9] == r['mouse_input_events']
    assert g['input_boundary_frames'] == r['input_boundary_frames']
    assert sha(desktop / 'source-snapshot/tools/explore-battle.py') == e['tool_sha256']
    for capture in e['captures']:
        actual = desktop / Path(capture['actual']).relative_to('/output')
        frame = desktop / Path(capture['event']['frame']['path']).relative_to('/output')
        assert sha(actual) == capture['sha256'] and capture['window_pixels_equal']
        assert Image.open(actual).convert('RGBA').tobytes() == Image.open(frame).convert('RGBA').tobytes()
    assert e['captures'][0]['event']['proof']['raw_indexed_sha256'] == saved_raw
    toggles = [event for event in read(desktop / 'desktop-status.json')['events'] if event['kind'] == 'toggle']
    assert toggles and all(event['guest_unchanged'] for event in toggles)
    report['desktop_sample'] = dict(result='PASS', actual_window_captures=len(e['captures']),
                                    input_prefix_events_equal=9, original_state_unchanged_on_toggles=True,
                                    receipt_sha256=sha(desktop / 'receipt.json'), scope='saved scene and army-table HD sample; no combat claim')
(case / 'verification.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
print(json.dumps({k: report[k] for k in ['result', 'scope', 'unchanged_resources', 'rgba_differences_including_cursor']}))
