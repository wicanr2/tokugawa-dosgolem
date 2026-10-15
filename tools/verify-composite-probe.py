"""DRAFT 唯讀探針的原版基準及實際來源／目的驗證。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
from PIL import Image

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--control-only', action='store_true')
p.add_argument('--source-root', type=Path, default=Path('/project'))
args = p.parse_args()
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def local(path):
    return root / Path(path).relative_to('/output')


def rect(value):
    return value['Min']['X'], value['Min']['Y'], value['Max']['X'], value['Max']['Y']


sources = read(root / 'build-sources.json')
assert len(sources) in [105, 106] and all(sha(args.source_root / n) == h for n, h in sources.items())
baseline = read(project / 'workplace/desktop-prototype/headless/startup-probe/latest.json')
control = root / 'control'
c = read(control / 'receipt.json')
fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip', 'trace_tail',
          'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame', 'cd_pcm', 'mounted_disc',
          'mouse_input_events', 'input_boundary_frames', 'real_mode_history', 'real_mode_last']
assert all(c[k] == baseline[k] for k in fields), [k for k in fields if c[k] != baseline[k]]
assert c['integration_input_sha256'] == sources and not c['composite_diagnostic_enabled'] and not c['desktop_script_handoff']
assert sha(control / 'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
control_report = dict(result='PASS', fields_equal=fields, ram_sha256=sha(control / 'startup-memory.bin'),
                      receipt_sha256=sha(control / 'receipt.json'), source_count=len(sources), tool_sha256=sha(Path(__file__)))
(control / 'verification.json').write_text(json.dumps(control_report, ensure_ascii=False, indent=2) + '\n')
if args.control_only:
    print('PASS original 750M control, 390 events and full RAM')
    raise SystemExit(0)

exploration, r = read(root / 'exploration-status.json'), read(root / 'receipt.json')
assert exploration['phase'] == 'finished' and r['stop'] == 'desktop_closed'
assert r['composite_diagnostic_enabled'] and r['desktop_script_handoff']
assert r['integration_input_sha256'] == sources and r['fd2_startup_calls'] == 0
assert r['mouse_input_schedule'] == baseline['mouse_input_schedule']
assert r['mouse_input_events'][:390] == baseline['mouse_input_events']
assert r['input_boundary_frames'] == baseline['input_boundary_frames']
assert all(e.get('source') == 'desktop' for e in r['mouse_input_events'][390:])
profiles = {v['name']: v for v in read(project / 'workplace/hd-prototype/catalog/decoded/inventory.json')['records']}
verified, rejected = [], []
for record in r['composite_draw_trace'].get('records') or []:
    if not record['verified']:
        rejected.append(record)
        continue
    assert record['Error'] in ['', None] if 'Error' in record else not record.get('observation_error')
    profile = profiles[record['Source']]
    source = local(record['SourcePath']).read_bytes()
    before, after = local(record['BeforePath']).read_bytes(), local(record['AfterPath']).read_bytes()
    assert hashlib.sha256(source).hexdigest() == record['SourceSHA'] == profile['indices_sha256']
    width, height = profile['width'], profile['height']
    assert len(source) == width * height
    x0, y0, x1, y1 = rect(record['SourceRect'])
    dx0, dy0, dx1, dy1 = rect(record['DestRect'])
    assert 0 <= x0 < x1 <= width and 0 <= y0 < y1 <= height
    assert 0 <= dx0 < dx1 <= 640 and 0 <= dy0 < dy1 <= 400
    selected = b''.join(source[y * width + x0:y * width + x1] for y in range(y0, y1))
    assert len(before) == len(after) == len(selected) == (dx1 - dx0) * (dy1 - dy0)
    if record['Kind'] == 'or_full':
        assert record['Entry'] == 0x1c291 and len(before) == 640 * 400
        expected = bytes(a | b for a, b in zip(before, selected))
    else:
        assert record['Kind'] == 'masked_rect' and record['Entry'] == 0x1c35b
        expected = bytes(b if b else a for a, b in zip(before, selected))
    assert expected == after
    assert hashlib.sha256(expected).hexdigest() == record['ExpectedSHA'] == record['ActualSHA']
    assert hashlib.sha256(before).hexdigest() == record['BeforeSHA']
    verified.append(record)
for capture in exploration['captures']:
    actual_path = local(capture['actual'])
    assert sha(actual_path) == capture['sha256'] and capture['window_pixels_equal']
    actual = Image.open(actual_path).convert('RGBA')
    presented = Image.open(local(capture['event']['frame']['path'])).convert('RGBA')
    if actual.height == 864:
        actual = actual.crop((0, 64, 1280, 864))
    assert actual.size == presented.size == (1280, 800) and actual.tobytes() == presented.tobytes()
report = dict(result='PASS' if verified else 'INCOMPLETE', scope='read-only original composite diagnostics and ordinary battle input',
              HD_rendering_integrated=False, sources_verified=sorted({v['Source'] for v in verified}),
              source_count=len(sources), original_control=control_report, native_composites=verified,
              rejected_observations=rejected, dropped_events=r['composite_draw_trace']['dropped_events'],
              normal_opaque_sources=sorted({d['resource'] for d in r['graphics_draw_trace'].get('records') or [] if d['verified']}),
              ordinary_events_after_prefix=len(r['mouse_input_events'])-390, actual_window_captures=len(exploration['captures']),
              receipt_sha256=sha(root / 'receipt.json'), tool_sha256=sha(Path(__file__)))
(root / 'verification-summary.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
print(json.dumps({k: report[k] for k in ['result', 'HD_rendering_integrated', 'sources_verified', 'actual_window_captures']}))
