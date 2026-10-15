"""擷取未完整 OZM 的已知圖像區；未知 byte 與八個像素明示保留。"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sys

from PIL import Image


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project, output = Path('/project'), Path('/output')
    assert output.stat().st_uid == 1000 and not (output / 'source-prefix.json').exists()
    catalog = project / 'workplace/hd-prototype/catalog'
    facts_path = catalog / 'remaining/source-boundary/facts.json'
    facts = json.loads(facts_path.read_text())
    dat = (project / 'original/game/DX5.DAT').read_bytes()
    assert sha(dat) == facts['dat_sha256']
    decoder_path = project / 'tools/hd-ozm-probe.py'
    assert sha(decoder_path.read_bytes()) == facts['decoder_sha256']
    spec = importlib.util.spec_from_file_location('ozm_decoder', decoder_path)
    decoder = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(decoder)
    palette = json.loads((catalog / 'remaining/history/palette-effects.json').read_text())['records']['OP4.OZM']['base_palette_rgb8']
    records = []
    for group in facts['OZM_groups']:
        if group['source'] == 'OP-FULL.OZM':
            continue
        start, size = group['dat_file_offset'], group['payload_length']
        payload = dat[start:start + size]
        assert sha(payload) == group['payload_sha256']
        captured = {}

        def trace(frame, event, arg):
            if frame.f_code.co_name == 'decode' and event == 'exception':
                local = frame.f_locals
                captured.update(planes=[bytes(x) for x in local['planes']], values=bytes(local['values']),
                                plane=local['plane'], position=local['position'])
            return trace

        sys.settrace(trace)
        try:
            decoder.decode(payload)
        except IndexError:
            pass
        finally:
            sys.settrace(None)
        columns, rows = group['columns'], group['rows']
        assert (columns, rows, captured['plane'], captured['position']) == (40, 150, 3, size)
        planes, prefix = captured['planes'], captured['values']
        assert [sha(x) for x in planes] == group['exception']['completed_plane_sha256']
        assert len(prefix) == columns * rows - 1 and sha(prefix) == group['exception']['known_partial_column_order_sha256']
        # Both layouts omit the final element. Only known elements are permuted.
        row_prefix = bytearray(columns * rows - 1)
        for x in range(columns):
            for y in range(rows):
                source, dest = x * rows + y, y * columns + x
                if source < len(prefix):
                    row_prefix[dest] = prefix[source]
        name = group['source']
        paths = []
        for bit, data in enumerate(planes + [bytes(row_prefix)]):
            path = output / f'{name}-plane-{bit}-known.bin'
            assert not path.exists()
            path.write_bytes(data)
            paths.append({'path': path.name, 'bytes': len(data), 'sha256': sha(data),
                          'layout': 'row-major plane bytes; plane 3 omits final unknown byte'})
        image = Image.new('RGBA', (columns * 8, rows))
        known = unknown = 0
        for y in range(rows):
            for x in range(columns * 8):
                offset, shift = y * columns + x // 8, 7 - x % 8
                if offset == len(row_prefix):
                    image.putpixel((x, y), (0, 0, 0, 0))
                    unknown += 1
                    continue
                index = sum(((data[offset] >> shift) & 1) << bit for bit, data in enumerate(planes))
                index |= ((row_prefix[offset] >> shift) & 1) << 3
                image.putpixel((x, y), tuple(palette[index]) + (255,))
                known += 1
        assert (known, unknown) == (47992, 8)
        raw = output / f'{name}-known-source.png'
        reference = output / f'{name}-known-reference.png'
        image.save(raw)
        image.resize((1280, 600), Image.Resampling.NEAREST).save(reference)
        records.append({'source': name, 'aliases': group['aliases'], 'payload_sha256': group['payload_sha256'],
                        'planes': paths, 'known_pixels': known, 'unknown_pixels': unknown,
                        'unknown_rect': group['unknown_output_pixels'], 'reference': reference.name,
                        'reference_sha256': sha(reference.read_bytes()), 'known_source': raw.name,
                        'known_source_sha256': sha(raw.read_bytes()),
                        'palette_scope': 'shared research palette; not a captured native frame',
                        'source_status': 'partial; no unknown byte filled and no runtime qualification claim'})
    result = {'classification': 'known-region art reference; original missing byte remains unknown',
              'tool_sha256': sha(Path(__file__).read_bytes()), 'facts_sha256': sha(facts_path.read_bytes()),
              'dat_sha256': facts['dat_sha256'], 'records': records}
    (output / 'source-prefix.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'groups': len(records), 'resources': sum(len(r['aliases']) for r in records), 'unknown_pixels_per_resource': 8}))


if __name__ == '__main__':
    main()
