"""核對正常交鋒的 HD 合成及共用視窗還原；原版收據不改寫。"""
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
parser.add_argument('--units', action='store_true')
args = parser.parse_args()
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
rect = lambda r: (r['Min']['X'], r['Min']['Y'], r['Max']['X'], r['Max']['Y'])
inside = lambda p, r: r[0] <= p[0] < r[2] and r[1] <= p[1] < r[3]
image_at = lambda p: Image.open(p).convert('RGBA')
sources = read(root / 'build-sources.json')
assert len(sources) == 106 and all(sha(args.source_root / p) == h for p, h in sources.items())
native = root / 'native-control'
reference = project / 'workplace/hd-prototype/catalog/remaining/consumers/battle/r14/native-turn3'
n, before = read(native / 'receipt.json'), read(reference / 'receipt.json')
fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip', 'trace_tail',
          'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame', 'cd_pcm', 'mounted_disc',
          'mouse_input_events', 'input_boundary_frames', 'real_mode_history', 'real_mode_last',
          'graphics_draw_trace', 'composite_draw_trace']
assert all(n[k] == before[k] for k in fields), [k for k in fields if n[k] != before[k]]
assert n['integration_input_sha256'] == sources and n['successful_steps'] == 600000000
assert n['fd2_startup_calls'] == n['native_runtime_instructions_skipped'] == 0
assert sha(native / 'startup-memory.bin') == sha(reference / 'startup-memory.bin') == '28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d'
cases = {}
for name in ['desktop', 'desktop-animation', 'desktop-missing'] + (['desktop-unit-missing'] if args.units else []):
    case = root / name
    if (case / 'ready/exploration-status.json').is_file():
        case = case / 'ready'
    e, r = read(case / 'exploration-status.json'), read(case / 'receipt.json')
    assert e['phase'] == 'finished' and not e['harness_only'] and r['stop'] == 'desktop_closed'
    assert r['integration_input_sha256'] == sources and r['mouse_input_schedule'] == n['mouse_input_schedule']
    assert r['mouse_input_events'][:19] == n['mouse_input_events']
    assert all(v['source'] == 'desktop' for v in r['mouse_input_events'][19:])
    assert r['input_boundary_frames'] == n['input_boundary_frames']
    assert r['fd2_startup_calls'] == r['native_runtime_instructions_skipped'] == 0
    assert sha(case / 'source-snapshot/tools/explore-battle.py') == e['tool_sha256']
    assert sha(case / 'tokugawa-desktop-prototype') == e['binary_sha256']
    assert e['input_data']['kind'] == 'original-game save'
    assert e['input_data']['dat_sha256'] == '3bf1827150e9955dd169c0d464b85a11a9ce2ff3e0764a7ecda4cb7a3678e1b5'
    captures = {}
    for capture in e['captures']:
        actual = case / Path(capture['actual']).relative_to('/output')
        presented = case / Path(capture['event']['frame']['path']).relative_to('/output')
        assert sha(actual) == capture['sha256'] and capture['window_pixels_equal']
        a, b = image_at(actual), image_at(presented)
        if a.size == (1280, 864):
            a = a.crop((0, 64, 1280, 864))
        assert a.size == b.size == (1280, 800) and a.tobytes() == b.tobytes()
        assert sha(presented) == capture['event']['frame']['png_sha256']
        capture['image'] = a
        capture['status'] = read(case / ('state-' + capture['label'] + '.json'))
        captures[capture['label']] = capture
    status = read(case / 'desktop-status.json')
    toggles = [v for v in status['events'] if v['kind'] == 'toggle']
    assert toggles and all(v['guest_unchanged'] for v in toggles)
    cases[name] = dict(captures=captures, status=status, receipt=r, exploration=e,
                       report=dict(artifact_root=str(case.relative_to(root)), actual_window_captures=len(captures), toggles=len(toggles),
                                   steps=r['successful_steps'], receipt_sha256=sha(case / 'receipt.json')))

