"""核對 MAP_CHIP 原版緩衝並細分研究圖塊；不定義正式素材包或遊戲索引集合。"""
import hashlib
import io
import json
import os
import struct
import sys
from pathlib import Path

from PIL import Image, ImageDraw, __version__ as pillow_version


def sha(data):
    return hashlib.sha256(data).hexdigest()


def write_checked(path, data):
    assert path.parent.stat().st_uid == 1000
    if path.exists():
        assert path.stat().st_uid == 1000 and path.read_bytes() == data, path
    else:
        path.write_bytes(data)


def png_bytes(image):
    buffer = io.BytesIO()
    image.save(buffer, format="PNG")
    return buffer.getvalue()


def main():
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000 and pillow_version == "9.4.0"
    project = Path("/project")
    catalog = project/"workplace/hd-prototype/catalog"
    output = Path("/output")
    assert output.stat().st_uid == 1000
    root = output/"terrain"
    root.mkdir(exist_ok=True)
    assert root.stat().st_uid == 1000
    exe_sha = "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a"
    assert sha(Path("/original/DX5.EXE").read_bytes()) == exe_sha
    dat = Path("/original/DX5.DAT").read_bytes()
    assert sha(dat) == "de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37"
    inventory = json.loads((catalog/"decoded/inventory.json").read_text())
    assert inventory["dat_sha256"] == sha(dat)
    assert inventory["tool_sha256"] == sha((project/"tools/hd-ozm-probe.py").read_bytes())
    source = next(r for r in inventory["records"] if r["name"] == "MAP_CHIP.OZM")
    assert source["status"] == "decoded" and (source["width"], source["height"]) == (640, 400)
    payload = dat[source["dat_file_offset"]:source["dat_file_offset"]+source["payload_bytes"]]
    assert sha(payload) == source["payload_sha256"]
    indices = (catalog/"decoded/MAP_CHIP.OZM.idx").read_bytes()
    assert len(indices) == 640*400 and sha(indices) == source["indices_sha256"]
    captures = list((catalog/"hd-assets").glob("MAP_CHIP*.json"))
    assert len(captures) == 1
    capture_path = captures[0]
    capture = json.loads(capture_path.read_text())
    assert capture["indices_sha256"] == source["indices_sha256"]
    palette = [tuple(c[k] for k in ("R", "G", "B")) for c in capture["palette_at_loader_return"]]
    rgb = bytes(c for index in indices for c in palette[index])
    atlas = Image.frombytes("RGB", (640, 400), rgb)
    assert sha(capture_path.with_suffix(".png").read_bytes()) == capture["png_sha256"]
    assert Image.open(capture_path.with_suffix(".png")).convert("RGB").tobytes() == rgb
    ram_path = catalog/"story/desktop/startup-memory.bin"
    ram = ram_path.read_bytes()
    assert sha(ram) == "4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9"
    plane_receipts = []
    for bit, address in enumerate((0x42174, 0x42168, 0x42170, 0x4216C)):
        base = struct.unpack_from("<I", ram, address)[0]
        assert base+32000 <= len(ram)
        plane = bytearray(32000)
        for n, index in enumerate(indices):
            if index & (1 << bit):
                plane[n//8] |= 1 << (7-n%8)
        digest = sha(plane)
        assert digest == source["plane_sha256"][bit] == capture["event"]["planes"][bit]["sha256"]
        assert ram[base:base+32000] == plane
        plane_receipts.append({"pointer_global_linear": address, "base_linear": base, "sha256": digest})
    evidence = {}
    for name in ("range-1fc00-20040.json", "range-1f8d2-1fc07.json",
                 "art-consumers-35988-3598c-35990-42168-4216c-42170-42174.json", "asset-xrefs-resources.json"):
        path = catalog/name
        record = json.loads(path.read_text())
        assert record["input_sha256"] == exe_sha and record["tool"] == "IDA Pro" and record["version"] == "9.4"
        evidence[name] = sha(path.read_bytes())
    records = []
    groups = (("floor", 0x35988, 0, 32, 8), ("object", 0x3598C, 256, 48, 2), ("unit", 0x35990, 352, 48, 1))
    for group, address, first_y, height, rows in groups:
        assert struct.unpack_from("<I", ram, address)[0] == first_y*80
        folder = root/group
        folder.mkdir(exist_ok=True)
        assert folder.stat().st_uid == 1000
        sheet = Image.new("RGB", (13*108, rows*(height*2+24)), "#ded6c2")
        draw = ImageDraw.Draw(sheet)
        for index in range(13*rows):
            x, y = index%13*48, first_y+index//13*height
            rect = (x, y, x+48, y+height)
            tile_indices = b"".join(indices[row*640+x:row*640+x+48] for row in range(y, y+height))
            image = atlas.crop(rect).convert("RGBA")
            # sub_1F9D0 對四平面 OR 後取反作遮罩，只有索引 0 透明。
            if group != "floor":
                image.putalpha(Image.frombytes("L", (48, height), bytes(255 if i else 0 for i in tile_indices)))
            name = f"{group}-{index:03d}-original.png"
            data = png_bytes(image)
            write_checked(folder/name, data)
            dx, dy = index%13*108, index//13*(height*2+24)
            sheet.paste(image.resize((96, height*2), Image.Resampling.NEAREST), (dx, dy+20),
                        image.getchannel("A").resize((96, height*2), Image.Resampling.NEAREST))
            draw.text((dx, dy), f"{group} {index:03d}", fill="black")
            records.append({"group": group, "source_grid_index": index, "source_rect": list(rect),
                            "indices_sha256": sha(tile_indices), "png": group+"/"+name, "png_sha256": sha(data),
                            "nonzero_source_pixels": sum(i != 0 for i in tile_indices),
                            "alpha_rule": "opaque" if group == "floor" else "source index zero transparent"})
        write_checked(root/(group+"-contact-sheet.png"), png_bytes(sheet))
    result = {"classification": "local original atlas subdivision; not generated HD art or a production pack schema",
              "semantic_level": "source planes and copy geometry confirmed; game-used index set unknown",
              "source": "MAP_CHIP.OZM", "exe_sha256": exe_sha, "dat_sha256": sha(dat),
              "payload_sha256": source["payload_sha256"], "indices_sha256": sha(indices),
              "original_capture_sha256": sha(capture_path.read_bytes()), "ram_sha256": sha(ram),
              "ram_source": str(ram_path.relative_to(project)), "planes": plane_receipts, "ida_evidence": evidence,
              "python_version": sys.version, "pillow_version": pillow_version,
              "ida_image": "sha256:4ac62de83339c215bab10e455cee3a22d9c6efed9fd0d8ed0f068327b83a06ab",
              "tool_sha256": sha(Path(__file__).read_bytes()),
              "counts": {group: rows*13 for group, _, _, _, rows in groups}, "unused_right_columns": [624, 640],
              "records": records}
    target = root/"terrain-extraction.json"
    data = (json.dumps(result, ensure_ascii=False, indent=2)+"\n").encode()
    if target.exists():
        assert target.stat().st_uid == 1000
        previous = target.read_bytes()
        if previous != data:
            write_checked(root/("terrain-extraction-"+sha(previous)+".json"), previous)
    target.write_bytes(data)
    print(json.dumps({"verified": True, "source_cells": len(records), "counts": result["counts"]}))


if __name__ == "__main__":
    main()
