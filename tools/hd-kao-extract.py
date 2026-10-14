#!/usr/bin/env python3
"""在固定 Docker 工具鏈擷取 KAO.GRP 的 81 張原版頭像與來源收據。"""
import hashlib
import json
import os
from pathlib import Path
import struct

from PIL import Image, __version__ as pillow_version


def sha(data):
    return hashlib.sha256(data).hexdigest()


def write_same(path, data):
    if path.exists():
        assert path.read_bytes() == data, f"refusing changed original capture: {path}"
    else:
        path.write_bytes(data)


def save_same(path, image):
    import io
    out = io.BytesIO()
    image.save(out, format="PNG")
    write_same(path, out.getvalue())
    return sha(out.getvalue())


def main():
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000
    assert pillow_version == "9.4.0"
    root = Path("/output/kao")
    assert root.parent.is_dir() and root.parent.stat().st_uid == 1000
    root.mkdir(exist_ok=True)
    assert root.stat().st_uid == 1000
    dat = Path("/original/DX5.DAT").read_bytes()
    assert sha(dat) == "de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37"
    inv = json.loads(Path("/project/workplace/hd-prototype/asset-inventory.json").read_text())
    entry = next(r for r in inv["records"] if r["name"] == "KAO.GRP")
    source = dat[entry["candidate_file_offset"]:entry["candidate_file_offset"] + entry["candidate_length"]]
    assert len(source) == 81 * 3200 and sha(source) == entry["payload_sha256"]
    ram = Path("/project/workplace/desktop-prototype/headless/startup-memory.bin").read_bytes()
    assert sha(ram) == "4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9"
    base = struct.unpack_from("<I", ram, 0x39888)[0]
    assert base >= 0xC0000 and ram[base:base + len(source)] == source
    capture_paths = [p for p in root.parent.glob("hd-assets/*.json") if json.loads(p.read_text())["event"]["name"] == "A-0.OZM"]
    assert len(capture_paths) == 1
    capture = json.loads(capture_paths[0].read_text())
    palette = [v for color in capture["palette_at_loader_return"] for v in (color["R"], color["G"], color["B"])]
    proof_path = root / "native-boundary-matches.json"
    proof = json.loads(proof_path.read_text())
    assert proof["source_index"] == 76 and proof["pixels"] == 6400
    assert any(r["step"] == 105000000 and r["rect"] == [96, 86, 80, 80] for r in proof["matches"])
    images, records = [], []
    for index in range(81):
        packed = source[index*3200:(index+1)*3200]
        pixels = bytearray(80 * 80)
        for y in range(80):
            for x in range(80):
                for plane in range(4):
                    pixels[y*80+x] |= ((packed[plane*800+y*10+x//8] >> (7-x%8)) & 1) << plane
        image = Image.frombytes("P", (80, 80), bytes(pixels))
        image.putpalette(palette)
        image = image.convert("RGB")
        name = f"kao-{index:02d}-original.png"
        digest = save_same(root / name, image)
        write_same(root / f"kao-{index:02d}.idx", bytes(pixels))
        records.append({"index": index, "source_relative_offset": index*3200, "source_bytes": 3200,
                        "raw_planar_sha256": sha(packed), "indices_sha256": sha(pixels),
                        "source_png": name, "source_png_sha256": digest,
                        "native_pixels_verified": 6400 if index == 76 else 0})
        images.append(image)
    for page in range(9):
        image = Image.new("RGB", (240, 240))
        for cell, source_image in enumerate(images[page*9:page*9+9]):
            image.paste(source_image, ((cell%3)*80, (cell//3)*80))
        save_same(root / f"page-{page:02d}-original.png", image)
    evidence = {}
    for name in ("art-consumers.json", "art-consumers-39888.json", "range-1c630-1c6e8.json", "range-1c6e8-1c800.json"):
        p = root.parent / name
        data = json.loads(p.read_text())
        assert data["version"] == "9.4" and data["input_sha256"] == "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a"
        evidence[name] = sha(p.read_bytes())
    summary = {"classification": "original-derived local research; native case 76 verified, other indices decoded from confirmed common format",
               "dat_sha256": sha(dat), "resource_name": "KAO.GRP", "resource_file_offset": entry["candidate_file_offset"],
               "resource_sha256": sha(source), "ram_sha256": sha(ram), "backing_pointer_global_linear": "0x39888",
               "backing_base_linear": base, "source_dimensions": [80, 80], "count": 81, "plane_bytes": 800,
               "tool": "Python + Pillow", "pillow_version": pillow_version, "tool_sha256": sha(Path(__file__).read_bytes()),
               "palette_capture_sha256": sha(capture_paths[0].read_bytes()), "native_proof_sha256": sha(proof_path.read_bytes()),
               "ida_address_space": "IDA LE loader linear EA; file offsets in referenced exports", "ida_evidence_sha256": evidence,
               "records": records}
    (root / "kao-extraction.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n")
    print("KAO.GRP: 81 portraits; loaded bytes verified; native portrait 76 matches 6400 pixels")


if __name__ == "__main__":
    main()
