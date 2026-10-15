"""重查預設啟動鏈的原版收據、設定沿用、完整 RAM 與實際視窗。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
from PIL import Image

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--source-root', type=Path, default=Path('/project'))
args = p.parse_args()
root, project = Path('/output'), Path('/project')
assert root.stat().st_uid == 1000


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


sources = read(root / 'build-sources.json')
assert len(sources) == 103
assert all(sha(args.source_root / n) == h for n, h in sources.items())
binary = sha(root / 'tokugawa-desktop-prototype')
game = read(project / 'workplace/desktop-prototype/headless/startup-probe/latest.json')
mark = read(project / 'workplace/hd-prototype/catalog/remaining/consumers/mark-probe/r8/startup-probe/receipt-75m.json')
fields = ['successful_steps', 'instruction_budget', 'stop', 'stop_instruction_linear_eip', 'trace_tail',
          'interrupt_counts', 'dpmi_calls', 'dpmi_unimplemented', 'frame', 'cd_pcm', 'mounted_disc',
          'mouse_input_events', 'input_boundary_frames', 'real_mode_history', 'real_mode_last']
cases = []
for mode in ['original', 'hd', 'cancel']:
    case = root / mode
    v, r = read(case / 'verification.json'), read(case / 'receipt.json')
    assert v['result'] == 'PASS' and v['mode'] == mode and v['receipt_sha256'] == sha(case / 'receipt.json')
    assert v['binary_sha256'] == sha(case / 'tokugawa-desktop-prototype') == binary
    assert v['verifier_sha256'] == sha(root / 'source-snapshot/tools/verify-startup-chain.py')
    prelude, child = r['publisher_prelude'], r['publisher_prelude']['receipt']
    child_path = case / Path(prelude['receipt_path']).relative_to('/output')
    assert sha(child_path) == prelude['receipt_sha256'] and read(child_path) == child
    assert child['integration_input_sha256'] == sources
    entry = dict(mode=mode, result='PASS', receipt_sha256=sha(case / 'receipt.json'),
                 verification_sha256=sha(case / 'verification.json'), publisher_receipt_sha256=sha(child_path))
    if mode == 'cancel':
        assert r['stop'] == child['stop'] == 'desktop_closed'
        assert r['game_started'] is False and not prelude['continued']
        assert child['successful_steps'] == 0 and not child['mouse_input_events']
        assert not (case / 'startup-memory.bin').exists()
        entry.update(game_started=False, publisher_steps=0)
    else:
        assert prelude['continued'] and prelude['next_display'] == mode
        assert r['integration_input_sha256'] == sources and r['game_started']
        assert all(r[k] == game[k] for k in fields)
        mark_fields = [k for k in fields if k not in ['mouse_input_events', 'input_boundary_frames']]
        assert all(child[k] == mark[k] for k in mark_fields)
        assert child['stop'] == 'dos_exit' and child['exit_code'] == child['fd2_startup_calls'] == 0
        assert child['desktop']['hd_enabled'] == (mode == 'hd')
        assert child['desktop']['startup_selection'] != mode
        assert len(child['desktop']['startup_selection_events']) == 1
        assert all(e['guest_unchanged'] and e['before']['step'] == e['after']['step'] == 0
                   for e in child['desktop']['startup_selection_events'])
        first = read(case / 'first-game-status.json')
        assert first['program'] == 'game' and first['enabled'] == (mode == 'hd')
        assert first['startup_selection'] == mode and not first['startup_pending'] and not first['startup_events']
        assert sha(case / 'mark-startup-memory.bin') == '62da54cc0cef42e9a55c64cb1ac958b05b851f19c12247023f6feddd023fa085'
        assert sha(case / 'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
        for c in v['mark_captures'] + v['game_captures']:
            actual_path = case / Path(c['actual']).relative_to('/output')
            assert sha(actual_path) == c['sha256'] and c['window_pixels_equal']
            actual = Image.open(actual_path).convert('RGBA')
            presented = Image.open(case / Path(c['event']['frame']['path']).relative_to('/output')).convert('RGBA')
            assert actual.size == (presented.width, presented.height + 64)
            assert actual.crop((0, 64, presented.width, presented.height + 64)).tobytes() == presented.tobytes()
        assert len(v['mark_captures']) == 2 and len(v['game_captures']) == 3
        assert v['mark_captures'][0]['event']['proof'] == v['mark_captures'][1]['event']['proof']
        assert all(c['event']['proof'] == v['game_captures'][0]['event']['proof'] for c in v['game_captures'])
        assert v['game_captures'][0]['event']['frame']['rgba_sha256'] == v['game_captures'][2]['event']['frame']['rgba_sha256']
        entry.update(game_started=True, publisher_steps=child['successful_steps'], game_steps=r['successful_steps'],
                     mouse_events=len(r['mouse_input_events']), game_fields_equal=fields, publisher_fields_equal=mark_fields,
                     mark_ram_sha256=sha(case / 'mark-startup-memory.bin'), game_ram_sha256=sha(case / 'startup-memory.bin'),
                     actual_window_captures=5)
    cases.append(entry)
summary = dict(result='PASS', game_startup_integrated=True, normal_last_displays=['original', 'hd'],
               verifier_sha256=sha(Path(__file__)),
               binary_sha256=binary, source_count=len(sources), sources_sha256=sha(root / 'build-sources.json'), cases=cases,
               scope='default desktop selection, original MARK and DX5 first battle; independent LE sessions and last display setting',
               limitations=['remaining HD animation consumers incomplete', 'desktop audio and other operating systems not verified',
                            'not DOS shell EXEC or shared virtual-clock parity'])
(root / 'verification-summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(dict(result='PASS', summary_sha256=sha(root / 'verification-summary.json'))))
