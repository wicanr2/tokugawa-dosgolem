"""校準生成戰場框架，保存明示幾何與原版動態區域保護的草稿。"""
import hashlib
import json
import os
from collections import deque
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def inside(r, x, y):
    return r[0] <= x < r[2] and r[1] <= y < r[3]


def warp_coordinate(pos, target, source):
    for i in range(len(target)-1):
        if target[i]*2 <= pos < target[i+1]*2:
            return source[i]+(pos-target[i]*2)*(source[i+1]-source[i])//((target[i+1]-target[i])*2)
    raise ValueError(pos)


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, output = Path('/project'), Path('/output')
    assert output.stat().st_uid == 1000
    verified = json.loads((output/'source-verification.json').read_text())
    index_sha = '99a8497101f145c62d01934204f1c3ed74282fa6bd993fdbf4f8969eb0f7277c'
    source_path = project/'workplace/hd-prototype/catalog/hd-assets'/f'MAPGAMEN.OZM-{index_sha}.idx'
    assert sha(source_path) == verified['indices_sha256'] == index_sha
    indices = source_path.read_bytes()
    art_path = project/'assets/hd-b/art/MAPGAMEN-hd-b-v1.png'
    assert sha(art_path) == '0a8fd2e1ead2bc8c5a83bb2d4da0cb9f8e81e7fa5bdc3c7d4b7fcaaa360af93e'
    art = Image.open(art_path).convert('RGBA')
    assert art.size == (1586, 992) and art.getextrema()[3] == (255, 255)
    native_x, generated_x = [0,27,64,400,430,640], [0,66,150,990,1060,1586]
    native_y, generated_y = [0,18,54,214,226,274,290,344,384,400], [0,45,136,515,552,670,702,838,935,992]
    post_native_y, post_generated_y = [0,88,120,152,184,216,248,280,312,400], [0,216,291,367,445,521,599,675,755,992]
    hx = [warp_coordinate(x, native_x, generated_x) for x in range(1280)]
    hy = [warp_coordinate(y, native_y, generated_y) for y in range(800)]
    post_hy = [warp_coordinate(y, post_native_y, post_generated_y) for y in range(800)]
    hd = Image.new('RGBA', (1280, 800))
    pixels = art.load(); target = hd.load()
    for y in range(800):
        for x in range(1280):
            target[x,y] = pixels[hx[x],post_hy[y] if x < 128 or 800 <= x < 860 else hy[y]]
    crest_region, crest_bounds, crest_crop = (100,70,360,320), (111,77,352,313), (294,194,851,726)
    # 徽記獨立對齊原版實際範圍，不沿用外框分區變形。
    for y in range(crest_region[1]*2, crest_region[3]*2):
        for x in range(crest_region[0]*2, crest_region[2]*2):
            target[x,y] = (0,0,0,255)
    w, h = (crest_bounds[2]-crest_bounds[0])*2, (crest_bounds[3]-crest_bounds[1])*2
    for y in range(h):
        for x in range(w):
            target[crest_bounds[0]*2+x,crest_bounds[1]*2+y] = pixels[crest_crop[0]+x*(crest_crop[2]-crest_crop[0])//w,crest_crop[1]+y*(crest_crop[3]-crest_crop[1])//h]
    black = bytearray(v in (0,8) for v in indices)
    mask = bytearray(not b for b in black)
    seen = bytearray(640*400)
    filled = []
    for start in range(len(black)):
        if not black[start] or seen[start]:
            continue
        seen[start] = 1; queue = deque([start]); component = []; touches_edge = False
        while queue:
            n = queue.popleft(); component.append(n); x, y = n%640, n//640
            touches_edge |= x in (0,639) or y in (0,399)
            for xx, yy in ((x-1,y),(x+1,y),(x,y-1),(x,y+1)):
                if 0 <= xx < 640 and 0 <= yy < 400:
                    nn = yy*640+xx
                    if black[nn] and not seen[nn]:
                        seen[nn] = 1; queue.append(nn)
        if not touches_edge and len(component) <= 256:
            for n in component:
                mask[n] = 1
            filled.append(len(component))
    weather_region = (464,291,608,365)
    minimap_region = (434,16,632,214)
    for y in range(400):
        for x in range(640):
            if any(inside(r,x,y) for r in (crest_region,minimap_region)):
                mask[y*640+x] = 0
    hd_path = output/'battle-frame-hd-b-v2.png'
    hd.save(hd_path)
    mask_path = output/'frame-mask-v2.idx'; mask_path.write_bytes(mask)
    mask_preview = Image.frombytes('L', (640,400), bytes(v*255 for v in mask))
    mask_preview.save(output/'frame-mask-v2.png')
    original = Image.open(output/'source-probe'/f"battle-frame-return-{verified['png_sha256']}.png").convert('RGBA')
    intro = original.resize((1280,800), Image.Resampling.NEAREST)
    for y in range(800):
        for x in range(1280):
            if mask[y//2*640+x//2] or inside(crest_region,x//2,y//2):
                intro.putpixel((x,y), target[x,y])
    intro.save(output/'frame-intro-draft-v2.png')
    source_scene = json.loads((project/'workplace/hd-prototype/catalog/terrain/scene/source-scene-verification.json').read_text())
    protected = [r['rect'] for r in source_scene['records'] if r['group'] != 'floor' and r['normal_source_scene_matches']]
    native_path = project/'workplace/startup-probe/frame-b6b7a0c895e4057887515967f6512abc567549fb212073b6b20db17d74c6d846.png'
    native = Image.open(native_path).convert('RGBA')
    weather_path = project/'workplace/hd-prototype/catalog/hd-assets/WEATHER1.OZM-ad397c7ae15a08571b43e66fed4fc9fc9236c37909b374fe69ef1e51771066c0.png'
    assert native.crop(weather_region).tobytes() == Image.open(weather_path).convert('RGBA').tobytes()
    preview = Image.open(project/'workplace/hd-prototype/catalog/terrain/scene/desktop/gui-scene-enabled.png').convert('RGBA')
    palette = verified['native_palette']
    changed = preserved = 0
    for y in range(400):
        for x in range(640):
            n = y*640+x
            if not mask[n]:
                continue
            if inside(weather_region,x,y):
                preserved += 1; continue
            if native.getpixel((x,y)) != tuple(palette[indices[n]]) or any(inside(r,x,y) for r in protected):
                preserved += 1; continue
            for dy in range(2):
                for dx in range(2):
                    px, py = x*2+dx, y*2+dy
                    if preview.getpixel((px,py)) == native.getpixel((x,y)):
                        preview.putpixel((px,py), target[px,py]); changed += 1
    preview.save(output/'frame-battle-draft-v2.png')
    result = {'classification':'discardable generated-frame geometry draft; not runtime or GUI acceptance',
              'whole_game_HD_complete':False, 'source_verification_sha256':sha(output/'source-verification.json'),
              'source_indices_sha256':index_sha,'asset':'MAPGAMEN-hd-b-v1.png','asset_sha256':sha(art_path),
              'prepared_asset':hd_path.name,'prepared_png_sha256':sha(hd_path),
              'native_x':native_x,'generated_x':generated_x,'native_y':native_y,'generated_y':generated_y,
              'post_native_y':post_native_y,'post_generated_y':post_generated_y,'post_native_x_regions':[[0,64],[400,430]],
              'crest_region':crest_region,'crest_source_bounds':crest_bounds,'crest_HD_crop':crest_crop,
              'weather_region':weather_region,'minimap_region':minimap_region,
              'weather_source_png_sha256':sha(weather_path),'weather_native_full_rectangle_match':True,
              'mask_sha256':sha(mask_path),'mask_native_pixels':sum(mask),'filled_black_component_sizes':filled,
              'draft_preview_changed_output_pixels':changed,'draft_preserved_native_mask_pixels':preserved,
              'preview_limit':'whole visible sprite rectangles conservatively preserved; HD underlay and exact index protection pending',
              'tool_sha256':sha(Path(__file__))}
    (output/'battle-frame-geometry-v2.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'mask_native_pixels':sum(mask),'prepared_png_sha256':sha(hd_path),'geometry_sha256':sha(output/'battle-frame-geometry-v2.json')}))


if __name__ == '__main__':
    main()
