"""獨立核對樹叢來源幾何、三種正常 GUI、建置輸入及原版狀態。"""
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000
    project,output=Path('/project'),Path('/output')
    terrain=project/'workplace/hd-prototype/catalog/terrain'
    root=terrain/'trees'
    source_path=root/'tree-source-verification.json'
    source=json.loads(source_path.read_text())
    assert source['tool_sha256']==sha(project/'tools/hd-terrain-tree-prepare.py')
    assert source['source_summary_sha256']==sha(terrain/'objects/verification-summary.json')
    assert source['source_original_png_sha256']==sha(terrain/'objects/sources/objects-000-original.png')
    assert source['asset_sha256']==sha(project/'assets/hd-b/terrain'/source['asset'])
    assert source['HD_crop']==[7,0,1247,1245] and source['source_opaque_bounds']==[0,0,47,47]
    assert source['maximum_alpha_outside_HD_crop']==1
    assert source['prepared_png_sha256']==sha(root/'tree-prepared-v2.png')
    expected_rects=sorted(tuple(r['rect']) for r in source['records'] if r['validation']=='ready_complete_scene')
    assert len(expected_rects)==3
    baseline_path=project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
    baseline=json.loads(baseline_path.read_text())
    fields=['successful_steps','instruction_budget','stop','stop_instruction_linear_eip','trace_tail',
            'interrupt_counts','dpmi_calls','dpmi_unimplemented','frame','cd_pcm','mounted_disc',
            'mouse_input_events','input_boundary_frames','real_mode_history','real_mode_last']
    records=[]
    for folder_name,mode in [('desktop','trees'),('desktop-only','trees-only'),('desktop-missing','trees-missing')]:
        folder=root/folder_name
        receipt_path,proof_path=folder/f'desktop-{mode}.json',folder/f'verification-{mode}.json'
        receipt,proof=json.loads(receipt_path.read_text()),json.loads(proof_path.read_text())
        assert proof['verified'] and proof['receipt_sha256']==sha(receipt_path)
        assert all(receipt[k]==baseline[k] for k in fields),[k for k in fields if receipt[k]!=baseline[k]]
        assert receipt['successful_steps']==750_000_000 and len(receipt['mouse_input_events'])==390 and receipt['fd2_startup_calls']==0
        assert sha(folder/'startup-memory.bin')=='4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
        for name,digest in receipt['integration_input_sha256'].items():
            assert sha(project/name)==digest,('runtime source changed',name)
            assert sha(folder/'source-snapshot'/name)==digest,('source snapshot changed',name)
        toggles=[e for e in receipt['desktop']['events'] if e['kind']=='toggle']
        assert len(toggles)==3 and all(e['guest_unchanged'] and e['before']==e['after'] for e in toggles)
        state=receipt['desktop']['hd_observation']['terrain']
        assert state['pending_return'] is None and state['completed_returns']==134 and state['completed_sprite_returns']==11
        assert len(state['draws'])==65 and len(state['sprites'])==11 and state['draw_sequence']==145
        info=receipt['desktop']['assets']['terrain_tree']
        assert info['source_sha256']==source['source_original_png_sha256']
        if mode=='trees-missing':
            assert info['validation']=='fallback_missing_asset' and 'hd_sha256' not in info
        else:
            assert info['validation']=='ready' and info['hd_sha256']==source['asset_sha256']
        for other in ['scroll','frame','window']:
            assert receipt['desktop']['assets'][other]['validation']!='ready'
        assert 'hd_sha256' not in receipt['desktop']['assets']
        assert all('hd_sha256' not in v for v in receipt['desktop']['assets']['kao'].values())
        assert all('hd_sha256' not in v for v in receipt['desktop']['assets']['story_portraits'].values())
        if mode!='trees':
            assert all(v['validation']=='fallback_missing_asset' for v in receipt['desktop']['assets']['terrain'].values())
        assert len(proof['tree_checks'])==1
        check=proof['tree_checks'][0]
        actual_rects=[]
        for d in check['active_tree_rectangles']:
            r=d['logical_rect'];actual_rects.append((r['Min']['X'],r['Min']['Y'],r['Max']['X'],r['Max']['Y']))
            assert d['group']=='objects' and d['index']==0
        assert sorted(actual_rects)==([] if mode=='trees-missing' else expected_rects)
        assert len(check['active_floor_rectangles'])==(65 if mode=='trees' else 0)
        assert check['outside_replacement_unchanged'] and check['original_other_foreground_unchanged'] and check['guest_unchanged']
        assert (check['changed_pixels']>0)==(mode!='trees-missing')
        assert check['other_foreground_native_pixels']==receipt['desktop']['tree_protected_pixels']
        assert receipt['desktop']['tree_hd_count']==len(actual_rects)
        if mode=='trees-only':
            assert check['old_outline_output_pixels_removed']>0
        for picture in proof['pictures']:
            assert picture['window_pixels_equal']
            assert sha(folder/Path(picture['actual_png']).relative_to('/output'))==picture['sha256']
        records.append({'mode':mode,'receipt_sha256':sha(receipt_path),'GUI_verification_sha256':sha(proof_path),
                        'binary_sha256':sha(folder/'tokugawa-desktop-prototype'),'verifier_snapshot_sha256':sha(folder/'verify-desktop.py'),
                        'HD_tree_rectangles':len(actual_rects),'HD_floor_rectangles':len(check['active_floor_rectangles']),
                        'changed_pixels':check['changed_pixels'],'protected_original_foreground_pixels':check['other_foreground_native_pixels'],
                        'old_outline_output_pixels_removed':check['old_outline_output_pixels_removed'],'guest_unchanged':True})
    assert len({r['binary_sha256'] for r in records})==1
    test_log=root/'startup-probe/platform-test.log'
    assert sum(line.startswith('ok  ') for line in test_log.read_text().splitlines())==4 and 'FAIL' not in test_log.read_text()
    result={'classification':'objects-000 forest clump v2; three normal complete scenes; clipped tree falls back',
            'whole_game_HD_complete':False,'approved_object_indices':[0],'normal_HD_object_cases':1,'normal_HD_object_instances':3,
            'source_verification_sha256':sha(source_path),'asset_sha256':source['asset_sha256'],
            'baseline_sha256':sha(baseline_path),'compared_fields':fields,'records':records,
            'platform_test_log_sha256':sha(test_log),'tool_sha256':sha(Path(__file__))}
    target=output/'verification-summary.json';target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'normal_HD_tree_instances':3,'summary_sha256':sha(target)}))


if __name__=='__main__':
    main()