animation = cases['desktop-animation']['captures']
original, hd = animation['saved-checkpoint'], animation['action-1']
assert original['event']['proof'] == hd['event']['proof']
reports = {v['resource']: v for v in hd['status']['graphics']}
assert all(reports[name]['mode'] == 'hd' for name in ['BATTLE02.OZM', 'BATTLE.OZM', 'JIMEN_0.OZM', 'JIMEN_1.OZM'])
assert hd['status']['kao_hd_count'] == 2
assert len(hd['status']['kao_draws']) == 2, 'closed ability portrait remains active'
mixed = before['composite_draw_trace']['records'][0]
source = (reference / Path(mixed['SourcePath']).relative_to('/output')).read_bytes()
result = (reference / Path(mixed['AfterPath']).relative_to('/output')).read_bytes()
mixed_positions = [(i % 640, i // 640) for i, value in enumerate(source) if value and value != result[i]]
assert len(mixed_positions) == mixed['MixedOR'] == 18
for x, y in mixed_positions:
    assert original['image'].crop((x*2, y*2, x*2+2, y*2+2)).tobytes() == hd['image'].crop((x*2, y*2, x*2+2, y*2+2)).tobytes()
foreground_regions = [(176, 52, 464, 148), (24, 170, 136, 216), (504, 170, 616, 216)]
unit_regions = [(240, 56, 288, 112), (400, 56, 448, 112)]
foreground = 0
for x0, y0, x1, y1 in foreground_regions:
    for y in range(y0, y1):
        for x in range(x0, x1):
            if any(inside((x, y), r) for r in unit_regions):
                continue
            pixel = original['image'].getpixel((x*2, y*2))
            if pixel == (0, 0, 0, 255) and source[y*640+x] != 0:
                assert all(hd['image'].getpixel((x*2+dx, y*2+dy)) == pixel for dy in range(2) for dx in range(2))
                foreground += 1
assert foreground >= 1000
unit_checks = []
if args.units:
    profiles_path = project / 'workplace/hd-prototype/catalog/terrain/units/profiles.json'
    profiles = {r['index']: r for r in read(profiles_path)['records']}
    assert hd['status']['combat_unit_hd_count'] == 2 and hd['status']['combat_unit_modes'] == ['hd', 'hd']
    units = [r for r in hd['status']['terrain_sprites'] if r.get('combat_return_linear') == 0x1f9be]
    assert [r['index'] for r in units] == [7, 3]
    paper = image_at(project / 'assets/hd-b/art/remaining/prepared/BATTLE.OZM-hd-v3.png')
    paper_indices = (project / 'workplace/hd-prototype/catalog/decoded/BATTLE.OZM.idx').read_bytes()
    atlas_indices = (project / 'workplace/hd-prototype/catalog/decoded/MAP_CHIP.OZM.idx').read_bytes()
    for unit, x in zip(units, [240, 400]):
        assert rect(unit['logical_rect']) == (x, 60, x+48, 108) and unit['background_verified']
        profile = profiles[unit['index']]
        assert unit['source_indexed_sha256'] == profile['source_indices_sha256']
        source_indices = b''.join(atlas_indices[y*640+unit['index']*48:y*640+unit['index']*48+48] for y in range(352, 400))
        background_indices = b''.join(paper_indices[y*640+x:y*640+x+48] for y in range(60, 108))
        after_indices = bytes(a or b for a, b in zip(source_indices, background_indices))
        assert hashlib.sha256(source_indices).hexdigest() == unit['source_indexed_sha256']
        assert hashlib.sha256(background_indices).hexdigest() == unit['background_indexed_sha256']
        assert hashlib.sha256(after_indices).hexdigest() == unit['after_indexed_sha256']
        source_path = project / profile['source']
        assert sha(source_path) == profile['source_png_sha256']
        prepared_path = profiles_path.parent / profile['prepared']
        assert sha(prepared_path) == profile['prepared_sha256']
        asset = image_at(prepared_path)
        assert asset.size == (96, 96)
        for y in range(96):
            for dx in range(96):
                back, front = paper.getpixel((x*2+dx, 120+y)), asset.getpixel((dx, y))
                alpha = front[3]
                expected = tuple((front[i]*alpha + back[i]*(255-alpha) + 127)//255 for i in range(3)) + (255,)
                assert hd['image'].getpixel((x*2+dx, 120+y)) == expected, ('unit alpha', unit['index'], dx, y)
        unit_checks.append(dict(index=unit['index'], native_rect=[x, 60, x+48, 108], prepared_sha256=sha(prepared_path)))
else:
    for bounds in unit_regions:
        area = tuple(v*2 for v in bounds)
        assert original['image'].crop(area).tobytes() == hd['image'].crop(area).tobytes()

animated = animation['action-2']
assert any(v['resource'] == 'ANIME08.OZM' and v['mode'] == 'hd' for v in animated['status']['graphics'])
assert animated['event']['proof'] == animation['action-3']['event']['proof'] == animation['action-4']['event']['proof']
if args.units:
    assert animated['status']['combat_unit_hd_count'] == 0
    assert animated['status']['combat_unit_modes'] == ['fallback_occluded', 'fallback_occluded']
    for x in [240, 400]:
        area = (x*2, 120, (x+48)*2, 200)
        assert animated['image'].crop(area).tobytes() == animation['action-3']['image'].crop(area).tobytes()
restored = animation['action-5']
assert any(v['kind'] == 'restore_return' and v['verified'] and rect(v['logical_rect']) == (216, 100, 424, 260)
           for v in restored['status']['window_layers']['events'])
assert not any(v['resource'] == 'ANIME08.OZM' for v in restored['status']['graphics'])
assert any(v['resource'] == 'BATTLE.OZM' and v['mode'] == 'hd' for v in restored['status']['graphics'])
restore_region = (432, 200, 848, 520)
assert original['image'].crop(restore_region).tobytes() == animation['action-6']['image'].crop(restore_region).tobytes(), 'original animation region changed before comparison'
assert hd['image'].crop(restore_region).tobytes() == restored['image'].crop(restore_region).tobytes(), 'HD animation region did not restore'
if args.units:
    assert restored['status']['combat_unit_hd_count'] == 2 and restored['status']['combat_unit_modes'] == ['hd', 'hd']

enabled = cases['desktop']['captures']['action-1']
missing = cases['desktop-missing']['captures']['action-1']
missing_original = cases['desktop-missing']['captures']['saved-checkpoint']
if args.units:
    assert enabled['event']['proof'] == cases['desktop']['captures']['saved-checkpoint']['event']['proof']
    assert missing['event']['proof'] == missing_original['event']['proof']
    assert cases['desktop']['captures']['saved-checkpoint']['image'].tobytes() == missing_original['image'].tobytes(), 'missing ground comparison requires identical full native visible frame'
else:
    assert enabled['event']['proof'] == missing['event']['proof'] == missing_original['event']['proof']
missing_reports = {v['resource']: v for v in missing['status']['graphics']}
assert missing_reports['JIMEN_1.OZM']['mode'] == 'fallback_missing_asset'
assert all(missing_reports[name]['mode'] == 'hd' for name in ['BATTLE02.OZM', 'BATTLE.OZM', 'JIMEN_0.OZM'])
tile = (project / 'workplace/hd-prototype/catalog/decoded/JIMEN_1.OZM.idx').read_bytes()
asset = image_at(project / 'assets/hd-b/art/remaining/prepared/JIMEN_1.OZM-hd-v3.png')
changed, native_fallback = 0, 0
for y in range(800):
    for x in range(1280):
        sx, sy = x//2-288, y//2-302
        allowed = 0 <= sx < 352 and 0 <= sy < 64 and tile[sy*352+sx] != 0
        a, b = enabled['image'].getpixel((x, y)), missing['image'].getpixel((x, y))
        if allowed:
            assert a == asset.getpixel((x-576, y-604))
            assert b == missing_original['image'].getpixel((x, y))
            native_fallback += 1
        else:
            assert a == b, 'missing right ground changed another HD layer'
        changed += a != b
assert native_fallback == 20575*4 and changed > 0
unit_missing_changed = 0
if args.units:
    missing_unit = cases['desktop-unit-missing']['captures']['action-1']
    missing_unit_original = cases['desktop-unit-missing']['captures']['saved-checkpoint']
    assert missing_unit['event']['proof'] == missing_unit_original['event']['proof']
    assert original['image'].tobytes() == missing_unit_original['image'].tobytes(), 'unit missing comparison requires identical full native visible frame'
    assert original['status']['terrain_sprites'] == missing_unit_original['status']['terrain_sprites']
    assert missing_unit['status']['combat_unit_modes'] == ['fallback_missing_asset', 'hd']
    assert missing_unit['status']['combat_unit_hd_count'] == 1
    for y in range(800):
        for x in range(1280):
            a, b = hd['image'].getpixel((x, y)), missing_unit['image'].getpixel((x, y))
            if inside((x, y), (480, 120, 576, 216)):
                assert b == missing_unit_original['image'].getpixel((x, y))
                unit_missing_changed += a != b
            else:
                assert a == b, 'missing unit changed another HD layer'
    assert unit_missing_changed > 0
report = dict(result='PASS', scope='normal battle composites, animation and region restoration; top unit icons remain original',
              source_count=106, native_steps=600000000, native_fields_equal=fields,
              native_ram_sha256=sha(native / 'startup-memory.bin'), source_reference_receipt_sha256=sha(reference / 'receipt.json'),
              cases={name: case['report'] for name, case in cases.items()}, mixed_OR_native_pixels_preserved=18,
              original_black_foreground_pixels_preserved=foreground, missing_ground_native_output_pixels=native_fallback,
              missing_ground_changed_output_pixels=changed, animation_restore_region=list(restore_region),
              tool_sha256=sha(Path(__file__)), python=sys.version.split()[0])
if args.units:
    report.update(scope='normal battle composites, top unit icons, individual missing art, animation occlusion and region restoration',
                  combat_unit_checks=unit_checks, missing_unit_changed_output_pixels=unit_missing_changed,
                  missing_unit_comparison=dict(classification='identical full original visible frame and source draws; per-case toggles preserve complete guest proof',
                                               normal_step=original['status']['step'], missing_step=missing_unit_original['status']['step']))
(root / 'verification-summary.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
print(json.dumps({k: report[k] for k in ['result', 'source_count', 'mixed_OR_native_pixels_preserved', 'original_black_foreground_pixels_preserved', 'missing_ground_native_output_pixels']}))
