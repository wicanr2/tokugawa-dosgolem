"""以已存在的正常載入 SHA 恢復唯一未知的 RAM byte，不補原始檔。"""
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import struct
import sys


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    assert Path('/.dockerenv').is_file() and os.getuid() == 1000
    project = Path('/project')
    base = project / 'workplace/hd-prototype/catalog/remaining/source-boundary'
    output = base / 'OP-FULL-runtime'
    assert not output.exists()
    receipt_path = base / 'menu-history/rcr/receipt.json'
    receipt = json.loads(receipt_path.read_text())
    assert receipt['successful_steps'] == 600000000 and receipt['stop'] == 'instruction_budget'
    decoder = project / 'tools/hd-ozm-probe.py'
    trace_tool = project / 'tools/hd-trace.go'
    assert sha(trace_tool.read_bytes()) == receipt['integration_input_sha256']['tools/hd-trace.go']
    assert sha((receipt_path.parent / 'source-snapshot/tools/hd-trace.go').read_bytes()) == sha(trace_tool.read_bytes())
    events = [event for event in receipt['hd_asset_trace']['events']
              if event['kind'] == 'load_return' and event['name'] == 'OP-FULL.OZM']
    assert len(events) == 1
    event = events[0]
    assert event['raw_columns'] == 80 and event['raw_rows'] == 400 and event['pitch_bytes'] == 80
    dat = (project / 'original/game/DX5.DAT').read_bytes()
    assert sha(dat) == 'de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37'
    payload = dat[2362614:2362614 + 63949]
    assert sha(payload) == '2e1766de32820be66ded41f855297fe622edcd5fcef6f68eefd6c35656f78bae'
    spec = importlib.util.spec_from_file_location('original_decoder', decoder)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    captured = {}

    def trace(frame, kind, arg):
        if frame.f_code == module.decode.__code__ and kind == 'exception':
            local = frame.f_locals
            captured.update(plane=local['plane'], columns=local['columns'], rows=local['rows'],
                            position=local['position'], values=bytes(local['values']),
                            planes=[bytes(plane) for plane in local['planes']])
        return trace

    sys.settrace(trace)
    try:
        module.decode(payload)
    except IndexError:
        pass
    finally:
        sys.settrace(None)
    columns, rows = struct.unpack_from('<HH', payload)
    assert (columns, rows) == (80, 400)
    assert captured['plane'] == 3 and captured['position'] == len(payload)
    assert len(captured['values']) == columns * rows - 1 and len(captured['planes']) == 3
    known = captured['planes']
    assert [sha(plane) for plane in known] == [plane['sha256'] for plane in event['planes'][:3]]
    prefix = captured['values']
    # 解碼器先以 column-major 寫入，載入收據的 SHA 則按 RAM 的 row-major 計算。
    row_order = bytearray(columns * rows)
    for index, value in enumerate(prefix):
        x, y = divmod(index, rows)
        row_order[y * columns + x] = value
    expected = event['planes'][3]['sha256']
    matches = []
    for value in range(256):
        row_order[-1] = value
        if sha(row_order) == expected:
            matches.append(value)
    assert len(matches) == 1, f'unknown byte candidates: {matches}'
    row_order[-1] = matches[0]
    planes = known + [bytes(row_order)]
    assert [sha(plane) for plane in planes] == [plane['sha256'] for plane in event['planes']]
    indexed = bytearray(columns * 8 * rows)
    for number, plane in enumerate(planes):
        for index, value in enumerate(plane):
            for bit in range(8):
                indexed[index * 8 + bit] |= ((value >> (7 - bit)) & 1) << number
    output.mkdir()
    (output / 'OP-FULL-runtime.planes.bin').write_bytes(b''.join(planes))
    (output / 'OP-FULL-runtime.idx').write_bytes(indexed)
    (output / 'probe-source.txt').write_bytes(Path(__file__).read_bytes())
    result = {
        'classification': 'confirmed RAM value in one ordinary history-menu execution; original payload unchanged',
        'address_space': 'DAT file offsets and row-major plane-relative offsets; original guest bases retained in load event',
        'source': 'OP-FULL.OZM', 'exe_sha256': sha((project / 'original/game/DX5.EXE').read_bytes()),
        'dat_sha256': sha(dat), 'payload_sha256': sha(payload), 'decoder_sha256': sha(decoder.read_bytes()),
        'trace_tool_sha256': sha(trace_tool.read_bytes()), 'tool_sha256': sha(Path(__file__).read_bytes()),
        'receipt_sha256': sha(receipt_path.read_bytes()), 'normal_load_event': event,
        'known_prefix_column_order_sha256': sha(prefix), 'unknown_plane': 3, 'unknown_plane_offset': 31999,
        'all_256_candidates_evaluated': True, 'matching_candidates': matches,
        'plane_sha256': [sha(plane) for plane in planes], 'indexed_sha256': sha(indexed),
        'python': sys.version.split()[0],
        'limitations': ['one original execution only; undefined-tail stability across other histories unverified',
                        'not a recovered on-disk literal, patched source, HD artwork or normal HD drawing receipt'],
    }
    (output / 'receipt.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'matched_byte': matches[0], 'all_four_planes_match': True,
                      'indexed_sha256': sha(indexed), 'receipt_sha256': sha((output / 'receipt.json').read_bytes())}))


if __name__ == '__main__':
    main()
