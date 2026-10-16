"""核對八次 MAP_WAKU 下方快取圖塊；保留原始非一致平面偏移。"""
import hashlib
import json
import os
import re
from pathlib import Path

assert Path('/.dockerenv').is_file() and os.getuid() == 1000
project, output = Path('/project'), Path('/output')
assert output.stat().st_uid == 1000
sha = lambda b: hashlib.sha256(b).hexdigest()
catalog = project / 'workplace/hd-prototype/catalog'
reference = catalog / 'remaining/consumers/battle/r22/native-control'
ram = (reference / 'startup-memory.bin').read_bytes()
assert sha(ram) == '28563d71227d03bbdd87f01eca9c10cf639bede783ddb2b0e6ef72f35533954d'
exe = project / 'original/game/DX5.EXE'
assert sha(exe.read_bytes()) == '2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a'
source = (catalog / 'decoded/MAP_WAKU.OZM.idx').read_bytes()
assert len(source) == 384*120 and sha(source) == '1d0ee5b9b52142beb6813ae52f1db215e5224fae1935a64ea83fe07b97fe9a6b'
base = int.from_bytes(ram[0x42188:0x4218c], 'little')
assert base >= 0xc0000 and base+0x4a00 <= len(ram)

def crop(x, y, w, h):
    return b''.join(source[(y+row)*384+x:(y+row)*384+x+w] for row in range(h))

def packed(indices, bit):
    result = bytearray(len(indices)//8)
    for i, value in enumerate(indices):
        result[i//8] |= ((value >> bit) & 1) << (7-i%8)
    return bytes(result)

def decode(planes):
    return bytes(sum(((plane[i//8] >> (7-i%8)) & 1) << bit
                     for bit, plane in enumerate(planes)) for i in range(len(planes[0])*8))

main = crop(0, 0, 384, 88)
prefix = b''.join(packed(main, bit) for bit in range(4))
assert ram[base:base+0x4200] == prefix
tiles = [crop(i*32, 88, 32, 32) for i in range(4)]
lower = b''.join(packed(tile, bit) for tile in tiles for bit in range(4))
assert len(lower) == 0x800 and ram[base+0x4200:base+0x4a00] == lower

ida = catalog / 'remaining/consumers/ida'
proofs, caller = [], None
for name in ['range-1ec31-1f050.json', 'range-1fff7-20090.json']:
    path = ida / name
    data = json.loads(path.read_text())
    assert data['input_sha256'] == sha(exe.read_bytes()) and data['version'].startswith('9.4')
    for row in data['rows']:
        address, raw = int(row['ida_linear_ea'], 16), bytes.fromhex(row['bytes'])
        assert ram[address:address+len(raw)] == raw, (name, row['ida_linear_ea'])
    proofs.append(dict(path=str(path.relative_to(project)), sha256=sha(path.read_bytes()), rows=len(data['rows'])))
    if name == 'range-1ec31-1f050.json':
        caller = data['rows']

destination, offsets, calls, records = None, [], [], []
for row in caller:
    listing = row['listing']
    match = re.fullmatch(r'mov\s+\[ebp\+var_4\], ([0-9A-F]+)h', listing)
    if match:
        assert not offsets and not calls
        destination = int(match[1], 16)
    match = re.fullmatch(r'add\s+eax, ([0-9A-F]+)h', listing)
    if match:
        offsets.append(int(match[1], 16))
    if listing == 'call    sub_20020':
        calls.append(int(row['ida_linear_ea'], 16))
        assert len(calls) == len(offsets)
        if len(calls) == 4:
            tile = (offsets[0]-0x4200)//0x200
            assert tile in range(4) and offsets[0] == 0x4200+tile*0x200
            actual = decode([ram[base+n:base+n+128] for n in offsets])
            expected = tiles[tile]
            if tile == 0:
                assert offsets == [0x4200, 0x4080, 0x4300, 0x4380]
                independent = bytes((v & ~2) | (((source[80*384+i] >> 3) & 1) << 1)
                                    for i, v in enumerate(expected))
                assert actual == independent
            else:
                assert offsets == [0x4200+tile*0x200+bit*128 for bit in range(4)]
                assert actual == expected
            linear = destination-0xa0000
            x, y = linear % 80*8, linear//80
            assert 0 <= x <= 608 and 0 <= y <= 368
            records.append(dict(source_tile=tile, source_rect=[tile*32,88,tile*32+32,120],
                                destination_rect=[x,y,x+32,y+32], destination_linear=destination,
                                helper_linear=0x20020, callsites=calls, fourth_plane_return=calls[-1]+5,
                                raw_plane_offsets=offsets, static_source_crop_sha256=sha(expected),
                                actual_reader_indices_sha256=sha(actual),
                                different_static_indices=sum(a != b for a,b in zip(expected, actual))))
            path = output / f'map-cache-tile-{tile}.idx'
            if path.exists():
                assert path.read_bytes() == actual
            else:
                path.write_bytes(actual)
            offsets, calls = [], []
assert len(records) == 8 and not calls and not offsets
assert [r['source_tile'] for r in records] == [0,0,0,2,1,1,1,3]
result = dict(result='PASS', scope='cached sources and original reader bytes; natural draw and HD GUI pending',
              input_file='DX5.EXE', input_sha256=sha(exe.read_bytes()),
              source='MAP_WAKU.OZM', source_indices_sha256=sha(source),
              reference_receipt_sha256=sha((reference/'receipt.json').read_bytes()),
              reference_ram_sha256=sha(ram), cache_base=base, cache_pointer_global_linear=0x42188,
              cached_lower_sha256=sha(lower), cached_prefix_sha256=sha(prefix),
              address_space='IDA 9.4 LE loaded linear EA and dosgolem flat linear RAM; raw file offsets remain in IDA exports',
              source_checks=proofs, records=records, tool_sha256=sha(Path(__file__).read_bytes()))
(output/'map-cache-source-verification.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print(json.dumps(result,ensure_ascii=False,indent=2))
