"""準備四項部分來源的已知區域與既有手繪素材，不補原始 byte。"""
import hashlib
import json
import os
from pathlib import Path
import platform

from PIL import Image, __version__ as pillow_version


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000 and pillow_version == '9.4.0'
    project, output = Path('/project'), Path('/output')
    assert output.stat().st_uid == output.stat().st_gid == 1000
    assert not (output/'profiles.json').exists()
    catalog = project/'workplace/hd-prototype/catalog'
    prefix = catalog/'remaining/source-boundary/art-prefix'
    proof_path = prefix/'source-prefix.json'
    proof = json.loads(proof_path.read_text())
    assert proof['tool_sha256'] == sha(project/'tools/hd-prefix-extract.py')
    assert proof['facts_sha256'] == sha(catalog/'remaining/source-boundary/facts.json')
    assert proof['dat_sha256'] == 'de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37'
    records, images = [], []
    for folder in ['sources', 'prepared', 'review']:
        (output/folder).mkdir()
    for source in proof['records']:
        name = source['source']
        assert name in ['B-6.OZM', 'E-4.OZM']
        assert source['unknown_rect'] == [312, 149, 320, 150]
        assert source['known_pixels'] == 47992 and source['unknown_pixels'] == 8
        planes = []
        for bit, metadata in enumerate(source['planes']):
            path = prefix/metadata['path']
            data = path.read_bytes()
            assert sha(path) == metadata['sha256'] and len(data) == (5999 if bit == 3 else 6000)
            planes.append(data)
        known = bytearray(48000)
        for position in range(48000):
            offset, shift = position//8, 7-position%8
            known[position] = (sum(((p[offset] >> shift) & 1) << bit for bit, p in enumerate(planes))
                               if offset < 5999 else 255)
        assert list(known[-8:]) == [255]*8
        source_path = output/'sources'/(name+'-known.idx')
        source_path.write_bytes(known)
        stem = name.removesuffix('.OZM')
        generation_path = catalog/'art'/(stem+'-generation-v1.json')
        generation = json.loads(generation_path.read_text())
        painting = generation['records'][0]
        assert painting['source'] == name and painting['aliases'] == source['aliases']
        raw = catalog/'art'/painting['asset']
        prompt = catalog/'art'/painting['prompt']
        assert sha(raw) == painting['sha256'] and sha(prompt) == painting['prompt_sha256']
        with Image.open(raw) as image:
            assert image.convert('RGBA').getextrema()[3] == (255, 255)
            width, height = image.size
            assert width >= 640 and height >= 300 and abs(width*150-height*320)*100 <= height*320
            # 既有素材校準的整數最近鄰取樣；沒有改寫人物或場景。
            original = image.convert('RGB')
            selected = Image.new('RGB', (640, 300))
            pixels, target = original.load(), selected.load()
            for y in range(300):
                for x in range(640):
                    target[x, y] = pixels[x*width//640, y*height//300]
        prepared = output/'prepared'/(stem+'-hd-v1.png')
        selected.save(prepared)
        images.append(selected)
        reference_path = prefix/source['reference']
        assert sha(reference_path) == source['reference_sha256']
        comparison = Image.new('RGBA', (1280, 300), (0, 0, 0, 0))
        reference = Image.open(reference_path).convert('RGBA').resize((640, 300), Image.Resampling.NEAREST)
        comparison.paste(reference, (0, 0))
        comparison.paste(selected, (640, 0))
        comparison_path = output/'review'/(stem+'-source-HD.png')
        comparison.save(comparison_path)
        for alias in source['aliases']:
            records.append({'resource': alias, 'art_source_resource': name, 'source_status': 'partial; unknown final high-plane byte excluded',
                            'columns': 40, 'rows': 150, 'known_pixels': 47992, 'unknown_rect': source['unknown_rect'],
                            'plane_known_sha256': [p['sha256'] for p in source['planes']], 'plane_known_bytes': [6000, 6000, 6000, 5999],
                            'known_indices': 'sources/'+source_path.name, 'known_indices_sha256': sha(source_path),
                            'asset': 'partial-story/'+prepared.name, 'prepared': 'prepared/'+prepared.name,
                            'asset_sha256': sha(prepared), 'raw_asset': painting['asset'], 'raw_asset_sha256': sha(raw),
                            'raw_size': [width, height], 'prepared_size': [640, 300],
                            'prompt': painting['prompt'], 'prompt_sha256': sha(prompt),
                            'generation_metadata_sha256': sha(generation_path),
                            'local_review': 'review/'+comparison_path.name, 'local_review_sha256': sha(comparison_path),
                            'paint_rects': [[0, 0, 320, 149], [0, 149, 312, 150]],
                            'labels': [], 'runtime_scope': 'DRAFT; common verified story consumer only after READY; normal case not sampled'})
    assert sorted(r['resource'] for r in records) == ['B-6.OZM', 'B-7.OZM', 'C-10.OZM', 'E-4.OZM']
    gallery = Image.new('RGB', (640, 600))
    for row, image in enumerate(images):
        gallery.paste(image, (0, row*300))
    gallery_path = output/'hd-contact-sheet.png'
    gallery.save(gallery_path)
    result = {'classification': 'reviewable known-region story art; source unknowns are preserved, not decoded',
              'tool_sha256': sha(Path(__file__)), 'python': platform.python_version(), 'pillow': pillow_version,
              'source_proof_sha256': sha(proof_path), 'dat_sha256': proof['dat_sha256'],
              'source_count': len(records), 'art_count': len(images), 'gallery': gallery_path.name,
              'gallery_sha256': sha(gallery_path), 'records': records}
    (output/'profiles.json').write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    (output/'prepare-source.txt').write_bytes(Path(__file__).read_bytes())
    lines = ['// Code generated by tools/hd-prefix-prepare.py; DO NOT EDIT.', 'package main',
             'var hdPrefixProfiles = map[string]hdPrefixProfile{']
    for record in records:
        q = lambda value: json.dumps(value, ensure_ascii=False)
        source_path = '/project/workplace/hd-prototype/catalog/remaining/partial-story/'+record['known_indices']
        lines.append(q(record['resource'])+': {Source: '+q(source_path)+', SourceSHA: '+q(record['known_indices_sha256'])+
                     ', Asset: '+q(record['asset'])+', AssetSHA: '+q(record['asset_sha256'])+
                     ', Planes: [4]string{'+','.join(q(s) for s in record['plane_known_sha256'])+'}},')
    lines += ['}', '']
    (output/'hd-prefix-profiles.go').write_text('\n'.join(lines))
    print(json.dumps({'sources': len(records), 'art': len(images), 'profiles_sha256': sha(output/'profiles.json')}))


if __name__ == '__main__':
    main()
