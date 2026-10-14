"""依使用者授權，保存生成的 HD 圖與提示詞至 private repo。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, __version__ as pillow_version


README = """# 戰國手繪 HD 素材

使用者於 2026-10-14 授權將目前生成的圖片保存於本私人儲存庫。這裡保存生成圖、由生成圖裁切的素材與完整提示詞，作為開發來源。公開發行與素材散布權另行確認。

- [圖片與提示詞索引](index.json) 記錄 SHA-256、尺寸、來源位置、提示詞與裁切依據。
- [劇情圖](art/) 及 [家康肖像](A-0-hd-b-v1.png)。
- [81 張頭像總覽](kao/selected/contact-sheet.png)、[九頁頭像圖集](kao/)。
- [捲軸與紅金框架](scroll/)、[共用視窗邊框](window/)。
- [風格比較](concept/style-comparison-v1.png)。

索引中的狀態區分已驗證、已製作與草稿。圖片存在不代表遊戲內整合完成。正式素材包格式及全遊戲覆蓋以專案 CONTEXT、WORKLIST 與 RESEARCH-LOG 為準。

保存工具為專案 tools/archive-hd-art.py，在固定 GUI Docker image 中執行。原版資料與擷取原圖仍使用本機 original/、workplace/；本目錄沒有遊戲資料包。
"""


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--verify-only", action="store_true")
    args = parser.parse_args()
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000 and pillow_version == "9.4.0"
    project = Path("/project")
    output = project / "assets/hd-b"
    if args.verify_only:
        index = json.loads((output / "index.json").read_text())
        assert index["archive_tool_sha256"] == sha(Path(__file__))
        assert (output / "README.md").read_text() == README
        expected = {"index.json", "README.md"}
        for record in index["files"]:
            assert not Path(record["path"]).is_absolute() and ".." not in Path(record["path"]).parts
            path = output / record["path"]
            assert sha(path) == record["sha256"] and path.stat().st_size == record["bytes"], path
            expected.add(record["path"])
        assert {str(p.relative_to(output)) for p in output.rglob("*") if p.is_file()} == expected
        assert index["counts"] == {"png": sum(r["kind"] == "png" for r in index["files"]),
                                   "prompts": sum(r["kind"] == "prompt" for r in index["files"]),
                                   "bytes": sum(r["bytes"] for r in index["files"])}
        print(json.dumps(index["counts"], ensure_ascii=False))
        return
    assert project.stat().st_uid == 1000
    output.mkdir(parents=True, exist_ok=True)
    assert output.stat().st_uid == 1000
    base = project / "workplace/hd-prototype"
    catalog = base / "catalog"
    files = {}

    def copy(source, name, kind, **metadata):
        assert source.is_file() and source.stat().st_uid == 1000, source
        relative = Path(name)
        assert not relative.is_absolute() and ".." not in relative.parts
        assert source.suffix in (".png", ".txt", ".json") and "-original" not in source.name
        destination = output / relative
        destination.parent.mkdir(parents=True, exist_ok=True)
        assert destination.parent.stat().st_uid == 1000
        data = source.read_bytes()
        if destination.exists():
            assert destination.stat().st_uid == 1000 and destination.read_bytes() == data, destination
        else:
            destination.write_bytes(data)
        record = {"path": name, "kind": kind, "bytes": len(data), "sha256": sha(destination),
                  "local_source": str(source.relative_to(project)), **metadata}
        if kind == "png":
            assert len(data) < 50*1024*1024
            with Image.open(destination) as im:
                assert im.format == "PNG" and im.convert("RGBA").getextrema()[3] == (255, 255)
                record["size"] = list(im.size)
        if name in files:
            assert files[name] == record
        files[name] = record

    def picture(source, name, prompt, prompt_name, status, **metadata):
        copy(prompt, prompt_name, "prompt")
        copy(source, name, "png", prompt=prompt_name, status=status, **metadata)

    picture(base/"style-comparison-v1.png", "concept/style-comparison-v1.png", base/"style-comparison-v1.prompt.txt", "concept/style-comparison-v1.prompt.txt", "style comparison; B selected")
    picture(base/"A-0-hd-b-v1.png", "A-0-hd-b-v1.png", base/"A-0-hd-b-v1.prompt.txt", "A-0-hd-b-v1.prompt.txt", "normal player path verified")
    for source in sorted((catalog/"art").glob("*-hd-b-v*.png")):
        name = source.name
        status = "normal player path verified" if name in {f"A-{i}-hd-b-v1.png" for i in range(1, 8)} else "draft; integration and review pending"
        picture(source, "art/"+name, source.with_suffix(".prompt.txt"), "art/"+source.with_suffix(".prompt.txt").name, status)
    for page in range(9):
        name = f"page-{page:02d}-hd-b-v1.png"
        picture(catalog/"kao"/name, "kao/"+name, (catalog/"kao"/name).with_suffix(".prompt.txt"), "kao/"+Path(name).with_suffix(".prompt.txt").name, "generated portrait atlas")
    selected = json.loads((catalog/"kao/selected/manifest.json").read_text())
    assert len(selected["records"]) == 81
    for record in selected["records"]:
        name = record["asset"]
        source = catalog/"kao/selected"/name
        assert sha(source) == record["asset_sha256"]
        copy(source, "kao/selected/"+name, "png", prompt="kao/"+record["prompt"],
             status="normal player path verified" if record["index"] in range(76, 80) else "common draw consumer integrated; native case not sampled",
             derivation=record)
    copy(catalog/"kao/selected/contact-sheet.png", "kao/selected/contact-sheet.png", "png", status="generated HD portrait overview")
    copy(catalog/"kao/selected/manifest.json", "kao/selected/manifest.json", "preparation-metadata")
    prompt = base/"scroll/scroll-hd-b-v1.prompt.txt"
    prompt_name = "scroll/scroll-hd-b-v1.prompt.txt"
    for name in ("scroll-full-hd-b-v1.png", "scroll-hd-b-v1.png"):
        picture(base/"scroll"/name, "scroll/"+name, prompt, prompt_name,
                "generated parent texture" if name.startswith("scroll-full") else "scroll region verified")
    for version in (1, 2):
        name = f"frame-hd-b-v{version}.png"
        receipt_name = "frame-art.json" if version == 1 else "frame-art-v2.json"
        provenance = json.loads((base/"scroll/frame"/receipt_name).read_text())
        picture(base/"scroll/frame"/name, "scroll/frame/"+name, prompt, prompt_name,
                "unused geometry draft" if version == 1 else "upper frame region verified",
                derivation=provenance)
    source = catalog/"window/window-skin-hd-b-v1.png"
    picture(source, "window/"+source.name, source.with_suffix(".prompt.txt"), "window/"+source.with_suffix(".prompt.txt").name, "generated; live integration pending")
    counts = {"png": sum(r["kind"] == "png" for r in files.values()),
              "prompts": sum(r["kind"] == "prompt" for r in files.values()),
              "bytes": sum(r["bytes"] for r in files.values())}
    index = {"classification": "private repository HD art archive; not a production asset-pack schema",
             "authorization_date": "2026-10-14", "authorization": "目前這些圖片都很棒 記得保存在 github repo 當中喔",
             "generator": "built-in image_gen.imagegen", "rights": "private project archive; no public release authorization",
             "counts": counts, "archive_tool_sha256": sha(Path(__file__)), "files": sorted(files.values(), key=lambda r:r["path"])}
    (output/"README.md").write_text(README)
    (output/"index.json").write_text(json.dumps(index, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(counts, ensure_ascii=False))


if __name__ == "__main__":
    main()
