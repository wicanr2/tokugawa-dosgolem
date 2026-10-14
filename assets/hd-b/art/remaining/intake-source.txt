"""依目前覆蓋清單保存缺項來源與動畫逐格參考；原版圖片只留本機。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, __version__


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000 and __version__ == '9.4.0'
    project = Path('/project')
    catalog = project/'workplace/hd-prototype/catalog'
    root = catalog/'remaining'
    assert catalog.stat().st_uid == 1000 and not root.exists()
    root.mkdir()
    (root/'frames').mkdir()
    coverage_path = catalog/'art-coverage.json'
    decoded_path = catalog/'decoded/inventory.json'
    decoded = {r['name']:r for r in json.loads(decoded_path.read_text())['records']}
    missing = [r['source'] for r in json.loads(coverage_path.read_text())['records']
               if r['art_status'] == 'missing' and r['source'] in decoded and decoded[r['source']]['status'] == 'decoded']
    assert len(missing) == 20
    palette = json.loads((catalog/'battle-frame/source-verification.json').read_text())['native_palette']
    palette = [v for c in palette for v in c[:3]]+[0]*(768-48)
    records = []
    images = []
    for name in missing:
        source = decoded[name]
        data = (catalog/'decoded'/(name+'.idx')).read_bytes()
        assert hashlib.sha256(data).hexdigest() == source['indices_sha256']
        width, height = source['width'], source['height']
        im = Image.frombytes('P',(width,height),data)
        im.putpalette(palette)
        im = im.convert('RGB')
        path = root/(name+'-reference.png')
        im.resize((width*2,height*2),Image.Resampling.NEAREST).save(path)
        record = {'source':source,'reference':path.name,'reference_sha256':sha(path),
                  'palette_basis':'canonical battle palette only; normal custom palette must be verified separately'}
        if name.startswith('ANIME'):
            assert (width,height) == (624,300)
            frames = []
            for index in range(6):
                x,y = index%3*208,index//3*150
                frame = im.crop((x,y,x+208,y+150))
                target = root/'frames'/f'{name}-{index}-reference.png'
                frame.resize((624,450),Image.Resampling.NEAREST).save(target)
                indices = b''.join(data[(y+j)*624+x:(y+j)*624+x+208] for j in range(150))
                frames.append({'index':index,'source_rect':[x,y,x+208,y+150],
                               'indices_sha256':hashlib.sha256(indices).hexdigest(),
                               'nonzero_pixels':sum(v!=0 for v in indices),
                               'reference':str(target.relative_to(root)),'reference_sha256':sha(target)})
            record['frames'] = frames
        records.append(record)
        thumb = im.copy();thumb.thumbnail((384,240),Image.Resampling.NEAREST)
        cell = Image.new('RGB',(400,270),(235,231,222));cell.paste(thumb,((400-thumb.width)//2,24+(240-thumb.height)//2))
        ImageDraw.Draw(cell).text((10,7),name,fill=(0,0,0));images.append(cell)
    for page,start in enumerate(range(0,len(images),8),1):
        sheet=Image.new('RGB',(800,270*4),(235,231,222))
        for i,im in enumerate(images[start:start+8]):sheet.paste(im,(i%2*400,i//2*270))
        sheet.save(root/f'original-remaining-{page}.png')
    result = {'classification':'local missing art intake; original images excluded from Git',
              'tool_sha256':sha(Path(__file__)),'coverage_sha256':sha(coverage_path),
              'decoded_inventory_sha256':sha(decoded_path),'records':records}
    (root/'sources.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    (root/'intake-source.txt').write_bytes(Path(__file__).read_bytes())
    print(json.dumps({'sources':len(records),'unique_animation_atlases':len({r['source']['indices_sha256'] for r in records if r['source']['name'].startswith('ANIME')})}))


if __name__ == '__main__':
    main()
