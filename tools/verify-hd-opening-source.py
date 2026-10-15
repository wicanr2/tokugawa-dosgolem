"""核對原版開場緩衝區與完整複製，不以載入名稱代替來源證據。"""
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def indexed(packed):
    assert len(packed) == 128000
    result = bytearray(640 * 400)
    for number in range(4):
        plane = packed[number * 32000:(number + 1) * 32000]
        for offset, value in enumerate(plane):
            for bit in range(8):
                result[offset * 8 + bit] |= ((value >> (7 - bit)) & 1) << number
    return bytes(result)


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project = Path('/project')
    history = project / 'workplace/hd-prototype/catalog/remaining/history'
    root = history / 'probe'
    receipt_path = root / 'startup-probe/latest.json'
    receipt = json.loads(receipt_path.read_text())
    baseline_root = history.parent / 'source-boundary/menu-history/rcr'
    baseline_path = baseline_root / 'receipt.json'
    baseline = json.loads(baseline_path.read_text())
    fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip', 'trace_tail',
              'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame', 'cd_pcm', 'mounted_disc',
              'mouse_input_events', 'input_boundary_frames', 'real_mode_history', 'real_mode_last']
    assert receipt['successful_steps'] == 600000000 and receipt['stop'] == 'instruction_budget'
    for field in fields:
        assert receipt[field] == baseline[field], field
    assert sha(root / 'startup-memory.bin') == sha(baseline_root / 'startup-memory.bin')
    for name, digest in receipt['integration_input_sha256'].items():
        assert sha(project / name) == digest == sha(root / 'source-snapshot' / name), name
    samples = root / 'opening-samples'
    records = receipt['opening_draw_trace']['sampled_returns']
    assert len(records) == 65 and receipt['opening_draw_trace']['dropped_samples'] == 0
    full = []
    shifts = []
    for record in records:
        assert record['end_step'] > record['begin_step'] and 'observation_error' not in record
        packed = {}
        for phase in ['before', 'after']:
            capture = record[phase]
            assert 'source_error' not in capture and 'native_error' not in capture
            source_path = samples / capture['candidate_planes']['path']
            assert sha(source_path) == capture['candidate_planes']['sha256']
            packed[phase] = source_path.read_bytes()
            native_path = samples / capture['native_indices']['path']
            assert sha(native_path) == capture['native_indices']['sha256']
        if record['entry_linear'] == 0x11f14:
            assert packed['before'] == packed['after']
            native = (samples / record['after']['native_indices']['path']).read_bytes()
            source = indexed(packed['before'])
            assert source == native
            loads = [event for event in receipt['hd_asset_trace']['events']
                     if event['kind'] == 'load_return' and event['step'] <= record['begin_step']]
            load = loads[-1]
            assert load['name'] == record['last_load_name_hint']
            assert load['raw_columns'] == 80 and load['raw_rows'] == 400
            bases = [plane['base_linear'] for plane in load['planes']]
            assert record['before']['candidate_bases'] == record['after']['candidate_bases'] == bases
            plane_sha = [hashlib.sha256(packed['before'][i * 32000:(i + 1) * 32000]).hexdigest()
                         for i in range(4)]
            assert plane_sha == [plane['sha256'] for plane in load['planes']]
            full.append({'source': load['name'], 'sample_id': record['sample_id'],
                         'entry_linear': record['entry_linear'], 'begin_step': record['begin_step'],
                         'end_step': record['end_step'], 'return_linear': record['return_linear'],
                         'candidate_bases': bases, 'loader_alias_verified': True,
                         'planes_before': record['before']['candidate_planes']['path'],
                         'native_after': record['after']['native_indices']['path'],
                         'plane_sha256': plane_sha, 'indexed_sha256': hashlib.sha256(source).hexdigest(),
                         'all_256000_pixels_equal': True})
        elif record['entry_linear'] == 0x11aeb:
            expected = bytearray(packed['before'])
            for number in range(4):
                start = number * 32000
                original = packed['before'][start:start + 32000]
                for offset in range(2, 16002, 2):
                    value = int.from_bytes(original[offset:offset + 2], 'big')
                    result = value >> 1 | (original[offset - 1] & 1) << 15
                    expected[start + offset:start + offset + 2] = result.to_bytes(2, 'big')
            assert bytes(expected) == packed['after'], ('shift bytes', record['sample_id'])
            assert record['before']['native_indices']['sha256'] == record['after']['native_indices']['sha256']
            shifts.append(record['sample_id'])
    assert [record['source'] for record in full] == ['OP4.OZM', 'OP5.OZM', 'VISUAL.OZM', 'OP-FULL.OZM']
    old_runtime = history.parent / 'source-boundary/OP-FULL-runtime/OP-FULL-runtime.idx'
    assert full[-1]['indexed_sha256'] == sha(old_runtime)
    log = (root / 'startup-probe/platform-test.log').read_text()
    assert 'FAIL' not in log and len([line for line in log.splitlines() if line.startswith('ok')]) == 4
    result = {'verified': True, 'classification': 'original read-only source, alias and complete-copy evidence; no HD rendering claim',
              'tool_sha256': sha(Path(__file__)), 'receipt_sha256': sha(receipt_path),
              'baseline_receipt_sha256': sha(baseline_path), 'complete_RAM_sha256': sha(root / 'startup-memory.bin'),
              'compared_fields': fields, 'source_snapshot_count': len(receipt['integration_input_sha256']),
              'sampled_returns': len(records), 'full_copy_records': full,
              'bit_shift_samples_verified': shifts,
              'runtime_tail_scope': 'two matching ordinary history executions; original payload remains unchanged',
              'limitations': ['other partial-copy wrappers remain outside this complete-copy contract',
                              'source evidence does not establish normal HD rendering or whole-game completion']}
    target = history / 'source-verification.json'
    assert not target.exists()
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'verified': True, 'full_copy_sources': [r['source'] for r in full],
                      'bit_shift_samples': len(shifts), 'summary_sha256': sha(target)}))


if __name__ == '__main__':
    main()
