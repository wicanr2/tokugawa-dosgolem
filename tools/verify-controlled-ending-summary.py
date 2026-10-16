"""受控原版結局與 HD 的獨立核對；不宣稱正常通關。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image

assert Path('/.dockerenv').is_file() and os.getuid() == 1000
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
digest = lambda value: hashlib.sha256(value).hexdigest()
rect = lambda r: (r['Min']['X'], r['Min']['Y'], r['Max']['X'], r['Max']['Y'])
manifest = read(root/'validation-manifest.json')
assert manifest['validation_only'] and not manifest['normal_playthrough_verified']
assert manifest['entry_step'] == 40000000 and manifest['original_callsite_linear'] == 0x101a4
assert manifest['prepare_tool_sha256'] == sha(root/'prepare-ending-validation.py') == sha(project/'tools/prepare-ending-validation.py')
assert len(manifest['original_compiled_sources']) == len(manifest['controlled_compiled_sources']) == 106
for name, expected in manifest['original_compiled_sources'].items():
    assert sha(project/name) == expected
for name, expected in manifest['controlled_compiled_sources'].items():
    assert sha(root/'source-snapshot'/name) == expected
changed = [name for name, value in manifest['original_compiled_sources'].items()
           if manifest['controlled_compiled_sources'][name] != value]
assert changed == ['tools/startup-probe.go']
control = project/manifest['control_relative']
assert sha(control/'startup-memory.bin') == manifest['control_ram_sha256']
native = read(root/'native/receipt.json')
assert native['stop'] == 'dos_exit' and native['exit_code'] == 0
assert native['integration_input_sha256'] == manifest['controlled_compiled_sources']
entry = native['ending_validation']
assert entry['validation_only'] and not entry['normal_playthrough_verified'] and entry['step'] == 40000000
assert entry['after_eip'] == 0x101a4 and entry['ram_bytes_changed'] == 0
assert entry['ram_sha256_before'] == entry['ram_sha256_after'] == manifest['control_ram_sha256']
assert entry['finish_input_step'] == native['ending_validation_stages'][-1]['step']+1000000
assert native['mouse_input_events'] == [{'source':'controlled_validation_finish_input', 'actual_step':entry['finish_input_step'],
                                        'requested_x':320,'requested_y':300,'buttons':1}]
assert native['native_runtime_instructions_skipped'] and not native['normal_player_path_verified']
stages = native['ending_validation_stages']
assert 1 <= len(stages) <= 64 and len({stage['label'] for stage in stages}) == len(stages)
assert stages[-1]['label'] == 'END-11B-final-copy'
shift = [r for r in native['graphics_draw_trace']['records'] if r['kind'] == 'ending_shift']
assert len(shift) == 241 and [r.get('source_shift', 0) for r in shift] == list(range(241))
assert all(r['verified'] and r['ExpectedSHA'] == r['ActualSHA'] for r in shift)
sky = [r for r in native['graphics_draw_trace']['records'] if r['resource'] == 'END-2B.OZM']
assert len(sky) == 8 and all(r['verified'] for r in sky)
assert {rect(r['SourceRect']) for r in sky} == {(x, y, x+256, y+84) for x in [0, 256] for y in [0, 84, 168, 252]}
story = {r['resource']: r for r in read(project/'workplace/hd-prototype/catalog/story-expanded/profiles.json')['records']}
art = {r['name']: r for r in read(project/'assets/hd-b/art/remaining/profiles.json')['records']}
assets = {}


def sized_asset(name, profile, width, height):
    if name not in assets:
        path = project/'assets/hd-b/art'/profile['asset']
        assert sha(path) == profile['asset_sha256']
        original = Image.open(path).convert('RGBA')
        ow, oh = original.size
        image = Image.new('RGBA', (width*2, height*2))
        pixels = original.load()
        image.putdata([pixels[x*ow//(width*2), y*oh//(height*2)]
                       for y in range(height*2) for x in range(width*2)])
        assets[name] = image
    return assets[name]


def expected_hd(base, state, native_indices, mode):
    expected = base.copy()
    painted = {}
    if state['portrait_mode'] == 'hd':
        name = state['portrait_resource']
        assert name in story and name.startswith('END-')
        profile = story[name]
        image = sized_asset(name, profile, 320, 150)
        destination=rect(state['portrait_rect'])
        assert destination in [(160,50,480,200),(16,16,336,166)]
        left,top=destination[0]*2,destination[1]*2
        expected.paste(image, (left, top))
        for x1, y1, x2, y2 in profile['label_rects']:
            area = (left+x1*2, top+y1*2, left+x2*2, top+y2*2)
            expected.paste(base.crop(area), area[:2])
        labels = {(x, y) for x1, y1, x2, y2 in profile['label_rects']
                  for y in range(y1, y2) for x in range(x1, x2)}
        painted[name] = (320*150-len(labels))*4
    reports = state['graphics'] or []
    draws = state['graphics_observation']['verified_draws'] or []
    assert len(reports) == len(draws)
    for report, draw in zip(reports, draws):
        name = draw['resource']
        assert report['resource'] == name
        if name == 'END-2.OZM' and mode == 'missing-end2':
            assert report['mode'] == 'fallback_missing_asset'
        if report['mode'] != 'hd':
            continue
        assert name in art and name in ['END-2.OZM', 'END-2B.OZM', 'END-11.OZM', 'END-11B.OZM']
        profile = art[name]
        width, height = profile['source']['width'], profile['source']['height']
        source = (project/'workplace/hd-prototype/catalog/decoded'/(name+'.idx')).read_bytes()
        assert len(source) == width*height and digest(source) == profile['source']['indices_sha256']
        image = sized_asset(name, profile, width, height)
        sx1, sy1, sx2, sy2 = rect(draw['SourceRect'])
        dx1, dy1, dx2, dy2 = rect(draw['DestRect'])
        shift_count = draw.get('source_shift', 0) if draw['kind'] == 'ending_shift' else 0
        visible = report['visible_rectangles']
        count = 0
        px1, py1, px2, py2 = profile['paint_rect']
        for bounds in visible:
            x1, y1, x2, y2 = rect(bounds)
            for y in range(y1, y2):
                sx, sy = sx1+x1-dx1+shift_count, sy1+y-dy1
                assert native_indices[y*640+x1:y*640+x2] == source[sy*width+sx:sy*width+sx+x2-x1]
            left, top = max(x1, dx1+px1-sx1-shift_count), max(y1, dy1+py1-sy1)
            right, bottom = min(x2, dx1+px2-sx1-shift_count), min(y2, dy1+py2-sy1)
            if left < right and top < bottom:
                sx, sy = sx1+left-dx1+shift_count, sy1+top-dy1
                area = (sx*2, sy*2, (sx+right-left)*2, (sy+bottom-top)*2)
                expected.paste(image.crop(area), (left*2, top*2))
                count += (right-left)*(bottom-top)*4
        assert count == report['painted_native_pixels']*4 and count > 0
        painted[name] = painted.get(name, 0)+count
    assert painted or (mode == 'missing-end2' and any(
        r['resource'] == 'END-2.OZM' and r['mode'] == 'fallback_missing_asset' for r in reports)), 'selected ending state did not present any verified HD'
    return expected, painted


cases = {}
for mode in ['hd', 'missing-end2']:
    case = root/('desktop-'+mode)
    capture_report = read(case/'controlled-ending-captures.json')
    receipt = read(case/'receipt.json')
    assert capture_report['status'] == 'PASS_CAPTURE' and capture_report['validation_only']
    assert not capture_report['normal_playthrough_verified']
    assert capture_report['tool_sha256'] == sha(case/'source-snapshot/tools/verify-controlled-ending.py') == sha(project/'tools/verify-controlled-ending.py')
    assert capture_report['receipt_sha256'] == sha(case/'receipt.json')
    assert capture_report['binary_sha256'] == sha(root/'tokugawa-desktop-prototype') == sha(case/'tokugawa-desktop-prototype')
    assert receipt['stop'] == 'dos_exit' and receipt['exit_code'] == 0
    assert receipt['integration_input_sha256'] == manifest['controlled_compiled_sources']
    assert receipt['ending_validation'] == entry and receipt['ending_validation_stages'] == stages
    assert capture_report['stages'] == stages
    assert receipt['successful_steps'] == native['successful_steps']
    events=receipt['desktop']['hd_observation']['draw_events']
    callers={0x18a07:'END-4.OZM',0x18a6e:'END-5.OZM',0x18ad5:'END-6.OZM',0x18b3c:'END-7.OZM',
             0x18ba3:'END-8.OZM',0x18c19:'END-9.OZM',0x18c8f:'END-10.OZM'}
    for ret,name in callers.items():
        enters=[event for event in events if event['kind']=='draw_enter' and event['return_linear']==ret]
        assert len(enters)==1 and enters[0]['source_name']==name and enters[0]['logical_rect']==[16,16,320,150]
        assert any(event['kind']=='draw_return' and event['linear_eip']==ret and event['eligible'] for event in events)
        if ret in [0x18c19,0x18c8f]:
            assert enters[0]['raw_plane_destinations']==[0xa0502]*4
    assert sha(case/'startup-memory.bin') == sha(root/'native/startup-memory.bin')
    for field in ['trace_tail', 'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'cd_pcm', 'graphics_draw_trace', 'frame']:
        assert receipt[field] == native[field], field
    groups = {}
    for capture in capture_report['captures']:
        groups.setdefault(capture['stage_id'], {})[capture['mode']] = capture
    checked = []
    for stage_id, group in groups.items():
        assert set(group) == {'original', 'hd', 'restored'}
        stage = stages[stage_id-1]
        images, states = {}, {}
        for kind, capture in group.items():
            path = case/Path(capture['actual']).relative_to('/output')
            assert sha(path) == capture['actual_sha256'] and capture['window_pixels_equal']
            images[kind] = Image.open(path).convert('RGBA')
            expected_path = case/Path(capture['event']['frame']['path']).relative_to('/output')
            assert sha(expected_path) == capture['event']['frame']['png_sha256']
            assert images[kind].size == (1280, 800) and images[kind].tobytes() == Image.open(expected_path).convert('RGBA').tobytes()
            proof = capture['event']['proof']
            assert proof['step'] == stage['step'] and proof['eip'] == stage['eip'] and proof['ram_sha256'] == stage['ram_sha256']
            assert proof['registers'] == stage['registers'] and proof['segments'] == stage['segments'] and proof['flags'] == stage['flags']
            assert proof['raw_indexed_sha256'] == stage['indices_sha256']
            states[kind] = read(case/capture['state_path'])
        assert group['original']['event']['proof'] == group['hd']['event']['proof'] == group['restored']['event']['proof']
        assert images['original'].tobytes() == images['restored'].tobytes()
        native_path = root/'native'/Path(stage['frame']['path']).relative_to('workplace')
        native_image = Image.open(native_path).convert('RGBA').resize((1280, 800), Image.Resampling.NEAREST)
        assert native_image.tobytes() == images['original'].tobytes()
        indices = (case/stage['indices_path']).read_bytes()
        assert digest(indices) == stage['indices_sha256']
        expected, painted = expected_hd(images['original'], states['hd'], indices, mode)
        assert expected.tobytes() == images['hd'].tobytes(), ('independent HD pixels', mode, stage['label'], painted)
        if stage['label'] == 'END-11B-final-copy' or stage['label'].startswith('END-11.OZM'):
            assert images['hd'].crop((0, 798, 1280, 800)).tobytes() == images['original'].crop((0, 798, 1280, 800)).tobytes()
        checked.append(dict(stage_id=stage_id, label=stage['label'], painted_output_pixels=painted))
    toggles = [event for event in read(case/'desktop-status.json')['events'] if event['kind'] == 'toggle']
    assert len(toggles) == len(groups)*2 and all(event['guest_unchanged'] for event in toggles)
    cases[mode] = dict(receipt_sha256=sha(case/'receipt.json'), captures=len(capture_report['captures']),
                       toggles=len(toggles), checkpoints=checked)
hd_checks = cases['hd']['checkpoints']
assert {r['label'] for r in hd_checks} >= {'END-2-shift-000', 'END-2-shift-120', 'END-2-shift-240', 'END-11B-final-copy'}
assert sum(r['label'].startswith('END-2B.OZM') for r in hd_checks) == 8
assert any(r['label'].startswith('END-11.OZM') for r in hd_checks)
controlled_sources = sorted({name for r in hd_checks for name in r['painted_output_pixels']})
assert set(controlled_sources)=={f'END-{n}.OZM' for n in [1,2,4,5,6,7,8,9,10,11]}|{'END-2B.OZM','END-11B.OZM'}
summary = dict(status='PASS', acceptance='user-authorized controlled original ending GUI',
               controlled_ending_GUI_verified=True, normal_playthrough_verified=False, whole_game_HD_complete=False,
               entry_step=40000000, ram_bytes_changed_by_entry=0, original_exit_code=0,
               successful_steps=native['successful_steps'], complete_ram_equal_across_cases=True,
               same_state_checkpoints=len(stages), ending_shift_returns_verified=241, sky_frames_verified=8,
               controlled_sources=controlled_sources, cases=cases, tool_sha256=sha(Path(__file__)),
               prepare_tool_sha256=manifest['prepare_tool_sha256'], GUI_tool_sha256=sha(project/'tools/verify-controlled-ending.py'),
               manifest_sha256=sha(root/'validation-manifest.json'), native_receipt_sha256=sha(root/'native/receipt.json'))
(root/'verification-summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
print(json.dumps(summary, ensure_ascii=False))
