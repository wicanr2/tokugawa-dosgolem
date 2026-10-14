"""以正常路徑的原始返回事件核對完整地面順序及可見碎片。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rect(draw):
    return draw['rect']


def contains(r, x, y):
    return r[0] <= x < r[2] and r[1] <= y < r[3]


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, output = Path('/project'), Path('/output')
    terrain = project/'workplace/hd-prototype/catalog/terrain'
    trace_path = terrain/'objects/trace/startup-probe/latest.json'
    trace = json.loads(trace_path.read_text())
    events = trace['terrain_draw_trace']['events']
    assert trace['terrain_draw_trace']['dropped_events'] == 0
    enters = {e['step']: e for e in events if e['kind'].endswith('_enter')}
    records = []
    for e in events:
        if not e['kind'].endswith('_return'):
            continue
        entry = enters[e['begin_step']]
        assert entry['raw_source_offset'] == e['raw_source_offset'] and entry['raw_destination_offset'] == e['raw_destination_offset']
        sx, sy = e['raw_source_offset'] % 80, e['raw_source_offset'] // 80
        x, y = e['raw_destination_offset'] % 80 * 8, e['raw_destination_offset'] // 80
        assert sx < 78 and sx % 6 == 0 and x+48 <= 640
        if e['kind'] == 'floor_return':
            assert sy < 256 and sy % 32 == 0 and y+32 <= 400
            group, index, height = 'floor', sy//32*13+sx//6, 32
        else:
            assert y+48 <= 400
            if entry['return_linear'] == 0x1fe69:
                assert sy in (256, 304)
                group, index = 'objects', (sy-256)//48*13+sx//6
            else:
                assert entry['return_linear'] == 0x1fea2 and sy == 352
                group, index = 'units', sx//6
            height = 48
        records.append({'group': group, 'index': index, 'rect': [x, y, x+48, y+height],
                        'order': len(records)+1, 'begin_step': entry['step'], 'end_step': e['step']})
    floors = [r for r in records if r['group'] == 'floor']
    sprites = [r for r in records if r['group'] != 'floor']
    assert len(records) == 145 and len(floors) == 134 and len(sprites) == 11
    cache = {}
    def image_for(d):
        key = d['group'], d['index']
        if key not in cache:
            name = f"floor-{d['index']:03d}-original.png" if d['group'] == 'floor' else f"{d['group']}-{d['index']:03d}-original.png"
            path = terrain/('prepared/sources' if d['group'] == 'floor' else 'objects/sources')/name
            cache[key] = Image.open(path).convert('RGBA')
        return cache[key]
    background_path = terrain/'backgrounds/startup-probe/latest.json'
    background = json.loads(background_path.read_text())
    background_images = {}
    for d in sprites:
        matches = [r for r in background['sprite_backdrop_records'] if r['begin_step'] == d['begin_step']]
        assert len(matches) == 1 and matches[0]['primitive_verified']
        background_images[d['order']] = Image.open(terrain/'backgrounds'/matches[0]['before']['path']).convert('RGBA')
    approved_keys = {('objects', 0), ('units', 1), ('units', 3)}
    candidates = [d for d in sprites if (d['group'], d['index']) in approved_keys]
    width = 640
    base = [-1]*(640*400)
    owner = [0]*(640*400)
    baseline_order = [0]*(640*400)
    pixels = [None]*(640*400)
    for i, d in enumerate(floors):
        im = image_for(d)
        for y in range(d['rect'][1], d['rect'][3]):
            for x in range(d['rect'][0], d['rect'][2]):
                n = y*width+x
                base[n], baseline_order[n], owner[n] = i, d['order'], d['order']
                pixels[n] = im.getpixel((x-d['rect'][0], y-d['rect'][1]))
    for d in candidates:
        im = background_images[d['order']]
        for y in range(d['rect'][1], d['rect'][3]):
            for x in range(d['rect'][0], d['rect'][2]):
                n = y*width+x
                if pixels[n] is None:
                    pixels[n] = im.getpixel((x-d['rect'][0], y-d['rect'][1]))
                    baseline_order[n] = d['order']-1
    for d in sprites:
        im = image_for(d)
        for y in range(d['rect'][1], d['rect'][3]):
            for x in range(d['rect'][0], d['rect'][2]):
                n = y*width+x
                if pixels[n] is None or d['order'] <= baseline_order[n]:
                    continue
                c = im.getpixel((x-d['rect'][0], y-d['rect'][1]))
                if c[3] == 255:
                    pixels[n], owner[n] = c, d['order']
    baseline_path = project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    baseline = json.loads(baseline_path.read_text())
    frame_path = project/'workplace/startup-probe'/Path(baseline['frame']['path']).name
    native = Image.open(frame_path).convert('RGBA')
    cursor = baseline['mouse_cursor']; mouse = baseline['mouse_input_events'][-1]
    cx, cy = mouse['x']-cursor['HotX'], mouse['y']-cursor['HotY']
    def is_cursor(x, y):
        return cx <= x < cx+16 and cy <= y < cy+16
    findings = []
    for d in records:
        mismatches = excluded = visible = 0
        for y in range(d['rect'][1], d['rect'][3]):
            for x in range(d['rect'][0], d['rect'][2]):
                n = y*width+x
                visible += (base[n] >= 0 and floors[base[n]]['order'] == d['order']) if d['group'] == 'floor' else owner[n] == d['order']
                if is_cursor(x, y):
                    excluded += 1
                else:
                    mismatches += pixels[n] is None or pixels[n] != native.getpixel((x, y))
        findings.append({**d, 'current_visible_native_pixels': visible,
                         'full_rectangle_non_cursor_mismatches': mismatches, 'cursor_pixels_excluded': excluded,
                         'normal_source_scene_matches': visible > 0 and mismatches == 0})
    matching_floors = [f for f in findings if f['group'] == 'floor' and f['normal_source_scene_matches']]
    matching_sprites = [f for f in findings if f['group'] != 'floor' and f['normal_source_scene_matches']]
    assert len(matching_sprites) == 6
    result = {'classification': 'original full draw journal and source scene; cursor-excluded research, not HD GUI acceptance',
              'whole_game_HD_complete': False, 'source_trace_sha256': sha(trace_path), 'source_background_receipt_sha256': sha(background_path),
              'native_frame_sha256': sha(frame_path), 'baseline_sha256': sha(baseline_path),
              'native_floor_draws': 134, 'native_sprite_draws': 11,
              'matching_visible_floor_draws': len(matching_floors), 'matching_visible_sprite_draws': len(matching_sprites),
              'records': findings, 'tool_sha256': sha(Path(__file__))}
    target = output/'source-scene-verification.json'
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'verified': True, 'matching_floor_draws': len(matching_floors), 'matching_sprite_draws': len(matching_sprites), 'summary_sha256': sha(target)}))


if __name__ == '__main__':
    main()
