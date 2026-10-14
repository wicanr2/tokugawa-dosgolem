"""核對來源與已審查美術；不把靜態素材準備當作執行器接入。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageChops


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def original_crop(data,width,rect):
    return b''.join(data[y*width+rect[0]:y*width+rect[2]] for y in range(rect[1],rect[3]))


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000
    parser=argparse.ArgumentParser()
    parser.add_argument('--version',type=int,default=3)
    args=parser.parse_args()
    project=Path('/project')
    catalog=project/'workplace/hd-prototype/catalog'
    root=catalog/'remaining'/f'draft-v{args.version}'
    manifest_path=root/'draft-manifest.json'
    manifest=json.loads(manifest_path.read_text())
    assert manifest['tool_sha256']==sha(root/'prepare-source.txt')
    assert manifest['generation_metadata_sha256']==sha(catalog/'art/remaining-full-generation-v1.json')
    assert manifest['source_intake_sha256']==sha(root.parent/'sources.json')
    review_path=root/'review.json'
    review=json.loads(review_path.read_text())
    assert review['approved_sources']==[r['name'] for r in manifest['records']]
    assert len(manifest['records'])==20
    images={}
    frames={}
    mirrored=empty=0
    for record in manifest['records']:
        name=record['name']
        path=root/record['prepared']
        assert path.stat().st_uid==path.stat().st_gid==1000
        assert sha(path)==record['asset_sha256']
        im=Image.open(path).convert('RGB')
        width,height=record['source']['width'],record['source']['height']
        assert im.size==(width*2,height*2)
        assert Image.open(path).convert('RGBA').getextrema()[3]==(255,255)
        data=(catalog/'decoded'/(name+'.idx')).read_bytes()
        assert hashlib.sha256(data).hexdigest()==record['source']['indices_sha256']
        images[name]=im
        for source in record['inputs']:
            assert sha(catalog/'art'/source['asset'])==source['sha256']
            assert sha(catalog/'art'/source['prompt'])==source['prompt_sha256']
        if name.startswith('ANIME'):
            assert len(record['frames'])==6 and im.size==(1248,600)
            frames[name]=[]
            for frame in record['frames']:
                i=frame['index']
                rect=[i%3*208,i//3*150,(i%3+1)*208,(i//3+1)*150]
                assert frame['source_rect']==rect
                native=original_crop(data,width,rect)
                assert hashlib.sha256(native).hexdigest()==frame['indices_sha256']
                art_path=root/frame['prepared']
                assert sha(art_path)==frame['asset_sha256']
                art=Image.open(art_path).convert('RGB')
                assert art.size==(416,300)
                assert art.tobytes()==im.crop(tuple(v*2 for v in rect)).tobytes()
                frames[name].append(art)
                if 'mirror' in frame['derivation']:
                    owner=frame['art_owner']
                    owner_native=original_crop((catalog/'decoded'/(owner+'.idx')).read_bytes(),624,rect)
                    flipped=b''.join(owner_native[y:y+208][::-1] for y in range(0,len(owner_native),208))
                    assert native==flipped
                    assert art.tobytes()==frames[owner][i].transpose(Image.Transpose.FLIP_LEFT_RIGHT).tobytes()
                    mirrored+=1
                if not any(native):
                    assert name=='ANIME04.OZM' and i==5
                    assert art.getextrema()==((0,0),(0,0),(0,0))
                    empty+=1
        if name=='END-11B.OZM':
            delta=ImageChops.difference(im,images['END-11.OZM']).getbbox()
            rect=record['reuse']['replacement_crest']
            assert delta and delta[0]>=rect[0]*2 and delta[1]>=rect[1]*2 and delta[2]<=rect[2]*2 and delta[3]<=rect[3]*2
            other=(catalog/'decoded/END-11.OZM.idx').read_bytes()
            for i,(a,b) in enumerate(zip(data,other)):
                if a!=b:assert rect[0]<=i%width<rect[2] and rect[1]<=i//width<rect[3]
        if name=='END-2.OZM':
            assert im.crop((0,300,1280,800)).getextrema()==((0,0),(0,0),(0,0))
        if name=='END-2B.OZM':
            assert len(record['frames'])==8
            for rect in record['reserved_buffer_rects']:
                assert im.crop(tuple(v*2 for v in rect)).getextrema()==((215,215),(0,0),(0,0))
        if name=='OP9.OZM':
            assert len(record['eye_strip_source_rects'])==4
            for rect in record['reserved_buffer_rects']:
                assert im.crop(tuple(v*2 for v in rect)).getextrema()==((20,20),(20,20),(227,227))
        if name.startswith('JIMEN_'):
            assert data.count(0)==record['original_zero_padding_pixels']
            for i,value in enumerate(data):
                if value==0:
                    x,y=i%width*2,i//width*2
                    assert im.crop((x,y,x+2,y+2)).getextrema()==((0,0),(0,0),(0,0))
            grass=im.getpixel((width,height//3))
            earth=im.getpixel((width,height*3//2))
            assert grass[1]>grass[0] and earth[0]>earth[1]
        if name=='BATTLE.OZM':
            edge=record['paint_rect'][3]
            assert edge==247 and not any(data[edge*640:])
            assert im.crop((0,edge*2,1280,800)).getextrema()==((0,0),(0,0),(0,0))
    assert mirrored==16 and empty==1 and len(frames)==8
    for gallery in manifest['galleries']:
        assert sha(root/gallery['asset'])==gallery['sha256']
    result={'verified':True,'classification':'reviewed calibrated art only; new runtime sources not yet registered',
            'tool_sha256':sha(Path(__file__)),'manifest_sha256':sha(manifest_path),'review_sha256':sha(review_path),
            'source_count':20,'animation_atlases':8,'animation_frames':48,'exact_mirrored_frames':mirrored,
            'original_empty_frames':empty,'additional_sky_frames':8,'new_runtime_sources_registered':0,
            'limitations':['runtime foreground preservation and source registration pending',
                           'normal title, animation, battle-dialog and remaining UI paths not verified by this art-only receipt',
                           'five unsupported OZM, PCX and whole-game HD completion remain open']}
    path=root/'art-verification.json'
    assert not path.exists()
    path.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({k:result[k] for k in ['verified','source_count','animation_frames','exact_mirrored_frames','new_runtime_sources_registered']}))


if __name__=='__main__':
    main()
