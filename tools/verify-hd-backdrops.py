"""獨立核對原版圖塊的繪製前背景、來源、返回畫面與完整玩家路徑。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, output = Path('/project'), Path('/output')
    receipt_path = output/'startup-probe/latest.json'
    receipt = json.loads(receipt_path.read_text())
    baseline_path = project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    baseline = json.loads(baseline_path.read_text())
    fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip',
              'trace_tail', 'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame',
              'cd_pcm', 'mounted_disc', 'mouse_input_events', 'input_boundary_frames',
              'real_mode_history', 'real_mode_last']
    assert all(receipt[k] == baseline[k] for k in fields)
    assert receipt['successful_steps'] == 750_000_000 and len(receipt['mouse_input_events']) == 390
    assert receipt['fd2_startup_calls'] == 0 and receipt['sprite_backdrop_dropped'] == 0
    assert sha(output/'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name, digest in receipt['integration_input_sha256'].items():
        assert sha(output/'source-snapshot'/name) == digest
    records = []
    previous = -1
    for row in receipt['sprite_backdrop_records']:
        assert row['primitive_verified'] and not row['observation_error']
        assert previous < row['begin_step'] < row['end_step'] < 750_000_000
        previous = row['end_step']
        pictures = {}
        for key in ['before', 'source', 'after']:
            info = row[key]
            path = output/info['path']
            assert sha(path) == info['png_sha256']
            im = Image.open(path).convert('RGBA')
            assert im.size == (48, 48)
            assert hashlib.sha256(im.tobytes()).hexdigest() == info['rgba_sha256']
            pictures[key] = im
        draw = row['draw']
        native_path = project/'workplace/hd-prototype/catalog/terrain/objects/sources'/f"{draw['group']}-{draw['index']:03d}-original.png"
        native = Image.open(native_path).convert('RGBA')
        assert pictures['source'].tobytes() == native.tobytes()
        before, source, after = (pictures[k].tobytes() for k in ['before', 'source', 'after'])
        expected = bytearray(before)
        for i in range(0, len(source), 4):
            assert source[i+3] in (0, 255) and before[i+3] == after[i+3] == 255
            if source[i+3] == 255:
                expected[i:i+4] = source[i:i+4]
        assert bytes(expected) == after
        records.append({'draw': draw, 'begin_step': row['begin_step'], 'end_step': row['end_step'],
                        'source_original_png_sha256': sha(native_path),
                        'before_png_sha256': row['before']['png_sha256'],
                        'after_png_sha256': row['after']['png_sha256'], 'exact_primitive_verified': True})
    assert len(records) == 11
    log = output/'startup-probe/platform-test.log'
    assert sum(l.startswith('ok  ') for l in log.read_text().splitlines()) == 4 and 'FAIL' not in log.read_text()
    result = {'classification': 'read-only original sprite primitive and pre-draw background; not HD GUI acceptance',
              'whole_game_HD_complete': False, 'receipt_sha256': sha(receipt_path),
              'baseline_sha256': sha(baseline_path), 'compared_fields': fields,
              'records': records, 'platform_test_log_sha256': sha(log), 'tool_sha256': sha(Path(__file__))}
    path = output/'background-verification.json'
    path.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'verified': True, 'original_primitive_instances': len(records), 'summary_sha256': sha(path)}))


if __name__ == '__main__':
    main()
