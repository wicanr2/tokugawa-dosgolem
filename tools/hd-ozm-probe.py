#!/usr/bin/env python3
"""離線 OZM 研究：只解已查證的獨立命令，拒絕 VGA 來源命令。"""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import struct


def sha(data):
    return hashlib.sha256(data).hexdigest()


def decode(payload):
    columns, rows = struct.unpack_from("<HH", payload)
    if not 1 <= columns <= 80 or not 1 <= rows <= 400:
        raise ValueError("unsupported dimensions")
    position = 4
    planes = []
    commands = Counter()
    for plane in range(4):
        values = bytearray()
        while len(values) < columns * rows:
            value = payload[position]
            position += 1
            if value:
                sequence, count = bytes([value]), 1
                commands["literal"] += 1
            else:
                command = payload[position]
                position += 1
                commands[str(command)] += 1
                if 1 <= command <= 4:
                    sequence, count = b"\x00", command
                elif command == 5:
                    sequence = payload[position:position + 1]
                    count = struct.unpack_from("<H", payload, position + 1)[0]
                    position += 3
                elif command == 6:
                    sequence = payload[position:position + 2]
                    count = struct.unpack_from("<H", payload, position + 2)[0]
                    position += 4
                elif 7 <= command <= 10:
                    raise ValueError(f"requires live VGA plane: command {command}, plane {plane}, offset {position-2}")
                else:
                    raise ValueError(f"unsupported command {command}")
                # x86 LOOP 的零值會回繞；寫滿該平面即由 CF 提早返回。
                if count == 0:
                    count = columns * rows - len(values)
            count = min(count, columns * rows - len(values))
            values.extend((sequence * ((count + len(sequence) - 1) // len(sequence)))[:count])
        packed = bytearray(columns * rows)
        for column in range(columns):
            for row in range(rows):
                packed[row * columns + column] = values[column * rows + row]
        planes.append(bytes(packed))
    indices = bytearray(columns * 8 * rows)
    for plane, packed in enumerate(planes):
        for row in range(rows):
            for x in range(columns * 8):
                indices[row * columns * 8 + x] |= ((packed[row * columns + x // 8] >> (7 - x % 8)) & 1) << plane
    return columns, rows, planes, bytes(indices), position, commands


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=Path("/output/decoded"))
    args = parser.parse_args()
    assert Path("/.dockerenv").is_file(), "Docker only"
    root = args.output
    assert root.parent.is_dir() and root.parent.stat().st_uid == 1000
    root.mkdir(exist_ok=True)
    assert root.stat().st_uid == 1000
    dat = Path("/original/DX5.DAT").read_bytes()
    assert sha(dat) == "de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37"
    inventory = json.loads(Path("/project/workplace/hd-prototype/asset-inventory.json").read_text())
    assert inventory["input_sha256"] == sha(dat) and inventory["record_count"] == 201
    captures = {}
    for path in root.parent.glob("hd-assets/*.json"):
        capture = json.loads(path.read_text())
        captures.setdefault(capture["event"]["name"], []).append(capture)
    records = []
    for item in inventory["records"]:
        if not item["name"].endswith(".OZM"):
            continue
        payload = dat[item["candidate_file_offset"]:item["candidate_file_offset"] + item["candidate_length"]]
        assert sha(payload) == item["payload_sha256"]
        result = {"name": item["name"], "dat_file_offset": item["candidate_file_offset"], "payload_bytes": len(payload), "payload_sha256": sha(payload), "semantic_level": "static decoder; normal-path matches recorded separately"}
        try:
            columns, rows, planes, indices, consumed, commands = decode(payload)
            hashes = [sha(p) for p in planes]
            result.update(status="decoded", width=columns * 8, height=rows, plane_sha256=hashes, indices_sha256=sha(indices), consumed_bytes=consumed, trailing_bytes=len(payload) - consumed, commands=dict(commands), normal_path_matches=0)
            for capture in captures.get(item["name"], []):
                assert hashes == [p["sha256"] for p in capture["event"]["planes"]], item["name"]
                assert result["indices_sha256"] == capture["indices_sha256"], item["name"]
                result["normal_path_matches"] += 1
            path = root / (item["name"] + ".idx")
            if path.exists():
                assert path.read_bytes() == indices, "refusing changed output"
            else:
                path.write_bytes(indices)
        except (ValueError, IndexError, struct.error) as exc:
            result.update(status="unsupported", reason=str(exc))
        records.append(result)
    summary = {"classification": "local-only original asset research; not HD completion", "dat_sha256": sha(dat), "tool_sha256": sha(Path(__file__).read_bytes()), "evidence": ["workplace/hd-prototype/range-28660-28860.json: IDA 9.4 linear 0x28689-0x28772", "workplace/hd-prototype/range-2adc8-2af40.json: IDA 9.4 linear 0x2ADC8-0x2AEE0"], "counts": dict(Counter(r["status"] for r in records)), "normal_path_sources_verified": sum(r.get("normal_path_matches", 0) > 0 for r in records), "records": records}
    (root / "inventory.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({k: summary[k] for k in ("counts", "normal_path_sources_verified")}, ensure_ascii=False))


if __name__ == "__main__":
    main()
