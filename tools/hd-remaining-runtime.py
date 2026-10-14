"""從已審查的美術與 READY 契約生成固定來源表；不改原版資料。"""
import hashlib
import json
import os
from pathlib import Path
import subprocess


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rect(value):
    assert len(value)==4 and all(isinstance(x,int) for x in value)
    return 'image.Rect('+', '.join(map(str,value))+')'


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000
    project=Path('/project')
    root=project/'workplace/hd-prototype/catalog/remaining'
    assert root.stat().st_uid==(project/'tools').stat().st_uid==1000
    ready_path=root/'runtime-ready-v1.json'
    ready=json.loads(ready_path.read_text())
    profiles_path=root/'profiles.json'
    profiles=json.loads(profiles_path.read_text())
    assert ready['status']=='READY' and ready['source_profiles_sha256']==sha(profiles_path)
    assert ready['static_foreground_only'] and ready['dynamic_VRAM_changes_still_require_original_index_equality']
    assert ready['source_names']==[r['name'] for r in profiles['records']] and len(profiles['records'])==20
    target=project/'tools/hd-graphics-remaining-profiles.go'
    assert not target.exists() and not (root/'runtime-profiles-v1.json').exists()
    lines=['package main','','import "image"','','var hdGraphicsRemainingProfiles = []hdGraphicsProfile{']
    records=[]
    for profile in profiles['records']:
        source=profile['source']
        data=(root.parent/'decoded'/(profile['name']+'.idx')).read_bytes()
        assert hashlib.sha256(data).hexdigest()==source['indices_sha256']
        assert sha(project/'assets/hd-b/art'/profile['asset'])==profile['asset_sha256']
        lines.append('{Source: graphicsSourceProfile{Name: '+json.dumps(profile['name'])+', Width: '+str(source['width'])+', Height: '+str(source['height'])+', Planes: []string{'+', '.join(json.dumps(x) for x in source['plane_sha256'])+'}},')
        lines.append('IndicesSHA: '+json.dumps(source['indices_sha256'])+', Asset: '+json.dumps(profile['asset'])+', AssetSHA: '+json.dumps(profile['asset_sha256'])+',')
        if profile.get('paint_rects'):
            lines.append('PaintRects: []image.Rectangle{'+', '.join(rect(x) for x in profile['paint_rects'])+'},')
        else:
            lines.append('PaintRect: '+rect(profile['paint_rect'])+',')
        regions=[]
        for region in profile.get('preserve_regions',[]):
            bits=65535 if region['indices']=='all' else sum(1<<i for i in set(region['indices']))
            assert 0<bits<=65535
            x,y,xx,yy=region['rect']
            count=sum(bool(bits&(1<<data[j*source['width']+i])) for j in range(y,yy) for i in range(x,xx))
            regions.append({**region,'index_bits':bits,'source_pixels':count})
        if regions:
            lines.append('Preserve: []hdGraphicsPreserve{'+', '.join('{Rect: '+rect(r['rect'])+', Indices: '+str(r['index_bits'])+'}' for r in regions)+'},')
        lines.append('},')
        records.append({'name':profile['name'],'source_indices_sha256':source['indices_sha256'],
                        'asset':profile['asset'],'asset_sha256':profile['asset_sha256'],
                        'paint_rect':None if profile.get('paint_rects') else profile['paint_rect'],
                        'paint_rects':profile.get('paint_rects',[]),'preserve_regions':regions})
    lines.append('}')
    target.write_text('\n'.join(lines)+'\n')
    subprocess.run(['gofmt','-w',str(target)],check=True,timeout=10)
    result={'classification':'registered source-specific opaque profiles; normal cases verified separately',
            'tool_sha256':sha(Path(__file__)),'ready_sha256':sha(ready_path),'art_profiles_sha256':sha(profiles_path),
            'go_profiles_sha256':sha(target),'registered_source_names':[r['name'] for r in records],'records':records}
    (root/'runtime-profiles-v1.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    (root/'runtime-profile-source.txt').write_bytes(Path(__file__).read_bytes())
    print(json.dumps({'registered_sources':20,'fixed_foreground_regions':sum(len(r['preserve_regions']) for r in records)}))


if __name__=='__main__':
    main()
