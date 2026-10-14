"""核對 WINDOW.OZM 的九個 16x16 圖塊、原版 buffer 與正常視窗。"""
import hashlib
import io
import json
import os
from pathlib import Path
import struct
import sys

from PIL import Image, __version__ as pillow_version


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save_same(path, data):
    if path.exists():
        assert path.read_bytes() == data, f"refuse changed original output: {path}"
    else:
        path.write_bytes(data)


def save_image(path, image):
    buf = io.BytesIO()
    image.save(buf, format="PNG")
    save_same(path, buf.getvalue())


def main():
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000 and pillow_version == "9.4.0"
    project = Path("/project")
    catalog = project / "workplace/hd-prototype/catalog"
    root = Path("/output")
    assert root.stat().st_uid == 1000
    output = root / "window"
    output.mkdir(exist_ok=True)
    assert output.stat().st_uid == 1000
    ram_path = project / "workplace/desktop-prototype/headless/startup-memory.bin"
    assert sha(ram_path) == "4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9"
    ram = ram_path.read_bytes()
    base = struct.unpack_from("<I", ram, 0x35153)[0]
    assert 0xc0000 <= base and base+1152 <= len(ram)
    raw = ram[base:base+1152]
    capture_path = next((catalog / "hd-assets").glob("WINDOW*.json"))
    capture = json.loads(capture_path.read_text())["event"]
    assert capture["name"] == "WINDOW.OZM" and capture["raw_columns"] == 18 and capture["raw_rows"] == 16
    for plane in range(4):
        assert hashlib.sha256(raw[plane*288:(plane+1)*288]).hexdigest() == capture["planes"][plane]["sha256"]
    source_path = catalog / "previews/WINDOW.OZM.png"
    source = Image.open(source_path).convert("RGB")
    assert source.size == (144, 16)
    native_path = project / "workplace/desktop-prototype/headless/startup-probe/frame-fd57c8d1887b4398f89a37b715d83799876cd7af5eabac30b06f14be4e456f84.png"
    assert sha(native_path) == "fd57c8d1887b4398f89a37b715d83799876cd7af5eabac30b06f14be4e456f84"
    native = Image.open(native_path).convert("RGB")
    checks = []
    for y in range(3):
        for x in range(8):
            index = (0 if y == 0 else 2 if y == 2 else 1)*3 + (0 if x == 0 else 2 if x == 7 else 1)
            if index == 4:
                continue
            rect = [80+x*16, 20+y*16, 96+x*16, 36+y*16]
            assert source.crop((index*16, 0, index*16+16, 16)).tobytes() == native.crop(rect).tobytes()
            checks.append({"tile": index, "native_xyxy": rect})
    center_rect = [176, 36, 192, 52]
    assert source.crop((64, 0, 80, 16)).tobytes() == native.crop(center_rect).tobytes()
    checks.append({"tile": 4, "native_xyxy": center_rect})
    assert {c["tile"] for c in checks} == set(range(9))
    save_same(output / "WINDOW-original.png", source_path.read_bytes())
    save_same(output / "WINDOW-packed.bin", raw)
    skin = Image.new("RGB", (48, 48))
    for index in range(9):
        tile = source.crop((index*16, 0, index*16+16, 16))
        save_image(output / f"tile-{index}-original.png", tile)
        skin.paste(tile, ((index % 3)*16, (index//3)*16))
    save_image(output / "window-skin-original.png", skin)
    save_image(output / "window-skin-reference.png", skin.resize((768, 768), Image.Resampling.NEAREST))
    evidence = {}
    for name in ("range-1af00-1b165.json", "range-1b165-1b300.json"):
        p = catalog / name
        d = json.loads(p.read_text())
        assert d["version"] == "9.4" and d["input_sha256"] == "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a"
        evidence[name] = sha(p)
    receipt = {"classification": "original UI source and native pixels verified; live draw observation pending",
               "source_resource": "WINDOW.OZM", "source_png_sha256": sha(source_path),
               "ram_sha256": sha(ram_path), "buffer_pointer_global_linear": "0x35153", "buffer_base_linear": base,
               "packed_sha256": hashlib.sha256(raw).hexdigest(), "plane_bytes": 288, "source_pitch_bytes": 18,
               "tile_count": 9, "tile_size": [16, 16], "native_png_sha256": sha(native_path),
               "native_sampled_pixels": len(checks)*256, "distinct_source_pixels_verified": 144*16,
               "native_checks": checks, "ida_address_space": "IDA LE loader linear EA; raw file offsets in exports",
               "ida_evidence_sha256": evidence, "source_capture_sha256": sha(capture_path),
               "python_version": sys.version, "pillow_version": pillow_version, "tool_sha256": sha(Path(__file__))}
    (output / "window-extraction.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+"\n")
    print("WINDOW: all 9 tiles, 2304 distinct source pixels, 4864 native sample pixels verified")


if __name__ == "__main__":
    main()
