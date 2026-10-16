"""核對 END-2 條件式接入；來源指令與正常結局驗收分開。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image

assert Path('/.dockerenv').is_file() and os.getuid() == 1000
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000
parser = argparse.ArgumentParser()
parser.add_argument('--source-root', type=Path, default=project)
args = parser.parse_args()
read = lambda path: json.loads(path.read_text())
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
digest = lambda data: hashlib.sha256(data).hexdigest()
sources = read(root/'build-sources.json')
assert len(sources) == 106
assert all(sha(args.source_root/path) == value for path, value in sources.items())
source = read(root/'ending-source-verification.json')
assert source['verified'] and not source['normal_ending_verified']
assert source['instructions_executed'] == 57606720 and source['writes_verified'] == 11520000
assert source['probe_source_sha256'] == sha(root/'ending-source-probe.go') == sha(args.source_root/'tools/ending-source-probe.go')
assert source['cpu_source_sha256'] == sha(root/'engine/internal/cpu386/cpu.go')
assert source['input_sha256'] == sha(project/'original/game/DX5.EXE')
assert len(source['IDA_exports']) == 3
for export in source['IDA_exports']:
    path = Path(export['path'])
    assert sha(path) == export['sha256']
    document = read(path)
    assert document['tool'] == 'IDA Pro' and document['version'] == '9.4'
    assert document['input_sha256'] == source['input_sha256']
    assert len(document['rows']) == export['loaded_rows_verified']
original = (project/'workplace/hd-prototype/catalog/decoded/END-2.OZM.idx').read_bytes()
assert len(original) == 256000 and digest(original) == source['source_indices_sha256']
assert len(source['states']) == 241
saved = []
for count, state in enumerate(source['states']):
    assert state['count'] == count
    current = bytes(original[min(i+count, 96000)] for i in range(96000)) + original[96000:]
    crop = b''.join(current[y*640:y*640+400] for y in range(150))
    assert digest(current) == state['indices_sha256'] and digest(crop) == state['display_crop_sha256']
    if 'planes_path' in state:
        packed = (root/state['planes_path']).read_bytes()
        assert len(packed) == 128000 and digest(packed) == state['planes_sha256']
        actual = bytearray(256000)
        for bit in range(4):
            plane = packed[bit*32000:(bit+1)*32000]
            for i, value in enumerate(plane):
                for x in range(8):
                    actual[i*8+x] |= ((value >> (7-x)) & 1) << bit
        assert bytes(actual) == current
        saved.append(count)
assert saved == [0, 1, 120, 240]
log = (root/'startup-probe/platform-test.log').read_text().splitlines()
assert len(log) == 4 and all(line.startswith('ok\t') or line.startswith('ok  \t') for line in log)
reference = project/'workplace/hd-prototype/catalog/remaining/consumers/battle/r23'
current, before = read(root/'native-control/receipt.json'), read(reference/'native-control/receipt.json')
fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip', 'trace_tail',
          'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame', 'cd_pcm', 'mounted_disc',
          'mouse_input_events', 'input_boundary_frames', 'real_mode_history', 'real_mode_last',
          'composite_draw_trace', 'graphics_draw_trace']
assert all(current[key] == before[key] for key in fields), [key for key in fields if current[key] != before[key]]
assert sha(root/'native-control/startup-memory.bin') == sha(reference/'native-control/startup-memory.bin') == '28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d'
assert current['integration_input_sha256'] == sources and current['successful_steps'] == 600000000
assert current['fd2_startup_calls'] == current['native_runtime_instructions_skipped'] == 0
case = root/'desktop-map'
exploration, receipt = read(case/'exploration-status.json'), read(case/'receipt.json')
assert exploration['phase'] == 'finished' and not exploration['harness_only']
assert receipt['stop'] == 'desktop_closed' and receipt['integration_input_sha256'] == sources
assert receipt['fd2_startup_calls'] == receipt['native_runtime_instructions_skipped'] == 0
assert len(receipt['mouse_input_schedule']) == 9 and receipt['mouse_input_schedule'] == current['mouse_input_schedule'][:9]
assert exploration['input_data']['kind'] == 'original-game save'
assert exploration['input_data']['dat_sha256'] == '3bf1827150e9955dd169c0d464b85a11a9ce2ff3e0764a7ecda4cb7a3678e1b5'
assert exploration['tool_sha256'] == sha(case/'source-snapshot/tools/explore-battle.py')
assert exploration['binary_sha256'] == sha(root/'tokugawa-desktop-prototype') == sha(case/'tokugawa-desktop-prototype')
old_captures = {c['label']: c for c in read(reference/'desktop-map/exploration-status.json')['captures']}
captures = {}
for capture in exploration['captures']:
    actual = case/Path(capture['actual']).relative_to('/output')
    presented = case/Path(capture['event']['frame']['path']).relative_to('/output')
    assert sha(actual) == capture['sha256'] and capture['window_pixels_equal']
    assert sha(presented) == capture['event']['frame']['png_sha256']
    image = Image.open(actual).convert('RGBA')
    if image.size == (1280, 864):
        image = image.crop((0, 64, 1280, 864))
    output = Image.open(presented).convert('RGBA')
    assert image.size == output.size == (1280, 800) and image.tobytes() == output.tobytes()
    old = old_captures[capture['label']]
    old_path = reference/'desktop-map'/Path(old['event']['frame']['path']).relative_to('/output')
    assert output.tobytes() == Image.open(old_path).convert('RGBA').tobytes()
    captures[capture['label']] = capture
assert set(captures) == {'saved-checkpoint', 'action-1', 'action-2'}
assert captures['saved-checkpoint']['event']['proof'] == captures['action-1']['event']['proof'] == captures['action-2']['event']['proof']
toggles = [event for event in read(case/'desktop-status.json')['events'] if event['kind'] == 'toggle']
assert len(toggles) == 2 and all(event['guest_unchanged'] for event in toggles)
assert all(sha(case/'source-snapshot'/path) == value for path, value in sources.items())
summary = dict(status='PASS', classification='conditional END-2 source and consumer integration; normal map regression',
               source_count=len(sources), original_instruction_states_verified=241,
               actual_instruction_fixture_counts=saved, original_instructions_executed=source['instructions_executed'],
               full_original_ram_equal=True, original_fields_equal=len(fields),
               ordinary_saved_game_steps=current['successful_steps'], actual_window_captures=3, state_unchanged_toggles=2,
               saved_game_display_equals_r23=True, ending_shift_consumer_registered=True,
               normal_ending_GUI_verified=False, whole_game_HD_complete=False,
               tool_sha256=sha(Path(__file__)), source_proof_sha256=sha(root/'ending-source-verification.json'),
               source_probe_tool_sha256=source['probe_source_sha256'], compiled_sources=sources,
               normal_control_receipt_sha256=sha(root/'native-control/receipt.json'),
               ordinary_GUI_receipt_sha256=sha(case/'receipt.json'))
(root/'verification-summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
print(json.dumps({key: value for key, value in summary.items() if key != 'compiled_sources'}, ensure_ascii=False))
