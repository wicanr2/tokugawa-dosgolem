"""將既有手繪底圖分區縮放至原版上方框架；不決定遊戲座標。"""
import hashlib
import io
import json
import os
from pathlib import Path

from PIL import Image, __version__ as pillow_version


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save_same(path, image):
    buf = io.BytesIO()
    image.save(buf, format="PNG")
    data = buf.getvalue()
    if path.exists():
        assert path.read_bytes() == data, f"refuse changed output: {path}"
    else:
        path.write_bytes(data)
    return sha(path)


def main():
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000 and pillow_version == "9.4.0"
    source_root = Path("/project/workplace/hd-prototype/scroll")
    original_path = source_root / "VISUAL-original.png"
    full_path = source_root / "scroll-full-hd-b-v1.png"
    prompt = source_root / "scroll-hd-b-v1.prompt.txt"
    assert sha(original_path) == "cd8d249121f87e9a86fa5d4aa7e51609a68ae3252864be46dcb9142cb3f1542b"
    assert sha(full_path) == "b2abace38962a2aa957d5b15361929c3ee71bbc5728786a7a95c2830dc2d58c9"
    root = Path("/output")
    assert root.stat().st_uid == 1000
    output = root / "frame"
    output.mkdir(exist_ok=True)
    assert output.stat().st_uid == 1000
    original = Image.open(original_path).convert("RGB")
    full = Image.open(full_path).convert("RGBA")
    assert original.size == (640, 400) and full.size == (1586, 992)
    assert full.getextrema()[3] == (255, 255)
    source_sha = save_same(output / "frame-original.png", original.crop((0, 0, 640, 250)))
    # 目的矩形來自原版 VISUAL 與已證實的劇情圖矩形。
    # 來源切線只用來處理生成圖的紙張位置偏差。
    sx, sy = [0, 396, 1189, 1586], [0, 123, 495, 620]
    dx, dy = [0, 320, 960, 1280], [0, 100, 400, 500]
    result = Image.new("RGB", (1280, 500))
    for y in range(3):
        for x in range(3):
            part = full.crop((sx[x], sy[y], sx[x+1], sy[y+1])).convert("RGB")
            part = part.resize((dx[x+1]-dx[x], dy[y+1]-dy[y]), Image.Resampling.LANCZOS)
            result.paste(part, (dx[x], dy[y]))
    art_sha = save_same(output / "frame-hd-b-v2.png", result)
    receipt = {"classification": "local HD frame preparation; runtime validation pending",
               "source_original_sha256": sha(original_path), "source_generated_sha256": sha(full_path),
               "generator": "built-in image_gen.imagegen", "prompt_sha256": sha(prompt),
               "source_crop_xyxy": [0, 0, 640, 250], "frame_source_png_sha256": source_sha,
               "frame_hd_png_sha256": art_sha, "source_grid_x": sx, "source_grid_y": sy,
               "destination_grid_x": dx, "destination_grid_y": dy,
               "preserve_original_logical_rect": [160, 50, 320, 150],
               "tool_sha256": sha(Path(__file__)), "pillow_version": pillow_version}
    (output / "frame-art-v2.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(receipt, ensure_ascii=False))


if __name__ == "__main__":
    main()
