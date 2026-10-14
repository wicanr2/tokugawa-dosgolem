"""核對繪圖前背景接入後的四處樹叢及三模式真實視窗。"""
import argparse
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--runtime-source-root', type=Path, default=Path('/project'))
    parser.add_argument('--verify-only', action='store_true')
    args = parser.parse_args()
    project, output = Path('/project'), Path('/output')
    baseline_path = project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    baseline = json.loads(baseline_path.read_text())
    background_path = output/'background-verification.json'
    source = json.loads(background_path.read_text())
    diagnostic_path = output/'startup-probe/latest.json'
    diagnostic = json.loads(diagnostic_path.read_text())
    assert source['receipt_sha256'] == sha(diagnostic_path) and len(source['records']) == 11
    fields = source['compared_fields']
    records = []
    for folder_name, mode in [('desktop', 'trees'), ('desktop-only', 'trees-only'), ('desktop-missing', 'trees-missing')]:
        folder = output/folder_name
        receipt_path, proof_path = folder/f'desktop-{mode}.json', folder/f'verification-{mode}.json'
        receipt, proof = json.loads(receipt_path.read_text()), json.loads(proof_path.read_text())
        assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
        assert all(receipt[k] == baseline[k] for k in fields)
        assert receipt['successful_steps'] == 750_000_000 and len(receipt['mouse_input_events']) == 390
        assert receipt['fd2_startup_calls'] == 0
        assert sha(folder/'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
        for name, digest in receipt['integration_input_sha256'].items():
            assert sha(args.runtime_source_root/name) == digest, ('runtime source changed', name)
            assert sha(folder/'source-snapshot'/name) == digest, ('snapshot changed', name)
        state = receipt['desktop']['hd_observation']['terrain']
        assert state['pending_return'] is None and state['completed_returns'] == 134
        assert state['completed_sprite_returns'] == 11 and state['draw_sequence'] == 145
        assert len(state['draws']) == 65 and len(state['sprites']) == 11
        for draw in state['sprites']:
            matches = [r for r in diagnostic['sprite_backdrop_records'] if r['draw']['group'] == draw['group'] and r['draw']['index'] == draw['index'] and r['draw']['logical_rect'] == draw['logical_rect']]
            assert len(matches) == 1 and draw['background_verified']
            row = matches[0]
            assert (draw['BeginStep'], draw['EndStep']) == (row['begin_step'], row['end_step'])
            for field, key in [('background_rgba_sha256', 'before'), ('source_rgba_sha256', 'source'), ('after_rgba_sha256', 'after')]:
                assert draw[field] == row[key]['rgba_sha256']
        toggles = [e for e in receipt['desktop']['events'] if e['kind'] == 'toggle']
        assert len(toggles) == 3 and all(e['guest_unchanged'] and e['before'] == e['after'] for e in toggles)
        info = receipt['desktop']['assets']['terrain_tree']
        if mode == 'trees-missing':
            assert info['validation'] == 'fallback_missing_asset' and 'hd_sha256' not in info
        else:
            assert info['validation'] == 'ready'
            assert info['hd_sha256'] == sha(project/'assets/hd-b/terrain/object-000-hd-b-v2.png')
        assert len(proof['tree_checks']) == 1
        check = proof['tree_checks'][0]
        expected = 0 if mode == 'trees-missing' else 4
        assert len(check['active_tree_rectangles']) == receipt['desktop']['tree_hd_count'] == expected
        assert len(check['active_floor_rectangles']) == (65 if mode == 'trees' else 0)
        assert check['captured_background_native_pixels'] == (480 if expected else 0)
        assert check['original_UI_output_pixels_preserved'] == (772 if expected else 0)
        assert check['outside_replacement_unchanged'] and check['original_other_foreground_unchanged'] and check['guest_unchanged']
        assert check['other_foreground_native_pixels'] == receipt['desktop']['tree_protected_pixels'] == (82 if expected else 0)
        assert (check['changed_pixels'] > 0) == bool(expected)
        if mode == 'trees-only':
            assert check['old_outline_output_pixels_removed'] > 1394
        for picture in proof['pictures']:
            assert picture['window_pixels_equal']
            assert sha(folder/Path(picture['actual_png']).relative_to('/output')) == picture['sha256']
        records.append({'mode': mode, 'receipt_sha256': sha(receipt_path), 'GUI_verification_sha256': sha(proof_path),
                        'binary_sha256': sha(folder/'tokugawa-desktop-prototype'), 'verifier_snapshot_sha256': sha(folder/'verify-desktop.py'),
                        'HD_tree_rectangles': expected, 'HD_floor_rectangles': len(check['active_floor_rectangles']),
                        'changed_pixels': check['changed_pixels'], 'protected_original_foreground_pixels': check['other_foreground_native_pixels'],
                        'captured_background_native_pixels': check['captured_background_native_pixels'],
                        'original_UI_output_pixels_preserved': check['original_UI_output_pixels_preserved'],
                        'old_outline_output_pixels_removed': check['old_outline_output_pixels_removed'], 'guest_unchanged': True})
    assert len({r['binary_sha256'] for r in records}) == 1
    log = output/'runtime/startup-probe/platform-test.log'
    assert sum(l.startswith('ok  ') for l in log.read_text().splitlines()) == 4 and 'FAIL' not in log.read_text()
    result = {'classification': 'objects-000 forest clump v2; four normal scenes including original UI edge background',
              'whole_game_HD_complete': False, 'approved_object_indices': [0], 'normal_HD_object_cases': 1, 'normal_HD_object_instances': 4,
              'background_verification_sha256': sha(background_path),
              'asset_sha256': sha(project/'assets/hd-b/terrain/object-000-hd-b-v2.png'),
              'baseline_sha256': sha(baseline_path), 'compared_fields': fields, 'records': records,
              'platform_test_log_sha256': sha(log), 'tool_sha256': sha(Path(__file__))}
    target = output/'verification-summary.json'
    if not args.verify_only:
        target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'verified': True, 'normal_HD_tree_instances': 4, 'summary_sha256': sha(target) if target.exists() else None}))


if __name__ == '__main__':
    main()
