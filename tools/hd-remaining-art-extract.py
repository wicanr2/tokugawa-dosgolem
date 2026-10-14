"""重生尚缺 OZM 的本機原圖與生成參考，不猜補未解碼來源。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, output = Path('/project'), Path('/output')
    assert output.stat().st_uid == 1000
    root = project/'workplace/hd-prototype/catalog'
    coverage_path = root/'art-coverage.json'
    coverage = json.loads(coverage_path.read_text())
    pending = {r['source'] for r in coverage['records'] if r['art_status'] == 'missing'}
    decoded_path = root/'decoded/inventory.json'
    decoded = json.loads(decoded_path.read_text())
    palette = json.loads((root/'battle-frame/source-verification.json').read_text())['native_palette']
    flat = [v for c in palette for v in c[:3]]+[0]*(256*3-48)
    records = []
    for source in decoded['records']:
        name = source['name']
        if name not in pending or source['status'] != 'decoded':
            continue
        path = root/'decoded'/f'{name}.idx'
        assert sha(path) == source['indices_sha256']
        data = path.read_bytes()
        assert len(data) == source['width']*source['height'] and max(data) < 16
        im = Image.frombytes('P',(source['width'],source['height']),data)
        im.putpalette(flat); im = im.convert('RGB')
        original = output/f'{name}-original.png'
        im.save(original)
        scale = min(6,max(1,1280//im.width))
        reference = output/f'{name}-reference.png'
        im.resize((im.width*scale,im.height*scale),Image.Resampling.NEAREST).save(reference)
        records.append({'source':name,'width':im.width,'height':im.height,
                        'source_indices_sha256':source['indices_sha256'],'source_payload_sha256':source['payload_sha256'],
                        'original':original.name,'original_png_sha256':sha(original),
                        'reference':reference.name,'reference_png_sha256':sha(reference)})
    rows = (len(records)+3)//4
    sheet = Image.new('RGB',(1280,rows*240),(222,216,197)); draw = ImageDraw.Draw(sheet)
    for i,r in enumerate(records):
        x,y = i%4*320,i//4*240
        draw.text((x+6,y+6),f"{r['source']} {r['width']}x{r['height']}",fill=(0,0,0))
        im = Image.open(output/r['original']); im.thumbnail((308,212),Image.Resampling.NEAREST)
        sheet.paste(im,(x+(320-im.width)//2,y+24+(212-im.height)//2))
    sheet.save(output/'remaining-contact-sheet.png')
    manifest = {'classification':'local original references; fixed palette, no runtime HD claim',
                'coverage_sha256':sha(coverage_path),'decoded_inventory_sha256':sha(decoded_path),
                'records':records,'tool_sha256':sha(Path(__file__))}
    (output/'source-references.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'original_references':len(records),'unsupported_not_filled':5}))


if __name__ == '__main__':
    main()
