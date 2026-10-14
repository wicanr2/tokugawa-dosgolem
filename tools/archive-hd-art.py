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
- [戰場地面、物件與單位草稿](terrain/)，物件／單位 PNG 保留透明背景。
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
                assert im.format == "PNG"
                alpha = im.convert("RGBA").getextrema()[3]
                policy = metadata.get("alpha_policy", "opaque")
                assert policy in ("opaque", "transparent")
                assert alpha == (255, 255) if policy == "opaque" else alpha == (0, 255)
                if policy == "transparent":
                    record["alpha_range"] = list(alpha)
                record["size"] = list(im.size)
        if name in files:
            assert files[name] == record
        files[name] = record

    def picture(source, name, prompt, prompt_name, status, **metadata):
        copy(prompt, prompt_name, "prompt")
        copy(source, name, "png", prompt=prompt_name, status=status, **metadata)

    picture(base/"style-comparison-v1.png", "concept/style-comparison-v1.png", base/"style-comparison-v1.prompt.txt", "concept/style-comparison-v1.prompt.txt", "style comparison; B selected")
    picture(base/"A-0-hd-b-v1.png", "A-0-hd-b-v1.png", base/"A-0-hd-b-v1.prompt.txt", "A-0-hd-b-v1.prompt.txt", "normal player path verified")
    story_profiles = catalog/"story-expanded/profiles.json"
    selected_art = {}
    normal_art = set()
    if story_profiles.exists():
        profiles = json.loads(story_profiles.read_text())
        assert profiles["tool_sha256"] == sha(project/"tools/hd-story-prepare.py")
        selected_art = {r["asset"]: r["asset_sha256"] for r in profiles["records"]}
        verified_path = catalog/"story-expanded/desktop/verification-portraits.json"
        if verified_path.exists():
            receipt_path = verified_path.with_name("desktop-portraits.json")
            verified = json.loads(verified_path.read_text())
            assert verified["verified"] and verified["receipt_sha256"] == sha(receipt_path)
            receipt = json.loads(receipt_path.read_text())
            by_resource = {r["resource"]: r for r in profiles["records"]}
            for check in verified["portrait_checks"]:
                if check["mode"] != "hd":
                    continue
                profile = by_resource[check["resource"]]
                loaded = receipt["desktop"]["assets"]
                if check["resource"] != "A-0.OZM":
                    loaded = loaded["story_portraits"][check["resource"]]
                assert loaded["hd_sha256"] == profile["asset_sha256"]
                normal_art.add(profile["asset"])
    remaining_path = catalog/"art/remaining-generation.json"
    reviewed_art = {}
    if remaining_path.exists():
        reviewed_art = {r["asset"]: r["sha256"] for r in json.loads(remaining_path.read_text())["records"]}
    council_path = catalog/"art/gungi-revision.json"
    council = json.loads(council_path.read_text()) if council_path.exists() else None
    if council:
        reviewed_art[council["asset"]] = council["sha256"]
        reviewed_art.pop(council["superseded_asset"], None)
    for source in sorted((catalog/"art").glob("*-hd-b-v*.png")):
        name = source.name
        status = "draft; integration and review pending"
        if name in selected_art:
            assert sha(source) == selected_art[name]
            status = "normal player path verified" if name in normal_art else "reviewed source and common draw consumer integrated; normal path not yet sampled"
        elif name in reviewed_art:
            assert sha(source) == reviewed_art[name]
            status = "illustration visually reviewed; source profile and runtime integration pending"
        picture(source, "art/"+name, source.with_suffix(".prompt.txt"), "art/"+source.with_suffix(".prompt.txt").name, status)
    revisions = catalog/"art/revisions.json"
    if revisions.exists():
        copy(revisions, "art/revisions.json", "revision-metadata")
    if remaining_path.exists():
        copy(remaining_path, "art/remaining-generation.json", "generation-metadata")
    if council:
        copy(council_path, "art/gungi-revision.json", "revision-metadata")
    terrain = catalog/"terrain/art"
    terrain_profiles = catalog/"terrain/prepared/profiles.json"
    verified_tree_asset = ""
    tree_summary_path = catalog/"terrain/trees/verification-summary.json"
    if tree_summary_path.exists():
        tree_summary = json.loads(tree_summary_path.read_text())
        tree_source_path = catalog/"terrain/trees/tree-source-verification.json"
        assert tree_summary["source_verification_sha256"] == sha(tree_source_path)
        assert tree_summary["normal_HD_object_instances"] == 3
        tree_source = json.loads(tree_source_path.read_text())
        verified_tree_asset = tree_source["asset"]
        assert tree_summary["asset_sha256"] == tree_source["asset_sha256"]
    approved_floor = {}
    normal_floor_art = set()
    if terrain_profiles.exists():
        prepared = json.loads(terrain_profiles.read_text())
        assert prepared["tool_sha256"] == sha(project/"tools/hd-terrain-prepare.py")
        approved_floor = {r["asset"]: r["asset_sha256"] for r in prepared["records"] if r["asset"]}
        proof_path = catalog/"terrain/runtime/desktop/verification-terrain.json"
        if proof_path.exists():
            proof = json.loads(proof_path.read_text())
            receipt_path = proof_path.with_name("desktop-terrain.json")
            assert proof["verified"] and proof["receipt_sha256"] == sha(receipt_path)
            receipt = json.loads(receipt_path.read_text())
            by_index = {r["index"]: r for r in prepared["records"]}
            for check in proof["terrain_checks"]:
                for draw in check["active_rectangles"]:
                    profile = by_index[draw["index"]]
                    assert receipt["desktop"]["assets"]["terrain"][str(draw["index"])]["hd_sha256"] == profile["asset_sha256"]
                    normal_floor_art.add(profile["asset"])
    if terrain.exists():
        for source in sorted(terrain.glob("*-hd-b-v*.png")):
            unit = source.name.startswith("unit-")
            object_draft = source.name.startswith("object-")
            status = "generated unit draft; original identity and footprint review pending" if unit else "superseded or unreviewed floor draft"
            if object_draft:
                status = "generated object draft; cell footprint and normal HD integration pending"
                if source.name == "object-000-hd-b-v1.png":
                    status = "superseded single-tree draft; original grove composition corrected in v2"
                if source.name == verified_tree_asset:
                    assert sha(source) == tree_source["asset_sha256"]
                    status = "normal first-battle forest clump v2 verified in three complete scenes; clipped edge falls back"
            if source.name in approved_floor:
                assert sha(source) == approved_floor[source.name]
                status = "normal first-battle floor verified" if source.name in normal_floor_art else "reviewed floor art and common draw consumer integrated; visible normal case not yet sampled"
            picture(source, "terrain/"+source.name, source.with_suffix(".prompt.txt"),
                    "terrain/"+source.with_suffix(".prompt.txt").name, status,
                    alpha_policy="transparent" if unit or object_draft else "opaque")
        for source in sorted(terrain.glob("*-hd-gallery-v*.png")):
            copy(source, "terrain/"+source.name, "png", status="generated-art gallery; includes drafts")
        for name in ("generation.json", "floor-generation.json", "floor-revisions.json", "object-generation.json", "object-revisions.json"):
            source = terrain/name
            if source.exists():
                copy(source, "terrain/"+name, "generation-metadata")
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
    window_status = "generated; live integration pending"
    proof = catalog/"window/desktop-window-modal/verification-window-modal.json"
    receipt = catalog/"window/desktop-window-modal/desktop-window-modal.json"
    if proof.exists():
        verified = json.loads(proof.read_text())
        assert verified["verified"] and verified["receipt_sha256"] == sha(receipt)
        loaded = json.loads(receipt.read_text())["desktop"]["assets"]["window"]
        assert loaded["validation"] == "ready" and loaded["hd_sha256"] == sha(source)
        window_status = "normal player path verified; selection, confirmation and status windows"
    picture(source, "window/"+source.name, source.with_suffix(".prompt.txt"), "window/"+source.with_suffix(".prompt.txt").name, window_status)
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
