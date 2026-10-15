"""核對普通歷史簡介的未位移局部來源；不修改原版或接入 HD。"""
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def decode(packed):
    assert len(packed) == 128000
    result = bytearray(256000)
    for plane in range(4):
        for offset, value in enumerate(packed[plane * 32000:(plane + 1) * 32000]):
            for bit in range(8):
                result[offset * 8 + bit] |= ((value >> (7 - bit)) & 1) << plane
    return bytes(result)


def crop(data, width, rect):
    x0, y0, x1, y1 = rect
    assert 0 <= x0 < x1 <= width and 0 <= y0 < y1 <= len(data) // width
    return b''.join(data[y * width + x0:y * width + x1] for y in range(y0, y1))


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project = Path('/project')
    history = project / 'workplace/hd-prototype/catalog/remaining/history'
    root = history / 'probe'
    receipt_path = root / 'startup-probe/latest.json'
    receipt = json.loads(receipt_path.read_text())
    previous = json.loads((history / 'source-verification.json').read_text())
    assert previous['verified'] and previous['receipt_sha256'] == sha(receipt_path)
    for name, digest in receipt['integration_input_sha256'].items():
        assert sha(root / 'source-snapshot' / name) == digest, name
    decoded = {r['name']: r for r in json.loads((history.parents[1] / 'decoded/inventory.json').read_text())['records']}
    profiles = json.loads((history.parent / 'profiles.json').read_text())['records']
    op9 = next(r for r in profiles if r['name'] == 'OP9.OZM')
    immutable = bytearray(b'\1' * 256000)
    for rect in op9['reserved_buffer_rects']:
        for y in range(rect[1], rect[3]):
            immutable[y * 640 + rect[0]:y * 640 + rect[2]] = b'\0' * (rect[2] - rect[0])
    records = []
    parameters = {0x11e37: (16, 119, 0xa0fbf), 0x11dcb: (40, 150, 0xa0fb4),
                  0x11ba6: (26, 200, 0xa097b), 0x11c16: (9, 27, 0xa1ebb)}
    for r in receipt['opening_draw_trace']['sampled_returns']:
        if r['entry_linear'] not in parameters:
            continue
        before, after = r['before'], r['after']
        source_path = root / 'opening-samples' / before['candidate_planes']['path']
        assert sha(source_path) == before['candidate_planes']['sha256'] == after['candidate_planes']['sha256']
        assert source_path.read_bytes() == (root / 'opening-samples' / after['candidate_planes']['path']).read_bytes()
        actual = decode(source_path.read_bytes())
        loads = [e for e in receipt['hd_asset_trace']['events'] if e['kind'] == 'load_return' and e['step'] <= r['begin_step']]
        load = loads[-1]
        name = load['name']
        assert name == r['last_load_name_hint']
        assert before['candidate_bases'] == after['candidate_bases'] == [p['base_linear'] for p in load['planes']]
        source = decoded[name]
        width, height = source['width'], source['height']
        original_path = history.parents[1] / 'decoded' / (name + '.idx')
        original = original_path.read_bytes()
        assert sha(original_path) == source['indices_sha256']
        observed = crop(actual, 640, [0, 0, width, height])
        changes = [i for i, (a, b) in enumerate(zip(observed, original)) if a != b]
        if name == 'OP9.OZM':
            assert all(not immutable[i] for i in changes), 'OP9 artwork changed outside reserved buffers'
        else:
            assert not changes, (name, 'source differs from static decode')
        cols, rows, dest = parameters[r['entry_linear']]
        offset = 0 if r['entry_linear'] == 0x11dcb else r['raw_stack_words'][0]
        sx, sy = offset % 80 * 8, offset // 80
        dx, dy = (dest - 0xa0000) % 80 * 8, (dest - 0xa0000) // 80
        src, dst = [sx, sy, sx + cols * 8, sy + rows], [dx, dy, dx + cols * 8, dy + rows]
        expected = crop(original, width, src)
        assert expected == crop(actual, 640, src)
        native = {}
        for phase in ['before', 'after']:
            meta = r[phase]['native_indices']
            path = root / 'opening-samples' / meta['path']
            assert sha(path) == meta['sha256']
            native[phase] = path.read_bytes()
        assert expected == crop(native['after'], 640, dst), (name, src, dst)
        unchanged = bytearray(native['after'])
        for y in range(dy, dy + rows):
            unchanged[y * 640 + dx:y * 640 + dx + cols * 8] = native['before'][y * 640 + dx:y * 640 + dx + cols * 8]
        assert bytes(unchanged) == native['before'], (name, 'writes outside declared rectangle')
        records.append({'source': name, 'sample_id': r['sample_id'], 'entry_linear': r['entry_linear'],
                        'begin_step': r['begin_step'], 'end_step': r['end_step'], 'source_rect': src, 'dest_rect': dst,
                        'source_match': 'exact except reviewed reserved buffers' if changes else 'exact whole resource',
                        'reserved_buffer_changed_pixels': len(changes), 'indexed_sha256': source['indices_sha256'],
                        'copied_pixels': len(expected), 'source_and_destination_equal': True, 'outside_destination_unchanged': True,
                        'active_palette': [after['palette'][i] for i in after['DAC']]})
    assert len(records) == 14, len(records)
    result = {'verified': True, 'classification': 'normal original partial-copy evidence; no HD completion claim',
              'tool_sha256': sha(Path(__file__)), 'receipt_sha256': sha(receipt_path),
              'complete_RAM_sha256': previous['complete_RAM_sha256'], 'source_snapshot_count': len(receipt['integration_input_sha256']),
              'records': records, 'limitations': ['scrolling sources require separate transform qualification',
                                                  'OP9 mutable reserved buffers cannot use whole-resource fingerprint unconditionally']}
    target = history / 'partial/source-verification.json'
    assert not target.exists()
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'verified': True, 'samples': len(records), 'sources': sorted({r['source'] for r in records}), 'sha256': sha(target)}))


if __name__ == '__main__':
    main()
