"""核對普通歷史簡介中的位移來源與跨列複製；不接入 HD。"""
import hashlib
import json
import os
from pathlib import Path
import platform


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def decode(packed):
    assert len(packed) == 128000
    out = bytearray(256000)
    for bit in range(4):
        for offset, value in enumerate(packed[bit * 32000:(bit + 1) * 32000]):
            for x in range(8):
                out[offset * 8 + x] |= ((value >> (7 - x)) & 1) << bit
    return bytes(out)


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, output = Path('/project'), Path('/output')
    history = project / 'workplace/hd-prototype/catalog/remaining/history'
    root = history / 'probe'
    receipt_path = root / 'startup-probe/latest.json'
    receipt = json.loads(receipt_path.read_text())
    previous_path = history / 'source-verification.json'
    previous = json.loads(previous_path.read_text())
    assert previous['verified'] and previous['receipt_sha256'] == sha(receipt_path)
    for name, digest in receipt['integration_input_sha256'].items():
        assert sha(root / 'source-snapshot' / name) == digest
    inventory = json.loads((history.parents[1] / 'decoded/inventory.json').read_text())
    profiles = {r['name']: r for r in inventory['records']}
    records = []
    for sample in receipt['opening_draw_trace']['sampled_returns']:
        entry = sample['entry_linear']
        if entry not in [0x11ea4, 0x11c83]:
            continue
        name = 'OP2.OZM' if entry == 0x11ea4 else 'OP8.OZM'
        sx, cols, rows, dx, dy = (240, 30, 150, 200, 50) if entry == 0x11ea4 else (440, 26, 200, 216, 30)
        count = sample['entry_count'] - 1
        assert sample['last_load_name_hint'] == name
        assert sample['before']['candidate_bases'] == sample['after']['candidate_bases']
        loads = [e for e in receipt['hd_asset_trace']['events'] if e['kind'] == 'load_return' and e['step'] <= sample['begin_step']]
        load = loads[-1]
        assert load['name'] == name and load['raw_columns'] == 80 and load['raw_rows'] == 200
        assert sample['before']['candidate_bases'] == [p['base_linear'] for p in load['planes']]
        assert [p['sha256'] for p in load['planes']] == profiles[name]['plane_sha256']
        path = root / 'opening-samples' / sample['before']['candidate_planes']['path']
        assert sha(path) == sample['before']['candidate_planes']['sha256'] == sample['after']['candidate_planes']['sha256']
        assert path.read_bytes() == (root / 'opening-samples' / sample['after']['candidate_planes']['path']).read_bytes()
        current = decode(path.read_bytes())
        original_path = history.parents[1] / 'decoded' / (name + '.idx')
        original = original_path.read_bytes()
        assert sha(original_path) == profiles[name]['indices_sha256'] and len(original) == 128000
        # sub_11AEB leaves the first 16 bits fixed; the remaining known prefix shifts right.
        expected = bytes(original[i if i < 16 else max(15, i - count)] for i in range(len(original)))
        assert current[:len(original)] == expected, (name, count, 'known source transform')
        meta = sample['after']['native_indices']
        native_path = root / 'opening-samples' / meta['path']
        assert sha(native_path) == meta['sha256']
        native = native_path.read_bytes()
        copied = b''.join(current[y * 640 + sx:y * 640 + sx + cols * 8] for y in range(rows))
        destination = b''.join(native[y * 640 + dx:y * 640 + dx + cols * 8] for y in range(dy, dy + rows))
        assert copied == destination, (name, count, 'linear source/destination')
        unknown, known = [], 0
        for y in range(rows):
            for x in range(cols * 8):
                offset = y * 640 + sx + x
                source = offset if offset < 16 else max(15, offset - count)
                if source >= len(original):
                    unknown.append([dx + x, dy + y])
                else:
                    assert native[(dy + y) * 640 + dx + x] == original[source]
                    known += 1
        records.append({'source': name, 'sample_id': sample['sample_id'], 'entry_linear': entry,
                        'begin_step': sample['begin_step'], 'end_step': sample['end_step'], 'shift_count': count,
                        'source_x': sx, 'columns': cols, 'rows': rows, 'dest_rect': [dx, dy, dx + cols * 8, dy + rows],
                        'known_source_prefix_equal': True, 'linear_copy_equal': True, 'known_art_pixels': known,
                        'outside_formal_source_pixels': unknown, 'active_palette': [sample['after']['palette'][i] for i in sample['after']['DAC']]})
    assert len(records) == 13
    ida_path = history / 'ida/range-11aeb-12900.json'
    result = {'verified': True, 'classification': 'normal original scrolling source and linear copy evidence; not HD completion',
              'tool_sha256': sha(Path(__file__)), 'python': platform.python_version(), 'receipt_sha256': sha(receipt_path),
              'source_proof_sha256': sha(previous_path), 'complete_RAM_sha256': previous['complete_RAM_sha256'],
              'ida_evidence_sha256': sha(ida_path), 'address_space': 'IDA LE loader linear EA; source pixel indices separately row-major',
              'records': records, 'limitations': ['OP8 initial right edge reads beyond its formal 200-row source; those pixels are not qualified art',
                                                 'palette eligibility and host HD transformation remain DRAFT']}
    target = output / 'source-verification.json'
    assert not target.exists()
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'verified': True, 'samples': len(records), 'sha256': sha(target)}))


if __name__ == '__main__':
    main()
