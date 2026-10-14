"""依來源與 READY alpha 契約，獨立計算 scene GUI 的預期呈現。"""
import hashlib
import json
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def bounds(draw):
    r = draw['logical_rect']
    return r['Min']['X'], r['Min']['Y'], r['Max']['X'], r['Max']['Y']


def inside(rect, x, y):
    return rect[0] <= x < rect[2] and rect[1] <= y < rect[3]


def blend(back, front):
    a = front[3]
    return tuple((f*a+b*(255-a)+127)//255 for b, f in zip(back[:3], front[:3]))+(255,)


def verify_scene_pixels(snapshot, enhanced, original, mode):
    project = Path('/project')
    terrain = project/'workplace/hd-prototype/catalog/terrain'
    report = snapshot['terrain_scene']
    floors, sprites = snapshot['terrain_history'], snapshot['terrain_sprites']
    source = json.loads((terrain/'scene/source-scene-verification.json').read_text())
    assert len(floors) == 134 and len(sprites) == 11
    for group, draws in [('floor', floors), ('sprite', sprites)]:
        expected = [r for r in source['records'] if (r['group'] == 'floor') == (group == 'floor')]
        assert len(expected) == len(draws)
        for d, e in zip(draws, expected):
            assert (d['index'], list(bounds(d)), d['draw_order']) == (e['index'], e['rect'], e['order'])
            if group != 'floor':
                assert d['group'] == e['group']
    fm, sm = report['floor_modes'], report['sprite_modes']
    active_floors = {i for i, m in enumerate(fm) if m == 'hd'}
    active_sprites = {d['draw_order']: i for i, (d, m) in enumerate(zip(sprites, sm)) if m == 'hd'}
    assert len(active_floors) == report['floor_hd_count'] == (67 if mode == 'scene' else 0)
    assert sum(sprites[i]['group'] == 'units' for i in active_sprites.values()) == report['unit_hd_count'] == (0 if mode == 'scene-missing' else 2)
    assert sum(sprites[i]['group'] == 'objects' for i in active_sprites.values()) == report['tree_hd_count'] == (0 if mode == 'scene-missing' else 4)
    native_cache = {}
    def native_image(group, index):
        key = group, index
        if key not in native_cache:
            name = f'floor-{index:03d}-original.png' if group == 'floor' else f'{group}-{index:03d}-original.png'
            native_cache[key] = Image.open(terrain/('prepared/sources' if group == 'floor' else 'objects/sources')/name).convert('RGBA')
        return native_cache[key]
    floor_profiles = {r['index']: r for r in json.loads((terrain/'prepared/profiles.json').read_text())['records']}
    hd_floors = {}
    for i in active_floors:
        index = floors[i]['index']
        if index in hd_floors:
            continue
        p = floor_profiles[index]
        path = project/'assets/hd-b/terrain'/p['asset']
        assert sha(path) == p['asset_sha256']
        im = Image.open(path).convert('RGBA')
        hd = Image.new('RGBA', (96, 64))
        for y in range(64):
            for x in range(96):
                hd.putpixel((x, y), im.getpixel((x*im.width//96, y*im.height//64)))
        hd_floors[index] = hd
    selected = {
        ('objects', 0): ('object-000-hd-b-v2.png', 'f6885ec6d1d557973fdb304fca537e095eec4c65e37439340ef2c39da2d94cd2', (0,0,47,47), (7,0,1247,1245)),
        ('units', 1): ('unit-001-hd-b-v1.png', 'a8c37f61ef22f645a58cb824758ab34620e535a2588f0fce5e71490cf3dd3a72', (2,0,47,48), (265,39,1098,1191)),
        ('units', 3): ('unit-003-hd-b-v1.png', '6c9a0ef3d9a3db01bec673fb37d7f8fca4a2f2a08cf6a14d81ecd101ec867654', (0,0,41,48), (117,90,984,1159)),
    }
    hd_sprites = {}
    for i in active_sprites.values():
        d = sprites[i]; key = d['group'], d['index']
        if key in hd_sprites:
            continue
        name, digest, box, crop = selected[key]
        path = project/'assets/hd-b/terrain'/name
        assert sha(path) == digest and native_image(*key).getbbox() == box
        im = Image.open(path).convert('RGBA'); hd = Image.new('RGBA', (96, 96))
        w, h = (box[2]-box[0])*2, (box[3]-box[1])*2
        for y in range(h):
            for x in range(w):
                hd.putpixel((box[0]*2+x, box[1]*2+y), im.getpixel((crop[0]+x*(crop[2]-crop[0])//w, crop[1]+y*(crop[3]-crop[1])//h)))
        hd_sprites[key] = hd
    diagnostic = json.loads((terrain/'backgrounds/startup-probe/latest.json').read_text())
    backdrops = {}
    for i, s in enumerate(sprites):
        rows = [r for r in diagnostic['sprite_backdrop_records'] if r['begin_step'] == s['BeginStep']]
        assert len(rows) == 1 and s['background_verified']
        row = rows[0]
        assert s['EndStep'] == row['end_step'] and s['background_rgba_sha256'] == row['before']['rgba_sha256']
        assert s['source_rgba_sha256'] == row['source']['rgba_sha256'] and s['after_rgba_sha256'] == row['after']['rgba_sha256']
        backdrops[i] = Image.open(terrain/'backgrounds'/row['before']['path']).convert('RGBA')
        assert hashlib.sha256(backdrops[i].tobytes()).hexdigest() == s['background_rgba_sha256']
    total = 640*400
    base = [-1]*total; background = [-1]*total; order = [0]*total; owner = [0]*total; scene = [None]*total
    for i, d in enumerate(floors):
        rect = bounds(d); im = native_image('floor', d['index'])
        for y in range(rect[1], rect[3]):
            for x in range(rect[0], rect[2]):
                n = y*640+x; base[n] = i; order[n] = owner[n] = d['draw_order']; scene[n] = im.getpixel((x-rect[0], y-rect[1]))
    for i, s in enumerate(sprites):
        rect = bounds(s)
        for y in range(rect[1], rect[3]):
            for x in range(rect[0], rect[2]):
                n = y*640+x
                if scene[n] is None:
                    scene[n] = backdrops[i].getpixel((x-rect[0], y-rect[1])); background[n] = i; order[n] = s['draw_order']-1
    for s in sprites:
        rect = bounds(s); im = native_image(s['group'], s['index'])
        for y in range(rect[1], rect[3]):
            for x in range(rect[0], rect[2]):
                n = y*640+x
                if scene[n] is None:
                    continue
                c = im.getpixel((x-rect[0], y-rect[1]))
                if c[3] != 255:
                    continue
                if s['draw_order'] <= order[n]:
                    if base[n] < 0:
                        owner[n] = s['draw_order'] if c == scene[n] else 0
                else:
                    scene[n], owner[n] = c, s['draw_order']
    affected = bytearray(total)
    for n, i in enumerate(base):
        affected[n] = i in active_floors
    for i in active_sprites.values():
        rect = bounds(sprites[i])
        for y in range(rect[1], rect[3]):
            affected[y*640+rect[0]:y*640+rect[2]] = b'\1'*(rect[2]-rect[0])
    baseline = json.loads((project/'workplace/desktop-prototype/headless/startup-probe/latest.json').read_text())
    cur, mouse = baseline['mouse_cursor'], baseline['mouse_input_events'][-1]
    cx, cy = mouse['x']-cur['HotX'], mouse['y']-cur['HotY']
    tree_rects = [bounds(sprites[i]) for i in active_sprites.values() if sprites[i]['group'] == 'objects']
    changed = checked = ui = protected = overlap_units = 0
    removed = {1: 0, 3: 0}
    for y in range(400):
        for x in range(640):
            n = y*640+x; cursor = cx <= x < cx+16 and cy <= y < cy+16
            hd_floor = base[n] in active_floors
            owner_hd = owner[n] in active_sprites
            foreground = owner[n] != 0 and (base[n] < 0 or owner[n] != order[n])
            if affected[n] and foreground and not owner_hd:
                protected += 1
            layers = [(i, s, bounds(s), native_image(s['group'], s['index'])) for i, s in enumerate(sprites) if s['draw_order'] > order[n] and inside(bounds(s), x, y)] if affected[n] else []
            if owner_hd:
                d = sprites[active_sprites[owner[n]]]
                overlap_units += d['group'] == 'units' and any(inside(rect, x, y) for rect in tree_rects)
            for dy in range(2):
                for dx in range(2):
                    pixel = ((y*2+dy)*1280+x*2+dx)*4
                    a, b = enhanced[pixel:pixel+4], original[pixel:pixel+4]
                    if a != b:
                        assert affected[n], ('outside qualified layers', x, y)
                        changed += 1
                    if not affected[n] or cursor:
                        continue
                    assert bytes(scene[n]) == b, ('original source scene differs', x, y)
                    if base[n] >= 0:
                        d = floors[base[n]]; rect = bounds(d)
                        c = hd_floors[d['index']].getpixel(((x-rect[0])*2+dx, (y-rect[1])*2+dy)) if hd_floor else native_image('floor', d['index']).getpixel((x-rect[0], y-rect[1]))
                    else:
                        s = sprites[background[n]]; rect = bounds(s)
                        c = backdrops[background[n]].getpixel((x-rect[0], y-rect[1]))
                    for i, s, rect, im in layers:
                        source_pixel = im.getpixel((x-rect[0], y-rect[1])); key = s['group'], s['index']
                        if s['draw_order'] in active_sprites:
                            if s['draw_order'] == owner[n] or (base[n] < 0 and source_pixel[3] == 0):
                                continue
                            c = blend(c, hd_sprites[key].getpixel(((x-rect[0])*2+dx, (y-rect[1])*2+dy)))
                        elif source_pixel[3] == 255:
                            c = source_pixel
                    if owner_hd:
                        s = sprites[active_sprites[owner[n]]]; rect = bounds(s)
                        alpha_pixel = hd_sprites[(s['group'], s['index'])].getpixel(((x-rect[0])*2+dx, (y-rect[1])*2+dy))
                        c = blend(c, alpha_pixel)
                        if s['group'] == 'units' and alpha_pixel[3] == 0 and a != b:
                            removed[s['index']] += 1
                    elif foreground:
                        c = scene[n]
                    if base[n] < 0 and owner[n] == 0:
                        c = scene[n]; ui += 1
                    assert a == bytes(c), ('HD reference composite differs', x, y, dx, dy, tuple(a), c)
                    checked += 1
    assert protected == report['protected_original_foreground_pixels'] == 0
    assert ui == report['preserved_UI_output_pixels']
    assert overlap_units == (0 if mode == 'scene-missing' else 82)
    assert (changed > 0) == (mode != 'scene-missing')
    if mode != 'scene-missing':
        assert checked > 20000 and all(v > 0 for v in removed.values())
    assert report['floor_native_pixels'] == (102720 if mode == 'scene' else 0)
    return {'changed_pixels': changed, 'independent_output_pixels_verified': checked,
            'HD_floor_draws': len(active_floors), 'HD_tree_draws': report['tree_hd_count'], 'HD_unit_draws': report['unit_hd_count'],
            'floor_native_pixels': report['floor_native_pixels'], 'original_visible_unit_pixels_overlapping_HD_tree': overlap_units,
            'old_unit_outline_output_pixels_removed': removed, 'original_UI_output_pixels_preserved': ui,
            'protected_original_foreground_pixels': protected, 'outside_qualified_layers_unchanged': True,
            'native_source_scene_verified': True, 'HD_reference_composite_verified': True,
            'cursor_formula_excluded_logical_rect': [cx, cy, cx+16, cy+16],
            'pixel_verifier_sha256': sha(Path(__file__))}
