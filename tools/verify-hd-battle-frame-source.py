"""獨立核對 MAPGAMEN 全畫面自然返回與實際 VGA 色彩。"""
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
    root = output/'source-probe'
    receipt_path = root/'startup-probe/latest.json'
    receipt = json.loads(receipt_path.read_text())
    baseline_path = project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    baseline = json.loads(baseline_path.read_text())
    fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip',
              'trace_tail', 'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame',
              'cd_pcm', 'mounted_disc', 'mouse_input_events', 'input_boundary_frames',
              'real_mode_history', 'real_mode_last']
    assert all(receipt[k] == baseline[k] for k in fields)
    assert receipt['successful_steps'] == 750_000_000 and len(receipt['mouse_input_events']) == 390
    assert receipt['fd2_startup_calls'] == 0
    assert sha(root/'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
    for name, digest in receipt['integration_input_sha256'].items():
        assert sha(root/'source-snapshot'/name) == digest
    state = receipt['battle_frame_observation']
    assert receipt['battle_frame_diagnostic_enabled'] and state['draw_completed']
    assert state['pending_return'] is None and state['dropped_events'] == 0
    enter, returned = state['events']
    assert enter['kind'] == 'battle_frame_enter' and returned['kind'] == 'battle_frame_return'
    assert enter['linear_eip'] == 0x1c084 and enter['return_linear'] == returned['linear_eip']
    assert enter['stack_linear']+4 == returned['stack_linear'] and enter['step'] == returned['begin_step']
    assert returned['eligible'] and enter['step'] < returned['step'] == 657204466
    info = receipt['battle_frame_capture']
    index_path, png_path = root/info['indices_path'], root/info['path']
    expected_sha = '99a8497101f145c62d01934204f1c3ed74282fa6bd993fdbf4f8969eb0f7277c'
    assert sha(index_path) == info['indices_sha256'] == returned['native_indexed_sha256'] == expected_sha
    original = project/'workplace/hd-prototype/catalog/hd-assets'/f'MAPGAMEN.OZM-{expected_sha}.idx'
    assert index_path.read_bytes() == original.read_bytes()
    indices = index_path.read_bytes()
    assert len(indices) == 640*400 and max(indices) <= 15
    assert sha(png_path) == info['png_sha256']
    im = Image.open(png_path).convert('RGBA')
    assert im.size == (640, 400)
    palette = [tuple(returned['palette'][returned['dac'][i]])+(255,) for i in range(16)]
    expected = b''.join(bytes(palette[i]) for i in indices)
    assert im.tobytes() == expected
    log = root/'startup-probe/platform-test.log'
    assert sum(l.startswith('ok  ') for l in log.read_text().splitlines()) == 4 and 'FAIL' not in log.read_text()
    result = {'classification': 'confirmed original MAPGAMEN full-copy source, native indices and displayed palette; no HD GUI claim',
              'whole_game_HD_complete': False, 'receipt_sha256': sha(receipt_path), 'baseline_sha256': sha(baseline_path),
              'compared_fields': fields, 'entry_step': enter['step'], 'return_step': returned['step'],
              'entry_linear_eip': enter['linear_eip'], 'return_linear_eip': returned['linear_eip'],
              'indices_sha256': expected_sha, 'png_sha256': sha(png_path), 'RGBA_sha256': hashlib.sha256(expected).hexdigest(),
              'native_palette': palette, 'platform_test_log_sha256': sha(log), 'tool_sha256': sha(Path(__file__))}
    path = output/'source-verification.json'
    path.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'verified': True, 'native_pixels': 256000, 'summary_sha256': sha(path)}))


if __name__ == '__main__':
    main()
