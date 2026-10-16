"""獨立核對 MAP_WAKU 快取主框架與正常讀檔的實際視窗。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image

assert Path('/.dockerenv').is_file() and os.getuid() == 1000
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000
parser = argparse.ArgumentParser()
parser.add_argument('--source-root', type=Path, default=project)
parser.add_argument('--corners', action='store_true')
args = parser.parse_args()
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
rect = lambda r: (r['Min']['X'], r['Min']['Y'], r['Max']['X'], r['Max']['Y'])
sources = read(root / 'build-sources.json')
assert len(sources) == 106 and all(sha(args.source_root / p) == h for p, h in sources.items())
reference = project / 'workplace/hd-prototype/catalog/remaining/consumers/battle/r14/native-turn3'
native = root / 'native-control'
current, before = read(native / 'receipt.json'), read(reference / 'receipt.json')
fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip', 'trace_tail',
          'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame', 'cd_pcm', 'mounted_disc',
          'mouse_input_events', 'input_boundary_frames', 'real_mode_history', 'real_mode_last',
          'composite_draw_trace']
assert all(current[k] == before[k] for k in fields), [k for k in fields if current[k] != before[k]]
assert sha(native / 'startup-memory.bin') == sha(reference / 'startup-memory.bin') == '28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d'
assert current['integration_input_sha256'] == sources and current['successful_steps'] == 600000000
assert current['fd2_startup_calls'] == current['native_runtime_instructions_skipped'] == 0
trace = current['graphics_draw_trace']
cache_kinds = ['cached_frame', 'cached_corner'] if args.corners else ['cached_frame']
old_trace = dict(trace, records=[r for r in trace['records'] if r['kind'] not in cache_kinds])
assert old_trace == before['graphics_draw_trace'], 'existing trace records changed'
cached = [r for r in trace['records'] if r['kind'] == 'cached_frame']
assert cached and all(r['verified'] for r in cached), 'normal cached frame was not verified'
for r in cached:
    assert r['resource'] == 'MAP_WAKU.OZM' and r['Return'] == 0x1ec31
    assert r['raw_helper_args'] == [r['raw_cache_base_linear'] + 0x3180, 0xa0005]
    assert r['raw_args'] == [0, 48, 88, 80, 0xa0005]
    assert rect(r['SourceRect']) == (0, 0, 384, 88) and rect(r['DestRect']) == (40, 0, 424, 88)
    assert r['ExpectedSHA'] == r['ActualSHA'] == '30ebc83c166b0cfa7f2666a5c71a63b98466f440dbb1315b88d56a1842bc2383'
corner_proof, corner_specs = None, {}
if args.corners:
    corner_proof = read(root/'map-cache-source-verification.json')
    assert corner_proof['result'] == 'PASS'
    assert sha(args.source_root/'tools/verify-hd-map-cache-source.py') == corner_proof['tool_sha256']
    corner_specs = {r['fourth_plane_return']: r for r in corner_proof['records']}
    assert len(corner_specs) == 8
    corners = [r for r in trace['records'] if r['kind'] == 'cached_corner']
    assert len(corners) == len(cached)*8
    assert len(cached)+len(corners) <= 1024 and len(old_trace['records']) <= 512
    for ret, spec in corner_specs.items():
        records = [r for r in corners if r['Return'] == ret]
        assert len(records) == len(cached)
        for r in records:
            assert r['verified'] and r['resource'] == 'MAP_WAKU.OZM'
            assert r['raw_helper_args'] == [r['raw_cache_base_linear']+spec['raw_plane_offsets'][3], spec['destination_linear']]
            assert r['raw_args'] == [88*80+spec['source_tile']*4,4,32,80,spec['destination_linear']]
            assert rect(r['SourceRect']) == tuple(spec['source_rect']) and rect(r['DestRect']) == tuple(spec['destination_rect'])
            assert r['ExpectedSHA'] == r['ActualSHA'] == spec['actual_reader_indices_sha256']
            assert r['source_crop_sha256'] == spec['static_source_crop_sha256']
            assert r['composite_source_sha256'] == corner_proof['source_indices_sha256']
            assert r['known_source_sha256'] == corner_proof['cached_lower_sha256']

cases = {}
for name in ['desktop-map', 'desktop-map-missing']:
    case = root / name
    exploration, receipt = read(case / 'exploration-status.json'), read(case / 'receipt.json')
    assert exploration['phase'] == 'finished' and not exploration['harness_only']
    assert receipt['stop'] == 'desktop_closed' and receipt['integration_input_sha256'] == sources
    assert receipt['fd2_startup_calls'] == receipt['native_runtime_instructions_skipped'] == 0
    assert exploration['input_data']['kind'] == 'original-game save'
    assert exploration['input_data']['dat_sha256'] == '3bf1827150e9955dd169c0d464b85a11a9ce2ff3e0764a7ecda4cb7a3678e1b5'
    assert len(receipt['mouse_input_schedule']) == 9
    assert receipt['mouse_input_schedule'] == current['mouse_input_schedule'][:9]
    assert receipt['mouse_input_events'][:9] == current['mouse_input_events'][:9]
    assert all(e['source'] == 'desktop' for e in receipt['mouse_input_events'][9:])
    assert sha(case / 'source-snapshot/tools/explore-battle.py') == exploration['tool_sha256']
    assert sha(case / 'tokugawa-desktop-prototype') == exploration['binary_sha256']
    captures = {}
    for capture in exploration['captures']:
        actual = case / Path(capture['actual']).relative_to('/output')
        presented = case / Path(capture['event']['frame']['path']).relative_to('/output')
        assert sha(actual) == capture['sha256'] and capture['window_pixels_equal']
        assert sha(presented) == capture['event']['frame']['png_sha256']
        image = Image.open(actual).convert('RGBA')
        if image.size == (1280, 864):
            image = image.crop((0, 64, 1280, 864))
        output = Image.open(presented).convert('RGBA')
        assert image.size == output.size == (1280, 800) and image.tobytes() == output.tobytes()
        capture['image'] = image
        capture['state'] = read(case / ('state-' + capture['label'] + '.json'))
        captures[capture['label']] = capture
    assert len(captures) == 3
    original, hd, restored = (captures[label] for label in ['saved-checkpoint', 'action-1', 'action-2'])
    assert original['event']['proof'] == hd['event']['proof'] == restored['event']['proof']
    assert original['image'].tobytes() == restored['image'].tobytes()
    toggles = [e for e in read(case / 'desktop-status.json')['events'] if e['kind'] == 'toggle']
    assert len(toggles) == 2 and all(e['guest_unchanged'] for e in toggles)
    reports = [r for r in hd['state']['graphics'] if r['resource'] == 'MAP_WAKU.OZM']
    assert len(reports) == (9 if args.corners else 1)
    assert all(r['mode'] == ('hd' if name == 'desktop-map' else 'fallback_missing_asset') for r in reports)
    main_report = next(r for r in reports if rect(r['rect']) == (40,0,424,88))
    assert hd['state']['terrain_scene']['reason'] == 'hd'
    assert hd['state']['terrain_scene']['floor_hd_count'] > 0
    cases[name] = dict(captures=captures, report=main_report, reports=reports, summary=dict(
        steps=receipt['successful_steps'], actual_window_captures=3, toggles=2,
        receipt_sha256=sha(case / 'receipt.json'), terrain_hd_count=hd['state']['terrain_scene']['floor_hd_count']))

normal, missing = cases['desktop-map']['captures'], cases['desktop-map-missing']['captures']
assert normal['saved-checkpoint']['image'].tobytes() == missing['saved-checkpoint']['image'].tobytes(), 'cross-case comparison requires identical complete native visible frame'
state = normal['action-1']['state']
draws = [d for d in state['graphics_observation']['verified_draws'] if d['resource'] == 'MAP_WAKU.OZM']
assert len(draws) == (9 if args.corners else 1) and all(d['verified'] for d in draws)
frame_draw = next(d for d in draws if d['kind'] == 'cached_frame')
assert rect(frame_draw['DestRect']) == (40, 0, 424, 88)
later = []
for key in ['terrain_history', 'terrain_sprites', 'window_draws', 'kao_draws']:
    for d in state[key] or []:
        completed = d['completed_step'] if key in ['window_draws', 'kao_draws'] else d['EndStep']
        if completed > frame_draw['EndStep']:
            bounds = rect(d['logical_rect'])
            assert 0 <= bounds[0] < bounds[2] <= 640 and 0 <= bounds[1] < bounds[3] <= 400
            later.append(bounds)
visible = [rect(r) for r in cases['desktop-map']['report']['visible_rectangles']]
for y in range(88):
    for x in range(40, 424):
        expected = not any(a <= x < c and b <= y < d for a, b, c, d in later)
        assert sum(a <= x < c and b <= y < d for a, b, c, d in visible) == int(expected)
indices = (project / 'workplace/hd-prototype/catalog/decoded/MAP_WAKU.OZM.idx').read_bytes()
assert hashlib.sha256(indices).hexdigest() == '1d0ee5b9b52142beb6813ae52f1db215e5224fae1935a64ea83fe07b97fe9a6b'
asset_path = project / 'assets/hd-b/art/remaining/prepared/MAP_WAKU.OZM-hd-v3.png'
assert sha(asset_path) == '87fbd4113e24f10aefcb02defbcaa1da3db61a9b155e3cdc221ab67121941f50'
asset = Image.open(asset_path).convert('RGBA')
base_path = project / 'workplace/hd-prototype/catalog/decoded/MAPGAMEN.OZM.idx'
assert sha(base_path) == '99a8497101f145c62d01934204f1c3ed74282fa6bd993fdbf4f8969eb0f7277c'
base_indices = base_path.read_bytes()
base_mask_path = project/'workplace/hd-prototype/catalog/battle-frame/frame-mask-v2.idx'
assert sha(base_mask_path) == 'b0fc8f0075d0259e59f665f44527487d93472c05cf64201104a4cc385d065eaf'
base_mask = base_mask_path.read_bytes()
base_art_path = project / 'assets/hd-b/battle-frame/battle-frame-hd-b-v2.png'
assert sha(base_art_path) == '3de79e3eebe01aa970e45b8bf377b6e381bec14c9b4d9f569994aaecbe78e435'
base_art = Image.open(base_art_path).convert('RGBA')
base_observation = state['battle_frame_observation']
assert base_observation['draw_completed'] and any(
    e.get('eligible') and e.get('native_indexed_sha256') == sha(base_path)
    for e in base_observation['events'])
native_palette = [(0,0,0),(32,32,32),(190,69,44),(81,20,20),(251,215,20),(44,81,0),
                  (20,20,227),(227,227,56),(0,0,0),(81,81,81),(227,190,93),(215,0,0),
                  (251,81,81),(44,215,44),(32,227,251),(251,251,251)]
enabled, fallback, original = normal['action-1']['image'], missing['action-1']['image'], missing['saved-checkpoint']['image']
pixel_plan, plan_counts, corner_checks = {}, {}, []
for d in draws:
    bounds, src = rect(d['DestRect']), rect(d['SourceRect'])
    own_later = []
    for key in ['terrain_history','terrain_sprites','window_draws','kao_draws']:
        for item in state[key] or []:
            step = item['completed_step'] if key in ['window_draws','kao_draws'] else item['EndStep']
            if step > d['EndStep']:
                own_later.append(rect(item['logical_rect']))
    report = next(r for r in cases['desktop-map']['reports'] if rect(r['rect']) == bounds)
    own_visible = [rect(r) for r in report['visible_rectangles']]
    actual = None
    if d['kind'] == 'cached_corner':
        spec = corner_specs[d['Return']]
        actual_path = root / ('map-cache-tile-%d.idx' % spec['source_tile'])
        assert sha(actual_path) == spec['actual_reader_indices_sha256']
        actual = actual_path.read_bytes()
    plan_counts[d['Return']] = 0
    for y in range(bounds[1], bounds[3]):
        for x in range(bounds[0], bounds[2]):
            sx, sy = src[0]+x-bounds[0], src[1]+y-bounds[1]
            static = indices[sy*384+sx]
            index = static if actual is None else actual[(y-bounds[1])*32+x-bounds[0]]
            visible = not any(a <= x < c and b <= y < e for a,b,c,e in own_later)
            assert sum(a <= x < c and b <= y < e for a,b,c,e in own_visible) == int(visible)
            assert y*640+x not in pixel_plan
            pixel_plan[y*640+x] = (sx,sy,index,visible,index == 8 or static == 8,d['Return'])
painted, retained, retained_hd, changed, source_black, fallback_base = 0, 0, 0, 0, 0, 0
for y in range(800):
    for x in range(1280):
        a, b = enabled.getpixel((x, y)), fallback.getpixel((x, y))
        plan = pixel_plan.get((y//2)*640+x//2)
        frame = plan is not None
        sx,sy,source_index,eligible,black,owner = plan if frame else (0,0,0,False,False,0)
        source_black += black
        if frame and eligible and not black:
            assert a == asset.getpixel((sx*2+x%2,sy*2+y%2)), ('approved frame pixel', x, y)
            assert original.getpixel((x, y))[:3] == native_palette[source_index]
            # 非黑來源與原生目的相同時，既有 MAPGAMEN 底圖仍有獨立資格。
            crest = 100 <= x//2 < 360 and 70 <= y//2 < 320
            expected = original.getpixel((x, y))
            if base_mask[(y//2)*640+x//2] and base_indices[(y//2)*640+x//2] == source_index and not crest:
                expected = base_art.getpixel((x, y))
                fallback_base += 1
            assert b == expected, ('individual missing frame/lower qualified layer', x, y)
            painted += 1
            plan_counts[owner] += 1
        else:
            assert a == b, ('underlay or unrelated HD changed', x, y)
            if frame:
                retained += 1
                retained_hd += a != original.getpixel((x, y))
        changed += a != b
assert painted > 0 and painted+retained == len(pixel_plan)*4
if not args.corners:
    assert source_black == 15906*4
else:
    for ret,spec in corner_specs.items():
        assert plan_counts[ret] > 0
        corner_checks.append(dict(fourth_plane_return=ret, source_tile=spec['source_tile'],
                                  native_rect=spec['destination_rect'], painted_output_pixels=plan_counts[ret]))
assert retained_hd > 0 and changed > 0
result = dict(status='PASS', classification='normal saved-game entry, cached upper 88 rows and eight lower tiles' if args.corners else 'normal saved-game entry, cached upper 88 rows only',
              source_count=len(sources), complete_original_ram_equal=True,
              original_fields_equal=len(fields), existing_graphics_trace_equal=True,
              added_cached_records=len(cached), cached_prefix_indices_sha256=cached[0]['ExpectedSHA'],
              frame_output_pixels=painted, retained_underlay_pixels=retained,
              original_source_black_output_pixels=source_black, later_scene_rectangles=len(later),
              retained_hd_output_pixels=retained_hd, individual_missing_changed_pixels=changed,
              missing_frame_qualified_base_output_pixels=fallback_base,
              missing_frame_policy='retain independently qualified MAPGAMEN HD underlay; native where no lower HD layer qualifies',
              upper_frame_output_pixels=plan_counts[0x1ec31], lower_corner_checks=corner_checks,
              added_corner_records=len(corners) if args.corners else 0,
              cases={k: v['summary'] for k, v in cases.items()},
              lower_cached_corners_verified=args.corners, whole_game_HD_complete=False,
              verification_tool_sha256=sha(Path(__file__)))
(root / 'verify-hd-map-frame.py').write_bytes(Path(__file__).read_bytes())
(root / 'verification-summary.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(result, ensure_ascii=False, indent=2))
