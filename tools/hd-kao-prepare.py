"""將已檢視的九張頭像圖集切成 81 張獨立素材，不改變人物比例。"""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw


# 圖集生成後的實際格線。第 4 頁及第 8 頁的橫線依欄不同。
GRIDS = [
    ([0, 418, 836, 1254], [[0, 418, 834, 1254]] * 3),
    ([0, 411, 819, 1254], [[0, 397, 789, 1254]] * 3),
    ([0, 418, 837, 1254], [[0, 417, 825, 1254]] * 3),
    ([0, 418, 836, 1254], [[0, 418, 836, 1254]] * 3),
    ([0, 418, 836, 1254], [[0, 365, 783, 1254], [0, 377, 783, 1254], [0, 389, 783, 1254]]),
    ([0, 418, 836, 1254], [[0, 418, 836, 1254]] * 3),
    ([0, 418, 836, 1254], [[0, 418, 836, 1254]] * 3),
    ([0, 419, 838, 1254], [[0, 403, 815, 1254]] * 3),
    ([0, 418, 836, 1254], [[0, 413, 831, 1254], [0, 415, 833, 1254], [0, 415, 833, 1254]]),
]


def sha(data):
    return hashlib.sha256(data).hexdigest()


def save_unchanged(path, data):
    if path.exists():
        assert path.read_bytes() == data, f"refuse changed output: {path}"
    else:
        path.write_bytes(data)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path("/output"))
    args = parser.parse_args()
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000
    root = args.root
    assert root.is_dir() and root.stat().st_uid == os.getuid()
    output = root / "selected"
    output.mkdir(exist_ok=True)
    assert output.stat().st_uid == os.getuid()
    sources = json.loads((root / "kao-extraction.json").read_text())
    records = []
    sheet = Image.new("RGB", (9 * 144, 9 * 164), (24, 24, 24))
    pen = ImageDraw.Draw(sheet)
    for page, (xs, ys) in enumerate(GRIDS):
        path = root / f"page-{page:02d}-hd-b-v1.png"
        prompt = root / f"page-{page:02d}-hd-b-v1.prompt.txt"
        data = path.read_bytes()
        image = Image.open(io.BytesIO(data)).convert("RGBA")
        assert image.size == (1254, 1254) and image.getextrema()[3] == (255, 255)
        for row in range(3):
            for col in range(3):
                index = page * 9 + row * 3 + col
                rect = (xs[col], ys[col][row], xs[col+1], ys[col][row+1])
                crop = image.crop(rect).convert("RGB")
                edge = max(crop.size)
                selected = Image.new("RGB", (edge, edge), "black")
                pad = ((edge-crop.width)//2, (edge-crop.height)//2)
                selected.paste(crop, pad)
                buffer = io.BytesIO()
                selected.save(buffer, format="PNG")
                result = buffer.getvalue()
                name = f"kao-{index:02d}-hd-b-v1.png"
                save_unchanged(output / name, result)
                records.append({"index": index, "asset": name, "asset_sha256": sha(result),
                                "size": [edge, edge], "atlas": path.name, "atlas_sha256": sha(data),
                                "prompt": prompt.name, "prompt_sha256": sha(prompt.read_bytes()),
                                "crop_xyxy": rect, "black_padding_xy": pad,
                                "original": f"kao-{index:02d}-original.png",
                                "original_png_sha256": sha((root / f"kao-{index:02d}-original.png").read_bytes())})
                selected.thumbnail((140, 140), Image.Resampling.LANCZOS)
                sx, sy = (index % 9)*144, (index//9)*164
                sheet.paste(selected, (sx+(144-selected.width)//2, sy))
                pen.text((sx+4, sy+143), f"{index:02d}", fill="white")
    manifest = {"classification": "local HD art preparation; runtime coverage is verified separately",
                "style": "B 戰國手繪風", "count": len(records), "records": records,
                "extract_receipt_sha256": sha((root / "kao-extraction.json").read_bytes()),
                "prepare_tool_sha256": sha(Path(__file__).read_bytes()),
                "original_input_receipt": sources.get("classification", "kao-extraction.json")}
    save_unchanged(output / "manifest.json", (json.dumps(manifest, ensure_ascii=False, indent=2)+"\n").encode())
    buffer = io.BytesIO()
    sheet.save(buffer, format="PNG")
    save_unchanged(output / "contact-sheet.png", buffer.getvalue())
    print(f"prepared {len(records)} portraits; no rescaling before black padding")


if __name__ == "__main__":
    main()
