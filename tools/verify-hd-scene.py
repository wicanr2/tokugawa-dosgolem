"""彙整地面完整順序與重疊 HD 人物的正常路徑驗證。"""
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
    source_path = output/'source-scene-verification.json'
    source = json.loads(source_path.read_text())
    assert source['tool_sha256'] == sha(project/'tools/hd-scene-probe.py')
    assert source['matching_visible_floor_draws'] == 67 and source['matching_visible_sprite_draws'] == 6
    baseline_path = project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    baseline = json.loads(baseline_path.read_text())
    background_path = project/'workplace/hd-prototype/catalog/terrain/backgrounds/background-verification.json'
    background = json.loads(background_path.read_text())
    fields = background['compared_fields']
    records = []
    unit_assets = {1: ('unit-001-hd-b-v1.png', 'a8c37f61ef22f645a58cb824758ab34620e535a2588f0fce5e71490cf3dd3a72'),
                   3: ('unit-003-hd-b-v1.png', '6c9a0ef3d9a3db01bec673fb37d7f8fca4a2f2a08cf6a14d81ecd101ec867654')}
    for folder_name, mode in [('desktop', 'scene'), ('desktop-only', 'scene-only'), ('desktop-missing', 'scene-missing')]:
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
        assert len(state['draws']) == 65 and len(state['draw_history']) == 134 and len(state['sprites']) == 11
        expected = [r for r in source['records'] if r['group'] == 'floor']
        for d, e in zip(state['draw_history'], expected):
            r = d['logical_rect']
            assert (d['index'], d['draw_order'], [r['Min']['X'], r['Min']['Y'], r['Max']['X'], r['Max']['Y']]) == (e['index'], e['order'], e['rect'])
        assert receipt['desktop']['terrain_scene_enabled']
        for index, (name, digest) in unit_assets.items():
            info = receipt['desktop']['assets']['terrain_units'][str(index)]
            assert info['source_sha256'] == sha(project/f'workplace/hd-prototype/catalog/terrain/objects/sources/units-{index:03d}-original.png')
            if mode == 'scene-missing':
                assert info['validation'] == 'fallback_missing_asset' and 'hd_sha256' not in info
            else:
                assert info['validation'] == 'ready' and info['hd_sha256'] == digest
                assert sha(project/'assets/hd-b/terrain'/name) == digest
        toggles = [e for e in receipt['desktop']['events'] if e['kind'] == 'toggle']
        assert len(toggles) == 3 and all(e['guest_unchanged'] and e['before'] == e['after'] for e in toggles)
        assert len(proof['scene_checks']) == 1
        check = proof['scene_checks'][0]
        report = receipt['desktop']['terrain_scene']
        assert check['HD_floor_draws'] == report['floor_hd_count'] == (67 if mode == 'scene' else 0)
        assert check['HD_tree_draws'] == report['tree_hd_count'] == (0 if mode == 'scene-missing' else 4)
        assert check['HD_unit_draws'] == report['unit_hd_count'] == (0 if mode == 'scene-missing' else 2)
        assert check['floor_native_pixels'] == report['floor_native_pixels'] == (102720 if mode == 'scene' else 0)
        assert check['original_visible_unit_pixels_overlapping_HD_tree'] == (0 if mode == 'scene-missing' else 82)
        assert check['HD_reference_composite_verified'] and check['native_source_scene_verified'] and check['outside_qualified_layers_unchanged'] and check['guest_unchanged']
        assert check['pixel_verifier_sha256'] == sha(project/'tools/verify_hd_scene_pixels.py') == sha(folder/'verify_hd_scene_pixels.py')
        assert check['protected_original_foreground_pixels'] == report['protected_original_foreground_pixels'] == 0
        assert check['original_UI_output_pixels_preserved'] == report['preserved_UI_output_pixels']
        if mode != 'scene-missing':
            assert check['changed_pixels'] > 0 and check['independent_output_pixels_verified'] > 20000
            assert all(n > 0 for n in check['old_unit_outline_output_pixels_removed'].values())
        else:
            assert check['changed_pixels'] == 0
        for picture in proof['pictures']:
            assert picture['window_pixels_equal']
            assert sha(folder/Path(picture['actual_png']).relative_to('/output')) == picture['sha256']
        records.append({'mode': mode, 'receipt_sha256': sha(receipt_path), 'GUI_verification_sha256': sha(proof_path),
                        'binary_sha256': sha(folder/'tokugawa-desktop-prototype'), 'verifier_snapshot_sha256': sha(folder/'verify-desktop.py'),
                        **{k: v for k, v in check.items() if k not in ('step',)}})
    assert len({r['binary_sha256'] for r in records}) == 1
    log = output/'runtime/startup-probe/platform-test.log'
    assert sum(l.startswith('ok  ') for l in log.read_text().splitlines()) == 4 and 'FAIL' not in log.read_text()
    result = {'classification': 'normal full-floor journal and jointly composited forest clumps plus units-001/003',
              'whole_game_HD_complete': False, 'approved_unit_indices': [1, 3], 'normal_HD_unit_cases': 2,
              'normal_HD_unit_instances': 2, 'approved_object_indices': [0], 'normal_HD_object_cases': 1, 'normal_HD_object_instances': 4,
              'normal_HD_floor_draws': 67, 'normal_HD_floor_indices': [0, 2, 6, 13, 15, 22], 'normal_HD_floor_native_pixels': 102720,
              'source_verification_sha256': sha(source_path), 'background_verification_sha256': sha(background_path),
              'baseline_sha256': sha(baseline_path), 'compared_fields': fields, 'records': records,
              'platform_test_log_sha256': sha(log), 'tool_sha256': sha(Path(__file__))}
    target = output/'verification-summary.json'
    if not args.verify_only:
        target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'verified': True, 'HD_floor_draws': 67, 'HD_unit_instances': 2, 'HD_object_instances': 4,
                      'summary_sha256': sha(target) if target.exists() else None}))


if __name__ == '__main__':
    main()
