"""框架的索引、palette、整區資格及逐像素獨立核對。"""
import hashlib
import json
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def inside(r, x, y):
    return r[0] <= x < r[2] and r[1] <= y < r[3]


class FrameReference:
    def __init__(self, snapshot, picture, root, enabled):
        project = Path('/project')
        folder = project/'workplace/hd-prototype/catalog/battle-frame'
        info = picture['presented']['frame']['native_indices']
        self.indices = (root/info['path']).read_bytes()
        assert len(self.indices) == 640*400 and sha(root/info['path']) == info['sha256']
        proof = picture['presented']['proof']
        assert info['sha256'] == proof['raw_indexed_sha256']
        self.palette = [tuple(info['palette'][info['dac'][i]])+(255,) for i in range(16)]
        source_proof = json.loads((folder/'source-verification.json').read_text())
        assert self.palette == [tuple(c) for c in source_proof['native_palette']]
        geometry = json.loads((folder/'battle-frame-geometry-v2.json').read_text())
        self.source = (folder/'source-probe/battle-frame-99a8497101f145c62d01934204f1c3ed74282fa6bd993fdbf4f8969eb0f7277c.idx').read_bytes()
        self.mask = (folder/'frame-mask-v2.idx').read_bytes()
        assert hashlib.sha256(self.mask).hexdigest() == geometry['mask_sha256']
        path = project/'assets/hd-b/battle-frame/battle-frame-hd-b-v2.png'
        assert sha(path) == geometry['prepared_png_sha256']
        self.asset = Image.open(path).convert('RGBA')
        self.crest = tuple(geometry['crest_region']); self.weather = tuple(geometry['weather_region'])
        self.crest_open = self.match(self.crest); self.weather_open = self.match(self.weather)
        self.enabled = enabled
        assert snapshot['battle_frame_observation']['draw_completed']
        if enabled:
            assert snapshot['battle_frame']['mode'] == 'hd'
        else:
            assert snapshot['battle_frame']['mode'] == 'fallback_missing_asset'
        cursor = proof['cursor']; x = proof['mouse_x']-cursor['HotX']; y = proof['mouse_y']-cursor['HotY']
        self.cursor_rect = (x,y,x+16,y+16)
        self.root = root

    def match(self, r):
        return all(self.indices[y*640+x] == self.source[y*640+x] for y in range(r[1],r[3]) for x in range(r[0],r[2]))

    def underlay(self, x, y, dx, dy, index):
        n = y*640+x
        if not self.enabled or not self.mask[n] or self.source[n] != index or (inside(self.weather,x,y) and not self.weather_open):
            return None
        return self.asset.getpixel((x*2+dx,y*2+dy))

    def candidate(self, x, y):
        n=y*640+x
        return self.enabled and (self.mask[n] or (self.crest_open and inside(self.crest,x,y))) and not (inside(self.weather,x,y) and not self.weather_open)

    def body(self, x, y):
        n=y*640+x
        return self.candidate(x,y) and self.indices[n] == self.source[n]


def verify_intro_frame(snapshot, picture, enhanced, original, root, enabled):
    ref = FrameReference(snapshot,picture,root,enabled)
    assert not snapshot.get('terrain_history') and not snapshot.get('terrain_sprites')
    checked = changed = painted = 0
    for y in range(400):
        for x in range(640):
            n = y*640+x; paint = ref.body(x,y); painted += paint
            for dy in range(2):
                for dx in range(2):
                    i = ((y*2+dy)*1280+x*2+dx)*4
                    a,b = enhanced[i:i+4],original[i:i+4]
                    changed += a != b
                    if inside(ref.cursor_rect,x,y):
                        continue
                    assert b == bytes(ref.palette[ref.indices[n]])
                    expected = ref.asset.getpixel((x*2+dx,y*2+dy)) if paint else tuple(b)
                    assert a == bytes(expected),('intro frame reference',x,y,dx,dy)
                    checked += 1
    report = snapshot['battle_frame']
    assert report['painted_native_pixels'] == painted
    assert (changed > 0) == enabled
    assert report['crest_HD'] == (enabled and ref.crest_open)
    assert report['weather_placeholder_HD'] == (enabled and ref.weather_open)
    if enabled:
        assert not ref.crest_open and ref.weather_open and painted > 70000
    return {'phase':'intro','changed_pixels':changed,'independent_output_pixels_verified':checked,
            'painted_frame_native_pixels':painted,'crest_HD':report['crest_HD'],
            'weather_placeholder_HD':report['weather_placeholder_HD'],
            'native_source_verified':True,'reference_composite_verified':True,
            'cursor_formula_excluded_logical_rect':ref.cursor_rect,'tool_sha256':sha(Path(__file__))}
