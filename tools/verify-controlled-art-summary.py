"""核對剩餘美術的受控原版、X11、HD 像素及缺圖；不驗收通關。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image

assert Path('/.dockerenv').is_file() and os.getuid() == 1000
root, project = Path('/output'), Path('/project')
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
digest = lambda b: hashlib.sha256(b).hexdigest()
rect = lambda r: (r['Min']['X'],r['Min']['Y'],r['Max']['X'],r['Max']['Y'])
manifest = read(root/'validation-manifest.json')
assert manifest['validation_only'] and not manifest['normal_playthrough_verified']
assert manifest['prepare_tool_sha256'] == sha(root/'prepare-art-validation.py') == sha(project/'tools/prepare-art-validation.py')
assert len(manifest['original_compiled_sources']) == len(manifest['controlled_compiled_sources']) == 106
assert [p for p,v in manifest['original_compiled_sources'].items()
        if manifest['controlled_compiled_sources'][p] != v] == ['tools/startup-probe.go']
assert all(sha(project/p)==v for p,v in manifest['original_compiled_sources'].items())
assert all(sha(root/'source-snapshot'/p)==v for p,v in manifest['controlled_compiled_sources'].items())
native = read(root/'native/receipt.json')
assert native['stop'] == 'controlled_art_complete' and native['native_runtime_instructions_skipped']
assert native['integration_input_sha256'] == manifest['controlled_compiled_sources']
stages, entries = native['art_validation_stages'], native['art_validation_entries']
assert len(stages)==55 and len(entries)==9
baseline=native['art_validation_baseline']
assert baseline['step']==80000000 and baseline['after_eip']==0x10b51 and baseline['ram_bytes_changed']==0
assert baseline['ram_sha256_before']==manifest['control_ram_sha256']
assert stages[0]['label']=='control-title-repaint' and stages[0]['case']==-1
assert entries[0]['ram_sha256_before'] == stages[0]['ram_sha256']
for i,entry in enumerate(entries):
    assert entry['case']==i
    if i<8:
        side,index,flags,resource=manifest['animation_plans'][i]
        assert entry['resource_id']==resource and entry['after_eip']==(0x2388a if side else 0x2387e)
        assert [(w['linear'],w['after'],w['bytes']) for w in entry['writes']]==[(0x3777a,flags,4),(0x376b9,index,4),(0x37732,index,4)]
    else:
        assert entry['writes']==[] and entry['before_eip']==0x23894 and entry['after_eip']==0x10b51
        change=entry['loader_override']
        assert change['eip']==0x1bf73 and change['before']==105 and change['after']==121 and change['bytes']==4
    if i:
        prior=[s for s in stages if s['case']==i-1][-1]
        assert entry['step']==prior['step'] and entry['ram_sha256_before']==prior['ram_sha256']
profiles={r['name']:r for r in read(project/'assets/hd-b/art/remaining/profiles.json')['records']}
source, art = {}, {}
for name in ['OP5.OZM','OP6.OZM']+[f'ANIME{i:02d}.OZM' for i in range(1,9)]:
    p=profiles[name]
    source[name]=(project/'workplace/hd-prototype/catalog/decoded'/(name+'.idx')).read_bytes()
    assert digest(source[name])==p['source']['indices_sha256']
    path=project/'assets/hd-b/art'/p['asset']
    assert sha(path)==p['asset_sha256']
    art[name]=Image.open(path).convert('RGBA')
    assert art[name].size==(p['source']['width']*2,p['source']['height']*2)
control=(project/manifest['control_relative']/'receipt.json')
original_title=profiles['OP5.OZM']['source']['indices_sha256']
assert stages[0]['indices_sha256']==original_title
for stage in stages:
    idx=(root/'native'/stage['indices_path']).read_bytes()
    assert len(idx)==640*400 and digest(idx)==stage['indices_sha256']
    label=stage['label']
    if label.endswith('-restored'):
        assert digest(idx)==original_title
        continue
    draw=stage['last_graphics_return']
    assert draw['verified'] and draw['ExpectedSHA']==draw['ActualSHA']
    name=draw['resource']; sx,sy,ex,ey=rect(draw['SourceRect']); dx,dy,xx,yy=rect(draw['DestRect'])
    width=profiles[name]['source']['width']
    for y in range(ey-sy):
        assert idx[(dy+y)*640+dx:(dy+y)*640+xx]==source[name][(sy+y)*width+sx:(sy+y)*width+ex]
    if name.startswith('ANIME'):
        assert (dx,dy,xx,yy)==(216,100,424,250) and draw['Return']==0x2350c
for i,count in enumerate([6,6,6,5,5,5,6,6],1):
    frames=[s for s in stages if s['label'].startswith(f'ANIME{i:02d}.OZM-frame-')]
    assert len(frames)==count
    assert [rect(s['last_graphics_return']['SourceRect']) for s in frames]==[(x,y,x+208,y+150) for x,y in [(0,0),(208,0),(416,0),(0,150),(208,150),(416,150)][:count]]
assert all(source['ANIME04.OZM'][y*624+416:y*624+624]==bytes(208) for y in range(150,300))


def expected_hd(base,state,indices,mode,proof,stage):
    expected=base.copy(); painted={}; preserved={}
    reports=state['graphics'] or []; draws=state['graphics_observation']['verified_draws'] or []
    assert len(reports)==len(draws)
    assert state['portrait_mode']!='hd' and not any(v=='hd' for v in (state['window_modes'] or []))
    for report,draw in zip(reports,draws):
        name=draw['resource']; assert report['resource']==name
        missing=mode=='missing' and name in ['ANIME03.OZM','OP6.OZM']
        if missing:
            assert report['mode']=='fallback_missing_asset'
        if report['mode']!='hd':
            continue
        assert name in profiles and name in art and not missing
        p=profiles[name]; width=p['source']['width']; sx,sy,ex,ey=rect(draw['SourceRect']); dx,dy,xx,yy=rect(draw['DestRect'])
        native_count,keep=0,0
        for bounds in report['visible_rectangles']:
            x1,y1,x2,y2=rect(bounds)
            px1,py1=sx+x1-dx,sy+y1-dy
            expected.paste(art[name].crop((px1*2,py1*2,(px1+x2-x1)*2,(py1+y2-y1)*2)),(x1*2,y1*2))
            native_count+=(x2-x1)*(y2-y1)
            for y in range(y1,y2):
                py=sy+y-dy
                assert indices[y*640+x1:y*640+x2]==source[name][py*width+px1:py*width+px1+x2-x1]
            for region in p.get('preserve_regions',[]):
                rx1,ry1,rx2,ry2=region['rect']
                for py in range(max(py1,ry1),min(py1+y2-y1,ry2)):
                    for px in range(max(px1,rx1),min(px1+x2-x1,rx2)):
                        if source[name][py*width+px] in region['indices']:
                            x,y=dx+px-sx,dy+py-sy
                            expected.paste(base.crop((x*2,y*2,x*2+2,y*2+2)),(x*2,y*2))
                            keep+=1; native_count-=1
        assert native_count==report['painted_native_pixels'] and keep==report.get('preserved_native_pixels',0)
        painted[name]=painted.get(name,0)+native_count*4
        preserved[name]=preserved.get(name,0)+keep*4
    cursor=proof['cursor']
    assert proof['palette']==stage['palette']
    cursor_pixels=0
    if cursor['Defined'] and cursor['DisplayCount']>=0:
        for row in range(16):
            for col in range(16):
                x=proof['mouse_x']-cursor['HotX']+col; y=proof['mouse_y']-cursor['HotY']+row
                if not (0<=x<640 and 0<=y<400): continue
                bit=0x8000>>col
                if cursor['Mask'][row]&bit and not cursor['Mask'][row+16]&bit: continue
                index=indices[y*640+x] if cursor['Mask'][row]&bit else 0
                if cursor['Mask'][row+16]&bit: index^=15
                color=tuple(stage['palette'][stage['DAC'][index]])+(255,)
                assert base.getpixel((x*2,y*2))==color
                for dy in range(2):
                    for dx in range(2): expected.putpixel((x*2+dx,y*2+dy),color)
                cursor_pixels+=4
    preserved['cursor_overlay']=cursor_pixels
    return expected,painted,preserved


cases={}
for mode in ['hd','missing']:
    case=root/('desktop-'+mode); captures=read(case/'controlled-ending-captures.json'); receipt=read(case/'receipt.json')
    assert captures['status']=='PASS_CAPTURE' and captures['validation_only'] and not captures['normal_playthrough_verified']
    assert captures['tool_sha256']==sha(project/'tools/verify-controlled-art.py')==sha(case/'source-snapshot/tools/verify-controlled-art.py')
    assert captures['receipt_sha256']==sha(case/'receipt.json') and captures['binary_sha256']==sha(root/'tokugawa-art-desktop')==sha(case/'tokugawa-art-desktop')
    assert receipt['integration_input_sha256']==manifest['controlled_compiled_sources']
    assert receipt['art_validation_entries']==entries and receipt['art_validation_stages']==stages and captures['stages']==stages
    assert receipt['art_validation_baseline']==baseline and receipt['mouse_input_events']==native['mouse_input_events']==[]
    assert receipt['stop']==native['stop'] and receipt['successful_steps']==native['successful_steps']
    assert sha(case/'startup-memory.bin')==sha(root/'native/startup-memory.bin')
    for field in ['trace_tail','interrupt_counts','dpmi_calls','dpmi_unimplemented','cd_pcm','graphics_draw_trace','frame']:
        assert receipt[field]==native[field],field
    groups={}
    for capture in captures['captures']:
        groups.setdefault(capture['stage_id'],{})[capture['mode']]=capture
    checks=[]
    for stage_id,group in groups.items():
        assert set(group)=={'original','hd','restored'}
        stage=stages[stage_id-1]; images={}; states={}
        for kind,capture in group.items():
            path=case/Path(capture['actual']).relative_to('/output'); assert sha(path)==capture['actual_sha256'] and capture['window_pixels_equal']
            images[kind]=Image.open(path).convert('RGBA'); assert images[kind].size==(1280,800)
            own=case/Path(capture['event']['frame']['path']).relative_to('/output')
            assert sha(own)==capture['event']['frame']['png_sha256'] and images[kind].tobytes()==Image.open(own).convert('RGBA').tobytes()
            proof=capture['event']['proof']
            for key in ['step','eip','registers','segments','flags','ram_sha256']:
                assert proof[key]==stage[key]
            assert proof['raw_indexed_sha256']==stage['indices_sha256']
            states[kind]=read(case/capture['state_path'])
        assert group['original']['event']['proof']==group['hd']['event']['proof']==group['restored']['event']['proof']
        assert images['original'].tobytes()==images['restored'].tobytes()
        native_frame=root/'native'/Path(stage['frame']['path']).relative_to('workplace')
        assert images['original'].tobytes()==Image.open(native_frame).convert('RGBA').resize((1280,800),Image.Resampling.NEAREST).tobytes()
        indices=(case/stage['indices_path']).read_bytes(); assert digest(indices)==stage['indices_sha256']
        expected,painted,preserved=expected_hd(images['original'],states['hd'],indices,mode,group['original']['event']['proof'],stage)
        assert expected.tobytes()==images['hd'].tobytes(),(mode,stage['label'],painted)
        if '-frame-' in stage['label']:
            name=stage['label'].split('-frame-')[0]
            if mode=='hd' or name!='ANIME03.OZM': assert painted.get(name,0)==208*150*4
        if stage['label']=='OP6.OZM-full' and mode=='hd': assert painted.get('OP6.OZM',0)>0 and preserved['OP6.OZM']>0
        checks.append(dict(stage_id=stage_id,label=stage['label'],painted_output_pixels=painted,preserved_output_pixels=preserved))
    toggles=[e for e in read(case/'desktop-status.json')['events'] if e['kind']=='toggle']
    assert len(groups)==18 and len(toggles)==36 and all(e['guest_unchanged'] for e in toggles)
    restored=[images for images in checks if images['label'].endswith('-restored')]; assert len(restored)==8
    cases[mode]=dict(captures=len(captures['captures']),toggles=len(toggles),checks=checks,receipt_sha256=sha(case/'receipt.json'))
summary=dict(status='PASS',validation_only=True,normal_playthrough_required=False,normal_playthrough_verified=False,
             controlled_sources=[f'ANIME{i:02d}.OZM' for i in range(1,9)]+['OP6.OZM'],new_controlled_sources=[f'ANIME{i:02d}.OZM' for i in range(1,8)]+['OP6.OZM'],
             same_state_checkpoints=55,original_animation_frames_verified=45,original_background_restorations_verified=8,
             complete_ram_equal_across_cases=True,successful_steps=native['successful_steps'],cases=cases,
             manifest_sha256=sha(root/'validation-manifest.json'),native_receipt_sha256=sha(root/'native/receipt.json'),
             tool_sha256=sha(Path(__file__)),GUI_tool_sha256=sha(project/'tools/verify-controlled-art.py'),prepare_tool_sha256=sha(project/'tools/prepare-art-validation.py'))
(root/'verification-summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({k:v for k,v in summary.items() if k!='cases'},ensure_ascii=False))
