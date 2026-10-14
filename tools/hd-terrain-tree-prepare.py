"""核對首戰樹木完整背景與透明素材幾何，生成本機預覽及編譯期指紋。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, __version__ as pillow_version


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rect(d):
    r = d['logical_rect']
    return r['Min']['X'], r['Min']['Y'], r['Max']['X'], r['Max']['Y']


def inside(r, x, y):
    return r[0] <= x < r[2] and r[1] <= y < r[3]


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000 and pillow_version == '9.4.0'
    project, output = Path('/project'), Path('/output')
    terrain = project/'workplace/hd-prototype/catalog/terrain'
    summary_path = terrain/'objects/verification-summary.json'
    summary = json.loads(summary_path.read_text())
    assert summary['original_cursor_verification']['verified']
    receipt_path = terrain/'objects/desktop/desktop-terrain.json'
    assert sha(receipt_path) == summary['records'][0]['receipt_sha256']
    receipt = json.loads(receipt_path.read_text())
    state = receipt['desktop']['hd_observation']['terrain']
    assert state['completed_returns'] == 134 and state['completed_sprite_returns'] == 11
    floors, sprites = state['draws'], state['sprites']
    idx_path = terrain.parent/'decoded/MAP_CHIP.OZM.idx'
    idx = idx_path.read_bytes()
    assert sha(idx_path) == json.loads((terrain/'objects/source-profiles.json').read_text())['source_indices_sha256']
    palette = json.loads((terrain/'trace/native-floor-palette-verification.json').read_text())['palette']
    native_path = project/receipt['frame']['path']
    assert sha(native_path) == receipt['frame']['png_sha256']
    native = Image.open(native_path).convert('RGB')
    cursor = receipt['mouse_cursor']
    last = receipt['mouse_input_events'][-1]
    cx, cy = last['x']-cursor['HotX'], last['y']-cursor['HotY']
    results = []
    for candidate in sprites:
        if (candidate['group'],candidate['index']) != ('objects',0):
            continue
        r = rect(candidate)
        expected = Image.new('RGB',(48,48))
        covered = visible = 0
        for y in range(r[1],r[3]):
            for x in range(r[0],r[2]):
                floor = next((d for d in floors if inside(rect(d),x,y)),None)
                if floor is None:
                    continue
                covered += 1
                fr = rect(floor)
                value = idx[(floor['index']//13*32+y-fr[1])*640+floor['index']%13*48+x-fr[0]]
                owner = floor['draw_order']
                for s in sprites:
                    if s['draw_order'] <= floor['draw_order'] or not inside(rect(s),x,y):
                        continue
                    sr = rect(s)
                    ax = s['index']%13*48
                    ay = 256+s['index']//13*48 if s['group']=='objects' else 352
                    v = idx[(ay+y-sr[1])*640+ax+x-sr[0]]
                    if v:
                        value,owner = v,s['draw_order']
                visible += owner == candidate['draw_order']
                if cursor['Defined'] and cursor['DisplayCount'] >= 0 and cx<=x<cx+16 and cy<=y<cy+16:
                    row, bit = y-cy, 0x8000>>(x-cx)
                    if not cursor['Mask'][row]&bit:
                        value = 0
                    if cursor['Mask'][row+16]&bit:
                        value ^= 15
                expected.putpixel((x-r[0],y-r[1]),tuple(palette[value]))
        reason = 'fallback_background_incomplete'
        if not visible:
            reason = 'fallback_overwritten_sprite'
        elif covered == 2304:
            assert expected.tobytes() == native.crop(r).tobytes(), ('tree complete scene mismatch',r)
            reason = 'ready_complete_scene'
        results.append({'rect':list(r),'order':candidate['draw_order'],'background_pixels':covered,'visible_native_pixels':visible,'validation':reason})
    assert sum(r['validation']=='ready_complete_scene' for r in results)==3
    source_path = terrain/'objects/sources/objects-000-original.png'
    source = Image.open(source_path).convert('RGBA')
    assert source.size==(48,48) and source.getbbox()==(0,0,47,47)
    asset_path = terrain/'art/object-000-hd-b-v2.png'
    asset = Image.open(asset_path).convert('RGBA')
    assert asset.size==(1254,1254) and asset.getextrema()[3]==(0,255)
    alpha = asset.getchannel('A')
    revision_path = terrain/'art/object-revisions.json'
    revision = json.loads(revision_path.read_text())['records'][0]
    assert sha(asset_path)==revision['asset_sha256']
    assert alpha.point(lambda v:255 if v>=8 else 0).getbbox()==tuple(revision['alpha8_bbox'])
    crop = tuple(revision['review_crop'])
    assert crop==(7,0,1247,1245)
    outside_max = max(alpha.getpixel((x,y)) for y in range(asset.height) for x in range(asset.width) if not inside(crop,x,y))
    assert outside_max<=7
    prepared = Image.new('RGBA',(96,96))
    for y in range(94):
        for x in range(94):
            prepared.putpixel((x,y),asset.getpixel((crop[0]+x*(crop[2]-crop[0])//94,crop[1]+y*(crop[3]-crop[1])//94)))
    prepared.save(output/'tree-prepared-v2.png')
    sheet=Image.new('RGB',(768,296),(219,211,190));draw=ImageDraw.Draw(sheet)
    full=asset.resize((96,96),Image.Resampling.NEAREST)
    for i,(im,label) in enumerate([(source.resize((96,96),Image.Resampling.NEAREST),'original 48x48'),(full,'draft full canvas'),(prepared,'fit to source 47x47')]):
        view=im.resize((256,256),Image.Resampling.NEAREST);sheet.paste(view,(i*256,24),view);draw.text((i*256+8,6),label,fill='black')
    sheet.save(output/'tree-geometry-review-v2.png')
    q=json.dumps
    lines=['// Code generated by tools/hd-terrain-tree-prepare.py; DO NOT EDIT.','package main','import "image"',
           'const hdTerrainTreeAsset = "object-000-hd-b-v2.png"',
           'const hdTerrainTreeAssetSHA = '+q(sha(asset_path)),
           'const hdTerrainTreeSourceSHA = '+q(sha(source_path)),
           'var hdTerrainTreeCrop = image.Rect(7,0,1247,1245)',
           'var hdTerrainTreeSourceBounds = image.Rect(0,0,47,47)','']
    (project/'tools/hd-terrain-tree-profile.go').write_text('\n'.join(lines))
    result={'classification':'normal source-backed tree scene and disposable alpha geometry; not runtime acceptance',
            'source_summary_sha256':sha(summary_path),'source_original_png_sha256':sha(source_path),'asset':asset_path.name,
            'asset_sha256':sha(asset_path),'HD_crop':list(crop),'source_opaque_bounds':[0,0,47,47],
            'maximum_alpha_outside_HD_crop':outside_max,'prepared_png_sha256':sha(output/'tree-prepared-v2.png'),
            'tool_sha256':sha(Path(__file__)),'records':results}
    target=output/'tree-source-verification.json'
    if target.exists():
        historical=output/('tree-source-verification-'+sha(target)+'.json')
        if not historical.exists():
            historical.write_bytes(target.read_bytes())
    target.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'full_tree_scenes_verified':3,'overwritten_trees':sum(r['validation']=='fallback_overwritten_sprite' for r in results),'clipped_tree_fallbacks':1,'outside_crop_alpha_max':outside_max,'verification_sha256':sha(target)}))


if __name__=='__main__':
    main()
