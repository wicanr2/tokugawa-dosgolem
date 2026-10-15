"""重核保存的正常／缺圖畫面；提前停止與完整 600M 收據分開記錄。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import runpy

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def captures(root, project):
    history = project/'workplace/hd-prototype/catalog/remaining/history'
    catalog = history.parents[1]
    graphics = {r['name']: r for r in read(history.parent/'profiles.json')['records']}
    stories = {r['resource']: r for r in read(catalog/'story-expanded/profiles.json')['records']}
    results = {}
    for mode, folder, count in [('partial', 'desktop', 14), ('partial-missing', 'desktop-missing', 17)]:
        directory = root/folder
        def local(name):
            path=Path(name)
            return directory/path.relative_to('/output') if path.is_absolute() else directory/path
        snapshot = directory/'verifier-source.txt'
        cases = runpy.run_path(str(snapshot))['stages']()
        completed = read(directory/'completed-stages.json')
        assert completed == [s['key'] for s in cases[:count]]
        state = read(directory/'desktop-status.json')
        events = state['events']
        frames = [e for e in events if e['kind'] == 'capture']
        toggles = [e for e in events if e['kind'] == 'toggle']
        assert len(frames) == count*3 and len(toggles) == 1+count*2
        assert all(e['guest_unchanged'] and e['before'] == e['after'] for e in toggles)
        checks = []
        for number, stage in enumerate(cases[:count]):
            source, src, dst = stage['source'], stage['src'], stage['dst']
            trio = frames[number*3:(number+1)*3]
            assert all(e['paused'] and e['proof'] == trio[0]['proof'] and e['step'] == trio[0]['step'] for e in trio)
            assert trio[0]['frame']['png_sha256'] == trio[2]['frame']['png_sha256']
            pictures = []
            for label, event in zip(['enabled', 'original', 'restored'], trio):
                actual = directory/f'gui-{mode}-{stage["key"]}-{label}.png'
                presented = local(event['frame']['path'])
                assert sha(presented) == event['frame']['png_sha256']
                assert Image.open(actual).convert('RGBA').tobytes() == Image.open(presented).convert('RGBA').tobytes()
                pictures.append({'actual_png':actual.name,'sha256':sha(actual),'event':event,'window_pixels_equal':True})
            native = trio[0]['frame']['native_indices']
            native_path = local(native['path'])
            assert sha(native_path) == native['sha256']
            indices = native_path.read_bytes()
            profile = stories[source] if stage.get('story') else graphics[source]
            width = 320 if stage.get('story') else profile['source']['width']
            source_path = catalog/'decoded'/(source+'.idx')
            original_source = source_path.read_bytes()
            expected_indices = b''.join(original_source[y*width+src[0]:y*width+src[2]] for y in range(src[1],src[3]))
            actual_indices = b''.join(indices[y*640+dst[0]:y*640+dst[2]] for y in range(dst[1],dst[3]))
            assert expected_indices == actual_indices
            before = Image.open(directory/pictures[1]['actual_png']).convert('RGB')
            after = Image.open(directory/pictures[0]['actual_png']).convert('RGB')
            painting = Image.open(project/'assets/hd-b/art'/profile['asset']).convert('RGB')
            physical = [v*2 for v in dst]
            if stage.get('story'):
                assert not profile['label_rects']
                selected = Image.new('RGB',(640,300));a,b=painting.load(),selected.load()
                for y in range(300):
                    for x in range(640):b[x,y]=a[x*painting.width//640,y*painting.height//300]
            else:
                selected = painting.crop([v*2 for v in src])
            missing = mode == 'partial-missing' and source in ['OP2_P.OZM','OP9.OZM','11.OZM']
            want = before.crop(physical) if missing else selected
            assert after.crop(physical).tobytes() == want.tobytes(),stage['key']
            changed = sum(a!=b for a,b in zip(before.crop(physical).getdata(),after.crop(physical).getdata()))
            assert (changed>0) != missing
            checks.append({'key':stage['key'],'source':source,'source_rect':src,'dest_rect':dst,
                           'mode':'fallback_missing_asset' if missing else 'hd','step':trio[0]['step'],
                           'changed_crop_pixels':changed,'pictures':pictures,'native_indices_sha256':native['sha256']})
        results[mode] = {'verifier_snapshot_sha256':sha(snapshot),'terminal_step':state['step'],
                         'complete_run':False,'capture_count':count,'toggle_count':len(toggles),'checks':checks}
    result = {'verified':True,'classification':'paired X11 and original-state samples; complete history state is a separate gate',
              'tool_sha256':sha(Path(__file__)),'binary_sha256':sha(root/'runtime/ready/tokugawa-desktop-prototype'),
              'source_proof_sha256':sha(root/'source-verification.json'),'runs':results,
              'normal_HD_sample_counts':{'duel':3,'story':5,'eyes':4,'portraits':2},
              'missing_sample_counts':{'duel':3,'story':5,'eyes':4,'portraits':5},
              'limitations':['three other normal HD portrait frames have source/shared-consumer tests; not claimed as normal HD X11 samples',
                             'original long sampling runs stopped before complete 600M; full/full-missing must verify complete state']}
    target = root/'capture-verification.json';assert not target.exists()
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'normal_samples':14,'missing_samples':17,'sha256':sha(target)}))


def summary(root, project):
    sampled_path=root/'capture-verification.json';sampled=read(sampled_path)
    assert sampled['verified'] and sampled['tool_sha256']==sha(Path(__file__))
    binary=sha(root/'runtime/ready/tokugawa-desktop-prototype');assert sampled['binary_sha256']==binary
    proofs={};sources=None
    for mode,folder,tool in [('full','full','verify-hd-opening-partial-full.py'),('full-missing','full-missing','verify-hd-opening-partial-full.py'),
                             ('title','title','verify-hd-remaining-runtime.py'),('weather','regression','verify-hd-weather.py')]:
        directory=root/folder;path=directory/f'verification-{mode}.json';proof=read(path)
        receipt_path=directory/f'desktop-{mode}.json';receipt=read(receipt_path)
        assert proof['verified'] and proof['receipt_sha256']==sha(receipt_path)
        assert proof['tool_sha256']==sha(project/'tools'/tool) and sha(directory/'tokugawa-desktop-prototype')==binary
        hashes=receipt['integration_input_sha256'];assert len(hashes)==84
        sources=hashes if sources is None else sources;assert hashes==sources
        for name,digest in hashes.items():assert digest==sha(project/name)==sha(directory/'source-snapshot'/name)
        proofs[mode]={'path':str(path.relative_to(root)),'sha256':sha(path),'receipt_sha256':sha(receipt_path),'complete_RAM_sha256':proof['complete_RAM_sha256']}
    log_path=root/'runtime/ready/startup-probe/platform-test.log';log=log_path.read_text()
    assert 'FAIL' not in log and sum(line.startswith('ok') for line in log.splitlines())==4
    previous=project/'workplace/hd-prototype/catalog/remaining/history/regression/palette'
    for picture in read(root/'regression/verification-weather.json')['pictures']:
        name=picture['actual_png'];assert Image.open(root/'regression'/name).convert('RGBA').tobytes()==Image.open(previous/name).convert('RGBA').tobytes()
    result={'verified':True,'classification':'all unshifted opening offsets integrated; normal HD sampled by source and draw class',
            'tool_sha256':sha(Path(__file__)),'binary_sha256':binary,'source_snapshot_count':84,'source_proof_sha256':sha(root/'source-verification.json'),
            'capture_proof_sha256':sha(sampled_path),'proofs':proofs,'normal_graphics_sources':['OP2_P.OZM','OP9.OZM'],
            'normal_story_sources':['6.OZM','11.OZM','19.OZM','28.OZM','43.OZM'],
            'implemented_frames':{'duel':3,'eyes':4,'portraits':5,'story':5},'normal_HD_sample_counts':sampled['normal_HD_sample_counts'],
            'missing_sample_counts':sampled['missing_sample_counts'],'platform_test_log_sha256':sha(log_path),
            'limitations':sampled['limitations'][:1]+['OP2/OP8 scrolling, other animation consumers, four partial OZM and PCX still incomplete',
                                                   'whole-game HD and platform packaging remain incomplete']}
    target=root/'verification-summary.json';assert not target.exists()
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'verified':True,'summary_sha256':sha(target),'binary_sha256':binary}))


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000
    parser=argparse.ArgumentParser();parser.add_argument('--mode',choices=['captures','summary'],required=True)
    args=parser.parse_args();root,project=Path('/output'),Path('/project')
    assert root.stat().st_uid==1000
    if args.mode=='captures':captures(root,project)
    else:summary(root,project)


if __name__=='__main__':main()
