"""以正常繪製順序核對透明前景，準備 39 個原版來源遮罩；不生成 HD。"""
import hashlib
import io
import json
import os
from pathlib import Path

from PIL import Image, __version__ as pillow_version


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_same(path, data):
    assert not path.exists() or path.stat().st_uid == 1000 and path.read_bytes() == data, path
    path.write_bytes(data)


def overlaps(a, b):
    return a[0] < b[2] and b[0] < a[2] and a[1] < b[3] and b[1] < a[3]


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000 and pillow_version == '9.4.0'
    project, output = Path('/project'), Path('/output')
    catalog = project/'workplace/hd-prototype/catalog'
    trace = output/'trace'
    receipt_path = trace/'startup-probe/latest.json'
    receipt = json.loads(receipt_path.read_text())
    baseline_path = project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    baseline = json.loads(baseline_path.read_text())
    fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip', 'trace_tail',
              'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame', 'cd_pcm', 'mounted_disc',
              'mouse_input_events', 'input_boundary_frames', 'real_mode_history', 'real_mode_last']
    assert all(receipt[k] == baseline[k] for k in fields), [k for k in fields if receipt[k] != baseline[k]]
    assert receipt['successful_steps'] == 750_000_000 and receipt['fd2_startup_calls'] == 0
    assert len(receipt['mouse_input_events']) == 390
    ram_sha = sha(trace/'startup-memory.bin')
    assert ram_sha == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name, digest in receipt['integration_input_sha256'].items():
        assert sha(trace/'source-snapshot'/name) == digest, name
    extraction_path = catalog/'terrain/terrain-extraction.json'
    extraction = json.loads(extraction_path.read_text())
    for name, digest in extraction['ida_evidence'].items():
        assert sha(catalog/name) == digest
    plane_sha = [p['sha256'] for p in extraction['planes']]
    palette_path = catalog/'terrain/trace/native-floor-palette-verification.json'
    palette_proof = json.loads(palette_path.read_text())
    assert sha(project/palette_proof['palette_capture']) == palette_proof['palette_capture_sha256']
    palette = palette_proof['palette']
    idx_path = catalog/'decoded/MAP_CHIP.OZM.idx'
    idx = idx_path.read_bytes()
    assert len(idx) == 640*400 and sha(idx_path) == extraction['indices_sha256'] == palette_proof['source_indices_sha256']
    sources = output/'sources'
    sources.mkdir(exist_ok=True)
    assert output.stat().st_uid == sources.stat().st_uid == 1000
    images, records = {}, []
    originals = {(r['group'], r['source_grid_index']): r for r in extraction['records']}
    for group, count, base_y in [('objects', 26, 256), ('units', 13, 352)]:
        for index in range(count):
            x, y = index%13*48, base_y+index//13*48
            raw = b''.join(idx[row*640+x:row*640+x+48] for row in range(y, y+48))
            raw_sha = hashlib.sha256(raw).hexdigest()
            assert raw_sha == originals[group[:-1], index]['indices_sha256']
            image = Image.frombytes('RGBA', (48, 48), bytes(c for i in raw for c in (*palette[i], 255 if i else 0)))
            buf = io.BytesIO()
            image.save(buf, format='PNG')
            name = f'{group}-{index:03d}-original.png'
            write_same(sources/name, buf.getvalue())
            images[group, index] = image
            records.append({'group': group, 'index': index, 'source': 'sources/'+name,
                            'source_png_sha256': sha(sources/name), 'indices_sha256': raw_sha,
                            'opaque_pixels': sum(i != 0 for i in raw)})
    diagnostic = receipt['terrain_draw_trace']
    assert diagnostic['dropped_events'] == 0 and diagnostic['pending_return'] is None
    draws, pending = [], None
    for e in diagnostic['events']:
        assert 'observation_error' not in e
        assert [p['sha256'] for p in e['planes']] == plane_sha
        if e['kind'].endswith('_enter'):
            assert pending is None
            assert (e['raw_35988'], e['raw_3598c'], e['raw_35990']) == (0, 0x5000, 0x6e00)
            assert not (e['raw_35984'] and e['raw_4217c'])
            pending = e
            continue
        assert pending is not None and e['kind'] == pending['kind'].replace('_enter', '_return')
        assert e['begin_step'] == pending['step'] and e['linear_eip'] == pending['return_linear']
        assert e['raw_source_offset'] == pending['raw_source_offset'] and e['raw_destination_offset'] == pending['raw_destination_offset']
        sy, sx = divmod(e['raw_source_offset'], 80)
        y, xb = divmod(e['raw_destination_offset'], 80)
        assert sx%6 == 0 and sx < 78
        if e['kind'] == 'floor_return':
            assert sy < 256 and sy%32 == 0
            group, index, h = 'floor', sy//32*13+sx//6, 32
        else:
            if 256 <= sy < 352:
                group, index = 'objects', (sy-256)//48*13+sx//6
                assert (sy-256)%48 == 0 and e['linear_eip'] == 0x1fe69
            else:
                assert sy == 352 and e['linear_eip'] == 0x1fea2
                group, index = 'units', sx//6
            h = 48
        rect = [xb*8, y, xb*8+48, y+h]
        assert rect[0] >= 0 and rect[1] >= 0 and rect[2] <= 640 and rect[3] <= 400
        draws.append({'group': group, 'index': index, 'rect': rect, 'order': len(draws)+1, 'step': e['step']})
        pending = None
    assert pending is None
    floors, sprites = [], []
    for d in draws:
        if d['group'] == 'floor':
            floors = [old for old in floors if not overlaps(old['rect'], d['rect'])]
            floors.append(d)
        else:
            sprites.append(d)
    native_path = project/'workplace/startup-probe'/('frame-'+receipt['frame']['png_sha256']+'.png')
    assert sha(native_path) == palette_proof['frame_sha256']
    native = Image.open(native_path).convert('RGBA')
    matched, rejected = [], []
    for d in floors:
        floor_path = catalog/'terrain/prepared/sources'/f'floor-{d["index"]:03d}-original.png'
        source = Image.open(floor_path).convert('RGBA')
        expected = source.copy()
        foreground = bytearray(48*32)
        for s in sprites:
            if s['order'] <= d['order'] or not overlaps(d['rect'], s['rect']):
                continue
            sprite = images[s['group'], s['index']]
            for y in range(max(d['rect'][1], s['rect'][1]), min(d['rect'][3], s['rect'][3])):
                for x in range(max(d['rect'][0], s['rect'][0]), min(d['rect'][2], s['rect'][2])):
                    c = sprite.getpixel((x-s['rect'][0], y-s['rect'][1]))
                    if c[3]:
                        lx, ly = x-d['rect'][0], y-d['rect'][1]
                        expected.putpixel((lx, ly), c)
                        foreground[ly*48+lx] = 1
        if expected.tobytes() == native.crop(d['rect']).tobytes():
            equal_colour = sum(1 for i, opaque in enumerate(foreground) if opaque and expected.getpixel((i%48, i//48)) == source.getpixel((i%48, i//48)))
            matched.append({**d, 'foreground_pixels': sum(foreground), 'foreground_equal_floor_pixels': equal_colour})
        else:
            rejected.append(d)
    profile = {'classification': 'local original sprite sources and index-zero masks; no HD sprites enabled',
               'pillow_version': pillow_version, 'tool_sha256': sha(Path(__file__)), 'source_indices_sha256': sha(idx_path),
               'extraction_sha256': sha(extraction_path), 'palette_verification_sha256': sha(palette_path), 'records': records}
    write_same(output/'source-profiles.json', (json.dumps(profile, ensure_ascii=False, indent=2)+'\n').encode())
    lines = ['// Code generated by tools/hd-terrain-object-prepare.py; DO NOT EDIT.', 'package main',
             'type hdTerrainSpriteProfile struct { Group string; Index int; SourceSHA string }',
             'var hdTerrainSpriteProfiles = []hdTerrainSpriteProfile{']
    for r in records:
        lines.append('{'+json.dumps(r['group'])+', '+str(r['index'])+', '+json.dumps(r['source_png_sha256'])+'},')
    lines += ['}', '']
    (project/'tools/hd-terrain-sprite-profiles.go').write_text('\n'.join(lines))
    result = {'verified': True, 'classification': 'normal original draw ordering and exact alpha-composition; no HD sprites',
              'receipt_sha256': sha(receipt_path), 'ram_sha256': ram_sha, 'baseline_sha256': sha(baseline_path),
              'compared_fields': fields, 'source_profiles_sha256': sha(output/'source-profiles.json'),
              'floor_returns': sum(d['group'] == 'floor' for d in draws), 'sprite_returns': len(sprites),
              'draws': draws, 'current_floors': floors, 'sprites': sprites,
              'matched_rectangles': matched, 'rejected_rectangles': rejected,
              'tool_sha256': sha(Path(__file__)), 'integration_input_sha256': receipt['integration_input_sha256']}
    write_same(trace/'verification.json', (json.dumps(result, ensure_ascii=False, indent=2)+'\n').encode())
    print(json.dumps({k: result[k] for k in ['verified', 'floor_returns', 'sprite_returns']}))
    print(json.dumps({'matched_floor_rectangles': len(matched), 'occluded_floor_rectangles': sum(d['foreground_pixels'] > 0 for d in matched),
                      'foreground_pixels': sum(d['foreground_pixels'] for d in matched), 'equal_colour_pixels': sum(d['foreground_equal_floor_pixels'] for d in matched)}))


if __name__ == '__main__':
    main()
