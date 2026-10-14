"""用共用手繪圖分區校準城牆幾何；原版索引只提供透明輪廓。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, __version__ as pillow_version


WALLS = list(range(4, 18)) + [20, 21]
SCALE = 8
# 矩形採原版 48×48 圖格座標。H/V 為材質呈現分區，不是原版規則欄位。
LAYOUT = {
    4: [('V', (3, 4, 22, 48)), ('H', (15, 4, 48, 34))],
    5: [('F', (0, 34, 48, 38)), ('H', (0, 4, 48, 34)), ('G', (9, 0, 38, 38))],
    6: [('H', (0, 4, 48, 34))],
    7: [('H', (0, 4, 30, 34)), ('V', (27, 4, 46, 48)), ('R', (0, 4, 35, 14))],
    8: [('V', (3, 0, 22, 44)), ('H', (3, 14, 48, 44))],
    9: [('F', (0, 44, 48, 48)), ('H', (0, 14, 48, 44)), ('G', (9, 10, 38, 48))],
    10: [('H', (0, 14, 48, 44))],
    11: [('V', (27, 0, 46, 44)), ('H', (0, 14, 46, 44))],
    12: [('V', (27, 0, 46, 48))],
    13: [('V', (3, 0, 22, 48))],
    14: [('V', (27, 0, 46, 34)), ('H', (27, 4, 48, 34))],
    15: [('V', (3, 0, 22, 34)), ('H', (0, 4, 22, 34))],
    16: [('V', (27, 0, 46, 44)), ('H', (27, 14, 48, 44))],
    17: [('V', (3, 0, 22, 44)), ('H', (0, 14, 22, 44))],
    20: [('V', (3, 0, 22, 48)), ('B20', (3, 13, 23, 31))],
    21: [('V', (27, 0, 46, 48)), ('B21', (27, 13, 46, 31))],
}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rect8(rect):
    return tuple(v*SCALE for v in rect)


def wrap_texture(im, axis):
    """在接縫的第一像素取共同色，只對一原生像素寬的內側作漸變。"""
    im = im.copy()
    length, cross = (im.width, im.height) if axis == 'x' else (im.height, im.width)
    for q in range(cross):
        a = im.getpixel((0,q) if axis == 'x' else (q,0))
        b = im.getpixel((length-1,q) if axis == 'x' else (q,length-1))
        common = tuple((x+y+1)//2 for x,y in zip(a,b))
        for d in range(SCALE):
            for at in [d,length-1-d]:
                point = (at,q) if axis == 'x' else (q,at)
                own = im.getpixel(point)
                im.putpixel(point,tuple((c*(SCALE-1-d)+v*d+(SCALE-1)//2)//(SCALE-1) for c,v in zip(common,own)))
    return im


def integrate(project, output, folder):
    terrain = project/'workplace/hd-prototype/catalog/terrain'
    draft_path, review_path = folder/'draft-manifest.json', folder/'review.json'
    draft, review = [json.loads(p.read_text()) for p in [draft_path,review_path]]
    assert draft['tool_sha256'] == sha(folder/'prepare-source.txt')
    assert review['verified'] and review['approved_indices'] == WALLS
    assert review['draft_manifest_sha256'] == sha(draft_path)
    base_path = terrain/'objects/expanded/profiles.json'
    base = json.loads(base_path.read_text())
    assert draft['base_profiles_sha256'] == sha(base_path)
    selected = {r['index']: r for r in draft['records']}
    assert set(selected) == set(WALLS)
    reused = {10:6,13:12}
    records = []
    prepared = output/'prepared'
    assert not prepared.exists()
    prepared.mkdir()
    for r in base['records']:
        r = dict(r)
        i = r['index']
        if not r['asset']:
            assert i in [24,25]
            records.append(r)
            continue
        if i in selected:
            owner = reused.get(i,i)
            art = selected[owner]
            path = folder/art['asset']
            assert sha(path) == art['asset_sha256']
            if owner != i:
                a,b = [Image.open(project/base['records'][n]['source']).convert('RGBA') for n in [i,owner]]
                a,b = [im.crop(im.getchannel('A').getbbox()) for im in [a,b]]
                assert a.size == b.size and a.tobytes() == b.tobytes()
            for key in ['prompt','prompt_sha256']:
                r.pop(key,None)
            r.update(asset='objects/walls/'+art['asset'], asset_sha256=art['asset_sha256'], crop=art['crop'],
                     asset_source_index=owner, asset_local_source=str((terrain/'objects/walls'/folder.name/art['asset']).relative_to(project)),
                     material_inputs=draft['inputs'], wall_geometry=selected[i], crop_margin=0)
            asset = Image.open(path).convert('RGBA')
            bounds,crop = r['source_bounds'],r['crop']
            preview = Image.new('RGBA',(96,96))
            width,height = (bounds[2]-bounds[0])*2,(bounds[3]-bounds[1])*2
            for y in range(height):
                for x in range(width):
                    preview.putpixel((bounds[0]*2+x,bounds[1]*2+y),asset.getpixel(
                        (crop[0]+x*(crop[2]-crop[0])//width,crop[1]+y*(crop[3]-crop[1])//height)))
            dest = prepared/f'object-{i:03d}-hd-preview.png'
            preview.save(dest)
        else:
            old = terrain/'objects/expanded'/r['prepared']
            assert sha(old) == r['prepared_sha256']
            dest = prepared/old.name
            dest.write_bytes(old.read_bytes())
            r['asset_source_index'] = i
        r.update(prepared=str(dest.relative_to(output)),prepared_sha256=sha(dest))
        records.append(r)
    assert len(records) == 26 and sum(bool(r['asset']) for r in records) == 24
    gallery = Image.new('RGB',(6*192,4*216),(222,216,197))
    draw = ImageDraw.Draw(gallery)
    for r in records[:24]:
        i = r['index']
        im = Image.open(output/r['prepared']).resize((192,192),Image.Resampling.NEAREST)
        draw.text((i%6*192+4,i//6*216+4),f'objects {i:03d}',fill=(0,0,0))
        gallery.paste(im,(i%6*192,i//6*216+24),im)
    gallery_path = output/'object-HD-gallery-v2.png'
    assert not gallery_path.exists()
    gallery.save(gallery_path)
    lines = ['// Code generated by tools/hd-wall-prepare.py; DO NOT EDIT.','package main','import "image"',
             '// 只記來源索引與呈現幾何，不推定地圖物件用途。',
             'type hdTerrainObjectProfile struct { Asset, AssetSHA string; SourceBounds, Crop image.Rectangle }',
             'var hdTerrainObjectProfiles = map[int]hdTerrainObjectProfile{']
    for r in records[:24]:
        a,b = ','.join(map(str,r['source_bounds'])),','.join(map(str,r['crop']))
        lines.append(f'{r["index"]}: {{{json.dumps(r["asset"])}, {json.dumps(r["asset_sha256"])}, image.Rect({a}), image.Rect({b})}},')
    lines += ['}','']
    target = project/'tools/hd-terrain-object-profiles.go'
    prior = terrain/'objects/expanded/runtime/source-snapshot/tools/hd-terrain-object-profiles.go'
    assert target.stat().st_uid == 1000 and sha(target) == sha(prior)
    previous_sha = sha(target)
    target.write_text('\n'.join(lines))
    result = {'classification':'all 24 nonempty object sources reviewed; calibrated wall geometry; original blank sources preserved',
              'tool_sha256':sha(Path(__file__)), 'draft_tool_sha256':draft['tool_sha256'],
              'base_profiles_sha256':sha(base_path),'prior_Go_profile_sha256':previous_sha,
              'draft_manifest':str(draft_path.relative_to(output)), 'draft_manifest_sha256':sha(draft_path),
              'review':str(review_path.relative_to(output)), 'review_sha256':sha(review_path),
              'source_slots':26, 'nonempty_sources':24, 'approved_indices':list(range(24)), 'empty_indices':[24,25],
              'wall_source_indices':WALLS,'exact_source_crop_reuse':reused,'unchanged_indices':base['approved_indices'],
              'scale':2,'records':records,'gallery':gallery_path.name,'gallery_sha256':sha(gallery_path)}
    target = output/'profiles.json'
    assert not target.exists()
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    (output/'profiles-source.txt').write_bytes(Path(__file__).read_bytes())
    print(json.dumps({'approved_objects':24,'calibrated_walls':16,'empty_indices':[24,25]}))


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000 and pillow_version == '9.4.0'
    parser = argparse.ArgumentParser()
    parser.add_argument('--revision', type=int, default=1)
    parser.add_argument('--integrate', action='store_true')
    args = parser.parse_args()
    assert 1 <= args.revision <= 20
    project, output = Path('/project'), Path('/output')
    assert output.stat().st_uid == output.stat().st_gid == 1000
    terrain = project/'workplace/hd-prototype/catalog/terrain'
    old = json.loads((terrain/'objects/expanded/profiles.json').read_text())
    by_index = {r['index']: r for r in old['records']}
    folder = output/f'draft-v{args.revision}'
    if args.integrate:
        integrate(project,output,folder)
        return
    assert not folder.exists()
    folder.mkdir()
    (folder/'prepare-source.txt').write_bytes(Path(__file__).read_bytes())
    textures, inputs = {}, []
    # 生成圖的實際分區由逐張檢視確認。Alpha 在這裡是材質的空白區，
    # 最終物件透明度來自原版 index-zero 輪廓，不回用原版 RGB。
    for name, i, crop in [('H',6,(0,333,1254,918)), ('V',12,(715,0,980,1254)),
                          ('G',5,(260,63,992,1143)), ('B20',20,(162,320,454,780)),
                          ('B21',21,(753,320,1087,780))]:
        r = by_index[i]
        path = terrain/'art'/r['asset']
        assert sha(path) == r['asset_sha256']
        im = Image.open(path).convert('RGB').crop(crop)
        textures[name] = im
        inputs.append({'part':name, 'source_index':i, 'asset':r['asset'], 'asset_sha256':sha(path),
                       'prompt':r['prompt'], 'prompt_sha256':r['prompt_sha256'], 'crop':list(crop)})
    horizontal = wrap_texture(textures['H'].resize((48*SCALE,30*SCALE),Image.Resampling.LANCZOS),'x')
    vertical = wrap_texture(textures['V'].resize((19*SCALE,48*SCALE),Image.Resampling.LANCZOS),'y')
    gate = Image.new('RGB',(28*SCALE,38*SCALE))
    for sy0, sy1, dy0, dy1 in [(0,437,0,14),(437,737,14,25),(737,1080,25,38)]:
        part = textures['G'].crop((0,sy0,textures['G'].width,sy1)).resize((28*SCALE,(dy1-dy0)*SCALE),Image.Resampling.LANCZOS)
        gate.paste(part,(0,dy0*SCALE))
    records = []
    for i in WALLS:
        r = by_index[i]
        source_path = project/r['source']
        assert sha(source_path) == r['source_png_sha256']
        original = Image.open(source_path).convert('RGBA')
        assert original.size == (48,48) and original.getchannel('A').getextrema() == (0,255)
        canvas = Image.new('RGB',(48*SCALE,48*SCALE),(27,28,26))
        for kind, rect in LAYOUT[i]:
            x0,y0,x1,y1 = rect
            if kind in ['H','R']:
                # 全部橫向構件沿同一個 48 像素材質相位裁切。
                patch = horizontal.crop((x0*SCALE,0,x1*SCALE,(y1-y0)*SCALE))
            elif kind == 'F':
                patch = horizontal.crop((x0*SCALE,26*SCALE,x1*SCALE,30*SCALE))
            elif kind == 'V':
                patch = vertical.crop((0,y0*SCALE,19*SCALE,y1*SCALE))
            elif kind == 'G':
                patch = gate.resize(((x1-x0)*SCALE,(y1-y0)*SCALE),Image.Resampling.LANCZOS)
            else:
                patch = textures[kind].resize(((x1-x0)*SCALE,(y1-y0)*SCALE),Image.Resampling.LANCZOS)
            assert patch.size == ((x1-x0)*SCALE,(y1-y0)*SCALE)
            canvas.paste(patch,(x0*SCALE,y0*SCALE))
        alpha = original.getchannel('A').filter(ImageFilter.GaussianBlur(0.65)).resize(canvas.size,Image.Resampling.LANCZOS)
        bounds = original.getchannel('A').getbbox()
        # 接縫延續到原版圖格邊緣，來源範圍外完全透明。
        clipped = Image.new('L',canvas.size)
        clipped.paste(alpha.crop(rect8(bounds)),rect8(bounds)[:2])
        canvas = canvas.convert('RGBA')
        canvas.putalpha(clipped)
        path = folder/f'wall-{i:03d}-calibrated-v{args.revision}.png'
        canvas.save(path)
        preview = canvas.resize((96,96),Image.Resampling.NEAREST)
        path96 = folder/f'wall-{i:03d}-preview.png'
        preview.save(path96)
        records.append({'index':i, 'source':r['source'], 'source_png_sha256':r['source_png_sha256'],
                        'source_indices_sha256':r['source_indices_sha256'], 'source_bounds':r['source_bounds'],
                        'asset':path.name, 'asset_sha256':sha(path), 'crop':list(rect8(bounds)),
                        'parts':[(k,list(v)) for k,v in LAYOUT[i]], 'preview':path96.name, 'preview_sha256':sha(path96)})
    pages = []
    for page in range(3):
        sheet = Image.new('RGB',(1200,480),(222,216,197))
        draw = ImageDraw.Draw(sheet)
        for n,r in enumerate(records[page*6:page*6+6]):
            x,y = n%3*400,n//3*240
            im = Image.open(project/r['source']).convert('RGBA').resize((192,192),Image.Resampling.NEAREST)
            hd = Image.open(folder/r['asset']).resize((192,192),Image.Resampling.LANCZOS)
            draw.text((x+4,y+4),f"wall {r['index']:03d}  original / HD",fill=(0,0,0))
            sheet.paste(im,(x+4,y+24),im)
            sheet.paste(hd,(x+204,y+24),hd)
        path = folder/f'review-page-{page+1:02d}.png'
        sheet.save(path)
        pages.append({'path':path.name, 'sha256':sha(path)})
    gallery = Image.new('RGB',(8*192,2*216),(222,216,197))
    draw = ImageDraw.Draw(gallery)
    for n,r in enumerate(records):
        im = Image.open(folder/r['asset']).resize((192,192),Image.Resampling.LANCZOS)
        draw.text((n%8*192+4,n//8*216+4),f"wall {r['index']:03d}",fill=(0,0,0))
        gallery.paste(im,(n%8*192,n//8*216+24),im)
    path = folder/f'wall-HD-gallery-v{args.revision}.png'
    gallery.save(path)
    fixtures = []
    for name, grid in [('enclosure',[[4,6,5,6,7],[13,None,None,None,12],[20,None,None,None,21],
                                     [13,None,None,None,12],[8,10,9,10,11]]),
                       ('partial',[[14,6,15],[16,10,17]])]:
        hd = Image.new('RGB',(len(grid[0])*192,len(grid)*192),(52,85,42))
        original = hd.copy()
        for y,row in enumerate(grid):
            for x,i in enumerate(row):
                if i is None:
                    continue
                im = Image.open(folder/f'wall-{i:03d}-calibrated-v{args.revision}.png').resize((192,192),Image.Resampling.LANCZOS)
                hd.paste(im,(x*192,y*192),im)
                im = Image.open(project/by_index[i]['source']).resize((192,192),Image.Resampling.NEAREST)
                original.paste(im,(x*192,y*192),im)
        hpath = folder/f'wall-HD-{name}-v{args.revision}.png'
        hd.save(hpath)
        sheet = Image.new('RGB',(hd.width*2,hd.height))
        sheet.paste(original,(0,0));sheet.paste(hd,(hd.width,0))
        local = folder/f'wall-{name}-comparison.png';sheet.save(local)
        fixtures.append({'kind':name,'grid':grid,'HD':hpath.name,'HD_sha256':sha(hpath),
                         'local_comparison':local.name,'local_comparison_sha256':sha(local)})
    result = {'classification':'wall geometry calibration draft; not map rules or production profiles',
              'tool_sha256':sha(Path(__file__)), 'base_profiles_sha256':sha(terrain/'objects/expanded/profiles.json'),
              'scale':SCALE, 'alpha':'original index-zero silhouette blurred by 0.65 native pixel and resampled; clipped to original bounds; no original RGB copied',
              'edge_wrap':'H columns and V rows share endpoint RGB; one native pixel inward blend',
              'inputs':inputs, 'records':records, 'review_pages':pages, 'fixtures':fixtures,
              'gallery':path.name, 'gallery_sha256':sha(path)}
    (folder/'draft-manifest.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'draft_revision':args.revision,'wall_sources':len(records),'gallery':str(path)}))


if __name__ == '__main__':
    main()
