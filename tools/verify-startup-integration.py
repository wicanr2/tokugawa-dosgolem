"""核對啟動選擇與既有首戰基準，來源與收據分別固定。"""
from pathlib import Path
import hashlib
import json
import os
import shutil

assert Path('/.dockerenv').exists() and os.getuid() == 1000
root = Path('/output')
project = Path('/project')
assert root.stat().st_uid == 1000

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def read(path):
    return json.loads(path.read_text())

sources = read(root/'build-sources.json')
assert len(sources) == 93
assert all(sha(project/name)==digest for name,digest in sources.items())
binary = sha(root/'tokugawa-desktop-prototype')
reports=[]
for mode in ['original','hd']:
    case=root/(mode+'-r2')
    v=read(case/'verification.json')
    r=read(case/'receipt.json')
    assert v['result']=='PASS' and v['binary_sha256']==binary==sha(case/'tokugawa-desktop-prototype')
    assert r['integration_input_sha256']==sources
    assert r['desktop']['startup_selection']==mode
    assert r['successful_steps']==200000000 and r['stop']=='instruction_budget'
    assert sha(case/'startup-memory.bin')==v['ram_sha256']
    e=r['desktop']['startup_selection_events'][0]
    assert e['before']==e['after'] and e['guest_unchanged'] and e['before']['step']==0
    # Recover the exact first successful verifier from its preserved prior revision.
    text=(root/'original/verifier-failed.py').read_text()
    text=text.replace("        run('xdotool','mouseup','1')\n    assert s['startup_selection']", "        run('xdotool','mousemove','--window',window,'700','500')\n        run('xdotool','mouseup','1')\n    assert s['startup_selection']")
    text=text.replace("    if mode=='original':", "    run('xdotool','mousemove','--window',window,'700','500')\n    if mode=='original':")
    text=text.replace("    assert not receipt['mouse_input_events'] and receipt['fd2_startup_calls']==0", "    assert all(e['buttons']==0 for e in receipt['mouse_input_events']) and receipt['fd2_startup_calls']==0")
    text=text.replace("            'binary_sha256':", "            'verifier_sha256':hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),\n            'binary_sha256':")
    assert hashlib.sha256(text.encode()).hexdigest()==v['verifier_sha256'], 'verifier revision must match its recorded hash'
    target=case/'verifier.py'
    if target.exists(): assert target.read_text()==text
    else: target.write_text(text)
    reports.append({'selection':mode,'receipt_sha256':sha(case/'receipt.json'),'verification_sha256':sha(case/'verification.json'),
                    'ram_sha256':v['ram_sha256'],'verifier_sha256':v['verifier_sha256']})
assert reports[0]['ram_sha256']==reports[1]['ram_sha256']
preview=read(root/'preview/startup-preview.json')
assert preview['result']=='PASS' and preview['binary_sha256']==binary and preview['step']==preview['mouse_events']==0
assert sha(root/'preview/startup-choice.png')==preview['png_sha256']

case=root/'regression'
r=read(case/'desktop-scene.json')
v=read(case/'verification-scene.json')
baseline=project/'workplace/desktop-prototype/headless/startup-probe/latest.json'
b=read(baseline)
fields=['successful_steps','instruction_budget','stop','stop_instruction_linear_eip','trace_tail','interrupt_counts','dpmi_calls',
        'dpmi_unimplemented','frame','cd_pcm','mounted_disc','mouse_input_events','input_boundary_frames','real_mode_history','real_mode_last']
differences=[key for key in fields if r[key]!=b[key]]
assert not differences, differences
assert r['integration_input_sha256']==sources and r['fd2_startup_calls']==0
assert v['verified'] and len(v['pictures'])==3
assert sha(case/'tokugawa-desktop-prototype')==binary
ram=sha(case/'startup-memory.bin')
assert ram=='4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
assert all(p['window_pixels_equal'] for p in v['pictures'])

snapshot=root/'source-snapshot'
for name,digest in sources.items():
    target=snapshot/name
    target.parent.mkdir(parents=True,exist_ok=True)
    if target.exists(): assert sha(target)==digest
    else: shutil.copy2(project/name,target)
report={'result':'PASS','scope':'startup display selection and toolbar; MARK restoration remains incomplete',
        'binary_sha256':binary,'source_count':len(sources),'sources_sha256':sha(root/'build-sources.json'),
        'startup_modes':reports,'startup_preview':preview,'regression':{'successful_steps':r['successful_steps'],
        'mouse_events':len(r['mouse_input_events']),'ram_sha256':ram,'original_fields_equal':fields,
        'receipt_sha256':sha(case/'desktop-scene.json'),'baseline_receipt_sha256':sha(baseline),'actual_window_captures':len(v['pictures'])},
        'limitations':['MARK has not reached its native draw return; no publisher HD integration',
                      'live pointer movements happen at different pause boundaries; mode tests share final RAM, not an identical input timestamp sequence',
                      'Windows and macOS not tested; packaging not started']}
(root/'verification-summary.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'result':'PASS','binary_sha256':binary,'summary_sha256':sha(root/'verification-summary.json')}))
