"""彙整 MAPGAMEN 框架、背景與正常視窗三模式驗收。"""
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
    source_path, geometry_path = output/'source-verification.json', output/'battle-frame-geometry-v2.json'
    source, geometry = json.loads(source_path.read_text()), json.loads(geometry_path.read_text())
    assert geometry['source_verification_sha256'] == sha(source_path)
    assert geometry['tool_sha256'] == sha(project/'tools/hd-battle-frame-prepare.py')
    assert geometry['mask_native_pixels'] == 100853 and geometry['weather_native_full_rectangle_match']
    assert geometry['weather_region'] == [464,291,608,365]
    asset_path = project/'assets/hd-b/battle-frame/battle-frame-hd-b-v2.png'
    assert sha(asset_path) == geometry['prepared_png_sha256']
    assert sha(output/'frame-mask-v2.idx') == geometry['mask_sha256']
    baseline_path = project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    baseline = json.loads(baseline_path.read_text())
    fields = source['compared_fields']
    records = []
    for folder_name, mode in [('desktop','battle-frame'),('desktop-only','battle-frame-only'),('desktop-missing','battle-frame-missing')]:
        folder = output/folder_name
        receipt_path, proof_path = folder/f'desktop-{mode}.json', folder/f'verification-{mode}.json'
        receipt, proof = json.loads(receipt_path.read_text()), json.loads(proof_path.read_text())
        assert proof['verified'] and proof['receipt_sha256'] == sha(receipt_path)
        assert all(receipt[k] == baseline[k] for k in fields)
        assert receipt['successful_steps'] == 750_000_000 and len(receipt['mouse_input_events']) == 390
        assert receipt['fd2_startup_calls'] == 0
        assert sha(folder/'startup-memory.bin') == '4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
        for name, digest in receipt['integration_input_sha256'].items():
            assert sha(args.runtime_source_root/name) == digest, ('runtime source changed',name)
            assert sha(folder/'source-snapshot'/name) == digest, ('snapshot changed',name)
        state = receipt['desktop']['hd_observation']
        assert state['battle_frame']['draw_completed'] and state['battle_frame']['pending_return'] is None
        assert state['battle_frame']['capture_step'] == source['return_step']
        assert state['terrain']['completed_returns'] == 134 and state['terrain']['completed_sprite_returns'] == 11
        assert state['terrain']['pending_return'] is None and len(state['terrain']['draw_history']) == 134
        toggles = [e for e in receipt['desktop']['events'] if e['kind'] == 'toggle']
        assert len(toggles) == 5 and all(e['guest_unchanged'] and e['before'] == e['after'] for e in toggles)
        info = receipt['desktop']['assets']['battle_frame']
        assert info['source_indices_sha256'] == source['indices_sha256'] and info['source_mask_sha256'] == geometry['mask_sha256']
        missing = mode == 'battle-frame-missing'
        if missing:
            assert info['validation'] == 'fallback_missing_asset' and 'hd_sha256' not in info
        else:
            assert info['validation'] == 'ready' and info['hd_sha256'] == sha(asset_path)
        checks = proof['battle_frame_checks']
        assert len(checks) == 2 and checks[0]['phase'] == 'intro' and checks[1]['phase'] == 'battle'
        intro, battle = checks
        assert intro['guest_unchanged'] and intro['native_source_verified'] and intro['reference_composite_verified']
        assert not intro['crest_HD'] and intro['weather_placeholder_HD'] == (not missing)
        assert (intro['changed_pixels'] > 0) == (not missing)
        assert intro['tool_sha256'] == sha(project/'tools/verify_hd_battle_frame_pixels.py') == sha(folder/'verify_hd_battle_frame_pixels.py')
        only = mode == 'battle-frame-only'
        assert battle['HD_floor_draws'] == (0 if only else 67)
        assert battle['HD_tree_draws'] == (0 if only else 4) and battle['HD_unit_draws'] == (0 if only else 2)
        assert battle['HD_reference_composite_verified'] and battle['native_source_scene_verified'] and battle['outside_qualified_layers_unchanged'] and battle['guest_unchanged']
        assert battle['pixel_verifier_sha256'] == sha(project/'tools/verify_hd_scene_pixels.py') == sha(folder/'verify_hd_scene_pixels.py')
        if not missing:
            assert battle['frame_native_painted'] > 70000 and battle['frame_reference_output_pixels_verified'] > 250000
            if not only:
                assert battle['HD_frame_underlay_output_pixels'] > 0
        else:
            assert battle['frame_native_painted'] == 0 and battle['HD_frame_underlay_output_pixels'] == 0
            assert battle['changed_pixels'] == 412342
        assert len(proof['pictures']) == 6
        for picture in proof['pictures']:
            assert picture['window_pixels_equal']
            assert sha(folder/Path(picture['actual_png']).relative_to('/output')) == picture['sha256']
            native = picture['presented']['frame']['native_indices']
            assert sha(folder/native['path']) == native['sha256'] == picture['presented']['proof']['raw_indexed_sha256']
        records.append({'mode':mode,'receipt_sha256':sha(receipt_path),'GUI_verification_sha256':sha(proof_path),
                        'binary_sha256':sha(folder/'tokugawa-desktop-prototype'),'verifier_snapshot_sha256':sha(folder/'verify-desktop.py'),
                        'intro':intro,'battle':battle})
    assert len({r['binary_sha256'] for r in records}) == 1
    log = output/'runtime/startup-probe/platform-test.log'
    assert sum(l.startswith('ok  ') for l in log.read_text().splitlines()) == 4 and 'FAIL' not in log.read_text()
    result = {'classification':'source-verified MAPGAMEN frame skin v2; pre-battle modal and first battle, three normal GUI modes',
              'whole_game_HD_complete':False,'source_verification_sha256':sha(source_path),'geometry_sha256':sha(geometry_path),
              'asset_sha256':sha(asset_path),'baseline_sha256':sha(baseline_path),'compared_fields':fields,
              'records':records,'platform_test_log_sha256':sha(log),'tool_sha256':sha(Path(__file__))}
    target = output/'verification-summary.json'
    if not args.verify_only:
        target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'normal_frame_scenes':2,'summary_sha256':sha(target) if target.exists() else None}))


if __name__ == '__main__':
    main()
