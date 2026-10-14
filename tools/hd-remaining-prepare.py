"""剩餘 OZM 的來源格線及純生成素材準備；不複製原版字模或 UI 文字。"""
import argparse
from collections import deque
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageChops, ImageDraw, __version__


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def digest(data):
    return hashlib.sha256(data).hexdigest()


def source_crop(data, width, rect):
    x,y,xx,yy = rect
    return b''.join(data[j*width+x:j*width+xx] for j in range(y,yy))


def mirrored(data, width):
    return b''.join(data[y:y+width][::-1] for y in range(0,len(data),width))


def black_components(data, width, height, seeds):
    mask = bytearray(len(data))
    queue = deque()
    for x,y in seeds:
        i=y*width+x
        if data[i]==0 and not mask[i]:
            mask[i]=255
            queue.append(i)
    while queue:
        i=queue.popleft()
        x,y=i%width,i//width
        for dx,dy in [(0,-1),(0,1),(-1,0),(1,0)]:
            xx,yy=x+dx,y+dy
            if 0<=xx<width and 0<=yy<height:
                j=yy*width+xx
                if data[j]==0 and not mask[j]:
                    mask[j]=255
                    queue.append(j)
    return bytes(mask)


def frame_grid(im):
    # 格線由畫布邊界及實際紅金邊框定位；不以生成圖推定原版格線。
    width,height=im.size
    rgb=im.load()
    red=lambda c:c[0]>140 and c[0]>c[1]*1.3 and c[0]>c[2]*1.8
    counts=[sum(red(rgb[x,y]) for x in range(width)) for y in range(height)]
    occupied=[y for y,c in enumerate(counts) if c>width//4]
    assert occupied
    bottom=max(occupied)+1
    middle=max(range(max(1,bottom//2-24),min(height,bottom//2+25)),key=lambda y:counts[y])
    xs=[0]
    for fraction in [1/3,2/3]:
        center=round(width*fraction)
        x=max(range(center-14,center+15),key=lambda x:sum(red(rgb[x,y]) for y in range(bottom)))
        xs.append(x)
    xs.append(width)
    return xs,[0,middle,bottom]


def main():
    assert Path('/.dockerenv').is_file() and os.getuid()==1000 and __version__=='9.4.0'
    parser=argparse.ArgumentParser()
    parser.add_argument('--version',type=int,default=1)
    parser.add_argument('--approve',action='store_true',help='選定已審查且核對通過的美術，不修改執行器')
    args=parser.parse_args()
    assert 1<=args.version<=9
    project=Path('/project')
    catalog=project/'workplace/hd-prototype/catalog'
    root=catalog/'remaining'
    output=root/f'draft-v{args.version}'
    if args.approve:
        manifest_path=output/'draft-manifest.json'
        manifest=json.loads(manifest_path.read_text())
        proof_path=output/'art-verification.json'
        proof=json.loads(proof_path.read_text())
        review_path=output/'review.json'
        review=json.loads(review_path.read_text())
        names=[r['name'] for r in manifest['records']]
        assert manifest['tool_sha256']==sha(output/'prepare-source.txt')==sha(Path(__file__))
        assert proof['verified'] and proof['manifest_sha256']==sha(manifest_path)
        assert proof['review_sha256']==sha(review_path)
        assert review['approved_sources']==names and len(names)==20
        assert not (root/'profiles.json').exists() and not (root/'prepared').exists()
        (root/'prepared').mkdir()
        for record in manifest['records']:
            source=output/record['prepared']
            assert sha(source)==record['asset_sha256']
            shutil.copy2(source,root/record['prepared'])
        selected={**manifest,'classification':'reviewed calibrated HD art; runtime consumers not yet registered',
                  'selected_draft':output.name,'review_sha256':sha(review_path),'art_verification_sha256':sha(proof_path)}
        (root/'profiles.json').write_text(json.dumps(selected,ensure_ascii=False,indent=2)+'\n')
        print(json.dumps({'approved_art_sources':20,'new_runtime_sources_registered':0}))
        return
    assert root.stat().st_uid==1000 and not output.exists()
    output.mkdir()
    (output/'prepared').mkdir()
    (output/'frames').mkdir()
    source_manifest=root/'sources.json'
    sources=json.loads(source_manifest.read_text())
    assert sources['tool_sha256']==sha(root/'intake-source.txt')
    originals={r['source']['name']:r for r in sources['records']}
    gen_path=catalog/'art/remaining-full-generation-v1.json'
    generation={r['source']:r for r in json.loads(gen_path.read_text())['records']}
    assert len(originals)==20 and len(generation)==18
    source_data={name:(catalog/'decoded'/(name+'.idx')).read_bytes() for name in originals}
    for name,record in originals.items():
        assert digest(source_data[name])==record['source']['indices_sha256']
    raw={}
    for name,record in generation.items():
        path=catalog/'art'/record['asset']
        assert sha(path)==record['sha256'] and sha(catalog/'art'/record['prompt'])==record['prompt_sha256']
        raw[name]=Image.open(path).convert('RGB')
    records=[]
    prepared={}
    animation_frames={}
    mirror_count=0
    for name,original in originals.items():
        source=original['source']
        width,height=source['width'],source['height']
        data=source_data[name]
        record={'name':name,'source':source,'asset':f'remaining/prepared/{name}-hd-v{args.version}.png',
                'prepared':f'prepared/{name}-hd-v{args.version}.png','paint_rect':[0,0,width,height],
                'runtime_status':'not yet registered; geometry review and rendering contract required'}
        inputs=[generation[name]] if name in generation else []
        if name.startswith('ANIME'):
            image=Image.new('RGB',(width*2,height*2),'black')
            animation_frames[name]=[]
            owner={'ANIME02.OZM':'ANIME01.OZM','ANIME08.OZM':'ANIME07.OZM'}.get(name)
            grid=frame_grid(raw[name]) if not owner else None
            record['raw_grid']=grid
            record['frames']=[]
            for index in range(6):
                x,y=index%3*208,index//3*150
                rect=[x,y,x+208,y+150]
                native=source_crop(data,width,rect)
                frame_owner=owner or ('ANIME05.OZM' if name=='ANIME06.OZM' and index<4 else None)
                info={'index':index,'source_rect':rect,'indices_sha256':digest(native)}
                if frame_owner:
                    other=source_crop(source_data[frame_owner],624,rect)
                    assert native==mirrored(other,208),(name,index)
                    frame=animation_frames[frame_owner][index].transpose(Image.Transpose.FLIP_LEFT_RIGHT)
                    info.update(art_owner=frame_owner,derivation='proven exact horizontal source mirror')
                    mirror_count+=1
                    if generation[frame_owner] not in inputs:inputs.append(generation[frame_owner])
                elif not any(native):
                    frame=Image.new('RGB',(416,300),'black')
                    info['derivation']='original empty frame; no generated image painted'
                else:
                    xs,ys=grid
                    crop=[xs[index%3],ys[index//3],xs[index%3+1],ys[index//3+1]]
                    frame=raw[name].crop(crop).resize((416,300),Image.Resampling.LANCZOS)
                    info.update(raw_crop=crop,art_owner=name,derivation='own frame from reviewed generated atlas')
                animation_frames[name].append(frame)
                target=output/'frames'/f'{name}-{index}-hd-v{args.version}.png'
                frame.save(target)
                info.update(prepared=str(target.relative_to(output)),asset_sha256=sha(target))
                record['frames'].append(info)
                image.paste(frame,(x*2,y*2))
        elif name=='OP2_P.OZM':
            image=Image.new('RGB',(width*2,height*2))
            im=raw[name]
            record['frames']=[]
            for i in range(3):
                crop=[round(im.width*i/3),0,round(im.width*(i+1)/3),im.height]
                frame=im.crop(crop).resize((256,238),Image.Resampling.LANCZOS)
                image.paste(frame,(256*i,0))
                record['frames'].append({'index':i,'source_rect':[128*i,0,128*(i+1),119],
                                         'indices_sha256':digest(source_crop(data,width,[128*i,0,128*(i+1),119])),
                                         'raw_crop':crop})
        elif name=='END-2.OZM':
            image=Image.new('RGB',(width*2,height*2),'black')
            im=raw[name]
            image.paste(im.crop((0,0,im.width,round(im.height*.375))).resize((1280,300),Image.Resampling.LANCZOS),(0,0))
            record['paint_rect']=[0,0,640,150]
        elif name=='END-2B.OZM':
            image=Image.new('RGB',(1280,800),(215,0,0))
            im=raw[name]
            record['frames']=[]
            for i in range(8):
                crop=[round(im.width*.4*(i%2)),round(im.height*.21*(i//2)),round(im.width*.4*(i%2+1)),round(im.height*.21*(i//2+1))]
                image.paste(im.crop(crop).resize((512,168),Image.Resampling.LANCZOS),(512*(i%2),168*(i//2)))
                rect=[256*(i%2),84*(i//2),256*(i%2+1),84*(i//2+1)]
                record['frames'].append({'index':i,'source_rect':rect,'indices_sha256':digest(source_crop(data,width,rect)),'raw_crop':crop})
            record['paint_rect']=[0,0,512,336]
            record['reserved_buffer_rects']=[[512,0,640,400],[0,336,512,400]]
        elif name.startswith('JIMEN_'):
            im=raw[name]
            grass=im.crop((round(im.width*.22),round(im.height*.08),round(im.width*.66),round(im.height*.32))).resize((width*2,height*2),Image.Resampling.LANCZOS)
            earth=im.crop((round(im.width*.1),round(im.height*.54),round(im.width*.8),round(im.height*.78))).resize((width*2,height*2),Image.Resampling.LANCZOS)
            image=Image.new('RGB',(width*2,height*2),'black')
            for y in range(height):
                for x in range(width):
                    if data[y*width+x]==0:continue
                    is_grass=y<64 and (x<351-y if name=='JIMEN_0.OZM' else x>=62-y)
                    material=grass if is_grass else earth
                    image.paste(material.crop((x*2,y*2,x*2+2,y*2+2)),(x*2,y*2))
            record['material_geometry']='source-zero reserved background; grass original diagonal and horizontal y=64 split; generated grass/earth material only'
            record['original_zero_padding_pixels']=data.count(0)
        elif name=='MAP_WAKU.OZM':
            im=raw[name]
            image=Image.new('RGB',(width*2,height*2),'black')
            src_ys=[0,round(im.height*.16),round(im.height*.44),round(im.height*.72),im.height]
            dst_ys=[0,30,110,176,240]
            for a,b,c,d in zip(src_ys,src_ys[1:],dst_ys,dst_ys[1:]):
                image.paste(im.crop((0,a,im.width,b)).resize((width*2,d-c),Image.Resampling.LANCZOS),(0,c))
            record['vertical_anchors']={'raw':src_ys,'prepared':dst_ys}
            record['preserve_regions']=[{'rect':[134,92,192,104],'indices':'all','reason':'original buffer/debug glyphs; not new HD lettering'}]
        else:
            image=raw[name].resize((width*2,height*2),Image.Resampling.LANCZOS)
        if name=='BATTLE.OZM':
            seeds=[(x,height-1) for x in range(width)]+[(80,100),(560,100)]
            mask=black_components(data,width,height,seeds)
            image.paste((0,0,0),(0,0),Image.frombytes('L',(width,height),mask).resize(image.size,Image.Resampling.NEAREST))
            last_row=max(i//width for i,value in enumerate(data) if value)+1
            record.update(paint_rect=[0,0,640,last_row],black_reserve_pixels=mask.count(255),black_reserve_sha256=digest(mask))
        if name=='OP5.OZM':
            source_mask=Image.frombytes('L',(width,height),bytes(255 if v not in [0,8] else 0 for v in data))
            rect=source_mask.getbbox()
            grayscale=image.convert('L').point(lambda v:255 if v>24 else 0)
            crop=grayscale.getbbox()
            target=Image.new('RGB',image.size,'black')
            target.paste(image.crop(crop).resize(((rect[2]-rect[0])*2,(rect[3]-rect[1])*2),Image.Resampling.LANCZOS),(rect[0]*2,rect[1]*2))
            image=target
            record.update(paint_rect=list(rect),calibration={'prepared_content_crop':list(crop),'native_art_bounds':list(rect)})
        if name=='OP9.OZM':
            reserved=[[624,0,640,200],[488,200,640,308],[416,308,640,400]]
            for rect in reserved:ImageDraw.Draw(image).rectangle((rect[0]*2,rect[1]*2,rect[2]*2-1,rect[3]*2-1),fill=(20,20,227))
            record.update(reserved_buffer_rects=reserved,
                          paint_rects=[[0,0,624,200],[0,200,416,400],[416,200,488,308]],
                          eye_strip_source_rects=[[416,200+i*27,488,227+i*27] for i in range(4)])
        if name=='OP6.OZM':
            record['preserve_regions']=[{'rect':[448,188,624,240],'indices':[15],'reason':'original chapter caption'},
                                        {'rect':[464,258,616,366],'indices':[0,1,8],'reason':'original year and location'},
                                        {'rect':[60,350,192,374],'indices':[0,1,8],'reason':'original coordinate text'},
                                        {'rect':[274,350,410,374],'indices':[0,1,8],'reason':'original bottom status'}]
        if name=='END-11B.OZM':
            # 原始兩圖只有完字不同；人物的生成素材強制重用，避免局部編輯漂移。
            other=source_data['END-11.OZM']
            changed=[i for i,(a,b) in enumerate(zip(data,other)) if a!=b]
            box=[min(i%width for i in changed),min(i//width for i in changed),max(i%width for i in changed)+1,max(i//width for i in changed)+1]
            crest=[55,20,295,256]
            assert box[0]>=crest[0] and box[1]>=crest[1] and box[2]<=crest[2] and box[3]<=crest[3],box
            target=prepared['END-11.OZM'].copy()
            target.paste(image.crop(tuple(v*2 for v in crest)),(crest[0]*2,crest[1]*2))
            image=target
            record['reuse']={'unchanged_art':'END-11.OZM','native_difference_bounds':box,'replacement_crest':crest,'basis':'original indices identical outside the complete ending Kanji region'}
            inputs.append(generation['END-11.OZM'])
        if name=='MAP_WAKU.OZM':
            seeds=[(x,height-1) for x in range(width)]+[(x,0) for x in range(width)]+[(0,y) for y in range(height)]+[(width-1,y) for y in range(height)]
            mask=black_components(data,width,height,seeds)
            image.paste((0,0,0),(0,0),Image.frombytes('L',(width,height),mask).resize(image.size,Image.Resampling.NEAREST))
            record.update(black_reserve_pixels=mask.count(255),black_reserve_sha256=digest(mask))
        path=output/record['prepared']
        image.save(path)
        record.update(asset_sha256=sha(path),size=list(image.size),inputs=inputs)
        prepared[name]=image
        records.append(record)
    assert mirror_count==16
    galleries=[]
    for page,start in enumerate(range(0,len(records),8),1):
        sheet=Image.new('RGB',(800,1080),(235,231,222))
        for i,record in enumerate(records[start:start+8]):
            im=prepared[record['name']].copy()
            im.thumbnail((384,240),Image.Resampling.LANCZOS)
            x,y=i%2*400,i//2*270
            sheet.paste(im,(x+(400-im.width)//2,y+24+(240-im.height)//2))
            ImageDraw.Draw(sheet).text((x+8,y+7),record['name']+' HD',fill=(0,0,0))
        path=output/f'HD-remaining-{page}-v{args.version}.png'
        sheet.save(path)
        galleries.append({'asset':path.name,'sha256':sha(path)})
    result={'classification':'source-calibrated generated HD drafts; not a runtime-completion claim',
            'version':args.version,'tool_sha256':sha(Path(__file__)),'source_intake_sha256':sha(source_manifest),
            'generation_metadata_sha256':sha(gen_path),'records':records,'exact_mirrored_frames':mirror_count,'galleries':galleries}
    (output/'draft-manifest.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    (output/'prepare-source.txt').write_bytes(Path(__file__).read_bytes())
    print(json.dumps({'sources':len(records),'exact_mirrored_frames':mirror_count,'output':str(output)}))


if __name__=='__main__':
    main()
