"""核對實際 MARK／HD 桌面及遊戲回歸，不能據此宣稱啟動器已串接。"""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import struct
from PIL import Image

assert Path('/.dockerenv').exists() and os.getuid()==1000
p=argparse.ArgumentParser();p.add_argument('--source-root',type=Path,default=Path('/project'));args=p.parse_args()
root=Path('/output');project=Path('/project');assert root.stat().st_uid==1000
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def read(path):return json.loads(path.read_text())
sources=read(root/'build-sources.json');assert len(sources)==102
assert all(sha(args.source_root/name)==value for name,value in sources.items())
binary=sha(root/'tokugawa-desktop-prototype')
data=(project/'original/game/DX5.DAT').read_bytes()
assert hashlib.sha256(data).hexdigest()=='de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37'
pcx=None
for n in range(201):
    at=12+n*21
    if data[at:at+13].split(b'\0')[0]==b'RAINEND.PCX':
        length,offset=struct.unpack_from('<II',data,at+13);pcx=data[offset:offset+length]
assert pcx is not None and len(pcx)==97864
assert hashlib.sha256(pcx).hexdigest()=='78d0179bac1793eabb170925fd9b24378249c83c49320eaac27c8f70f65c1763'
native=Image.open(io.BytesIO(pcx));indices=native.tobytes()
assert native.mode=='P' and native.size==(640,480)
indexed_sha=hashlib.sha256(indices).hexdigest()
assert indexed_sha=='fb43b08b3202d43f0e502528683ea9ecc91a8b43778b680bd5c5c5a69228c5d4'
target=bytes(v>>2 for v in native.getpalette()[:768])
native.putpalette([v<<2|v>>4 for v in target])
original=native.convert('RGBA').resize((1280,960),Image.Resampling.NEAREST)
baseline_root=project/'workplace/hd-prototype/catalog/remaining/consumers/mark-probe/r8'
baseline_ram=sha(baseline_root/'startup-memory.bin')
reports=[]
def presented(case,capture):return Image.open(case/Path(capture['event']['frame']['path']).relative_to('/output')).convert('RGBA')
for mode in ['hd','missing']:
    case=root/mode;v=read(case/'verification.json');r=read(case/'receipt.json')
    assert v['result']=='PASS' and sha(case/'tokugawa-desktop-prototype')==binary==v['binary_sha256']
    assert r['integration_input_sha256']==sources
    assert r['successful_steps']==15989208 and r['stop']=='dos_exit' and r['exit_code']==r['fd2_startup_calls']==0
    assert sha(case/'mark-startup-memory.bin')==baseline_ram==v['ram_sha256']
    assert r['mark_observation']['target_palette_pointer']>0
    assert all(c['window_pixels_equal'] for c in v['captures'])
    originals=[presented(case,c) for c in v['captures']]
    assert originals[0].tobytes()==original.tobytes()
    if mode=='missing':assert all(im.tobytes()==original.tobytes() for im in originals)
    else:
        asset=project/'assets/hd-b/art/RAINEND-hd-b-v1.png'
        assert sha(asset)=='d81bfa59067217ef2150cdf325a3bbde93385f6486230da9dda1233f79686d93'
        im=Image.open(asset).convert('RGBA');raw=im.tobytes();width,height=im.size
        expected=bytearray(original.tobytes())
        for y in range(898):
            sy=y*height//960
            expected[y*1280*4:(y+1)*1280*4]=b''.join(raw[(sy*width+x*width//1280)*4:(sy*width+x*width//1280)*4+4] for x in range(1280))
        assert originals[1].tobytes()==bytes(expected)
        assert originals[2].tobytes()==original.tobytes()
    reports.append({'mode':mode,'receipt_sha256':sha(case/'receipt.json'),'verification_sha256':sha(case/'verification.json'),
                    'ram_sha256':v['ram_sha256'],'actual_window_captures':len(v['captures'])})
case=root/'regression';r=read(case/'desktop-scene.json');v=read(case/'verification-scene.json')
b=read(project/'workplace/desktop-prototype/headless/startup-probe/latest.json')
fields=['successful_steps','instruction_budget','stop','stop_instruction_linear_eip','trace_tail','interrupt_counts','dpmi_calls','dpmi_unimplemented',
        'frame','cd_pcm','mounted_disc','mouse_input_events','input_boundary_frames','real_mode_history','real_mode_last']
differences=[name for name in fields if r[name]!=b[name]];assert not differences,differences
assert r['integration_input_sha256']==sources and v['verified'] and len(v['pictures'])==3
assert sha(case/'startup-memory.bin')=='4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9'
assert all(p['window_pixels_equal'] for p in v['pictures'])
report={'result':'PASS','scope':'standalone original MARK desktop and HD; game launcher integration remains incomplete',
        'binary_sha256':binary,'source_count':len(sources),'sources_sha256':sha(root/'build-sources.json'),'mark':reports,
        'independent_source':{'tool':'Pillow 9.4.0','payload_sha256':hashlib.sha256(pcx).hexdigest(),'indexed_sha256':indexed_sha,
                              'target_palette_sha256':hashlib.sha256(target).hexdigest(),'original_copyright_rows':[449,480],
                              'HD_formula':'floor(x*1448/1280),floor(y*1086/960); original rows 449..479 preserved'},
        'game_regression':{'steps':r['successful_steps'],'mouse_events':len(r['mouse_input_events']),'fields_equal':fields,
                           'receipt_sha256':sha(case/'desktop-scene.json'),'actual_window_captures':3},
        'limitations':['full desktop startup has not yet chained MARK to DX5','fade palette effects are a presentation approximation; unknown palettes fall back',
                       'remaining animation consumers and three-platform packages not complete']}
(root/'verification-summary.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'result':'PASS','summary_sha256':sha(root/'verification-summary.json')}))
