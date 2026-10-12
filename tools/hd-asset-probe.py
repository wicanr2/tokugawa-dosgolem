"""DAT 資源目錄的有界唯讀研究探針；候選格式不是正式素材包契約。"""

import collections
import hashlib
import json
import os
import pathlib
import struct
import sys


assert pathlib.Path("/.dockerenv").exists(), "請在 Docker 容器內執行"
source = pathlib.Path("/original/DX5.DAT")
data = source.read_bytes()
digest = hashlib.sha256(data).hexdigest()
assert digest == "de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37"
assert data[:8] == b"Ken Lin\0"
count = struct.unpack_from("<I", data, 8)[0]
assert count == 201
end = 12 + count * 21
rows = []
names = set()
extensions = collections.Counter()
for i in range(count):
    at = 12 + i * 21
    raw = data[at:at + 21]
    assert len(raw) == 21 and b"\0" in raw[:13]
    name = raw[:13].split(b"\0", 1)[0].decode("ascii")
    assert name and name not in names and "/" not in name and "\\" not in name
    names.add(name)
    length, offset = struct.unpack_from("<II", raw, 13)
    assert offset == end and offset + length <= len(data)
    end = offset + length
    extensions[name.rsplit(".", 1)[-1]] += 1
    rows.append({"index": i, "record_file_offset": at, "raw_record_hex": raw.hex(),
                 "name": name, "candidate_length": length, "candidate_file_offset": offset,
                 "payload_sha256": hashlib.sha256(data[offset:end]).hexdigest(),
                 "payload_prefix_hex": data[offset:min(end, offset + 32)].hex()})
assert end == len(data)
result = {
    "input_file": source.name, "input_sha256": digest, "input_bytes": len(data),
    "tool": "Python", "tool_version": sys.version.split()[0],
    "probe_sha256": hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),
    "address_space": "DX5.DAT file byte offsets, decimal",
    "semantic_level": "strong inference: contiguous bounds and names; loader consumer still required",
    "production_ready": False, "record_count": count, "candidate_record_bytes": 21,
    "candidate_name_bytes": 13, "candidate_table_end": 12 + count * 21,
    "extension_counts": dict(sorted(extensions.items())), "records": rows,
}
output = pathlib.Path("/output/asset-inventory.json")
assert output.parent.stat().st_uid == os.getuid()
if output.exists():
    assert output.stat().st_uid == os.getuid()
output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
print(json.dumps({k: v for k, v in result.items() if k != "records"}, ensure_ascii=False, indent=2))
