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
- [地面圖集的材質與衍生圖](terrain/materials/)，保留兩版製作紀錄及完整提示詞。
- [十項非空單位的透明裁切](terrain/units/)，原版三個空白來源保持空白。
- [物件的透明裁切及審查](terrain/objects/)，獨立物件分項接入，城牆接縫草稿另列。
- [城牆校準與完整物件圖集](terrain/objects/walls/)，共用手繪材質對齊屋簷、石基及接縫，原版空白格保留。
- [晴天與陰天天氣圖](art/weather-generation.json)，第二版保留原版岩峰構圖。
- [開場騎兵與武將群像草稿](art/opening-generation.json)，保留原始生成圖，幾何與呈現待驗證。
- [戰鬥背景與開場徽記草稿](art/remaining-generation-v1.json)，保留原版參考定位、完整提示詞及未校準的原始生成圖。
- [校準背景、徽記與開場圖](art/static/)，五項有效美術區及原版緩衝區的幾何紀錄。
- [Family Soft 字標](art/static/logo/)，保留原始品牌字樣與位置，另存生成圖與完整提示詞。
- [動畫、標題與剩餘介面素材](art/remaining/)，20 項來源的逐格裁切、鏡像與留白核對；美術準備與執行器接入分開記錄。
- [OP-FULL 開場肖像](art/opening-full-generation-v2.json)與[完整提示詞](art/OP-FULL-hd-b-v2.prompt.txt)，第二版校正原生色盤下的藍白雲帶；[選定素材](art/opening-full/OP-FULL.OZM-hd-v2.png)與第一版均保存，來源、校準與遊戲內驗證分開記錄。
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
                assert policy in ("opaque", "transparent", "transparent_sampled")
                if policy == "opaque":
                    assert alpha == (255, 255)
                elif policy == "transparent":
                    assert alpha == (0, 255)
                else:
                    # 有限取樣可能沒取到原圖少數 alpha=255 像素；不改生成圖的 alpha。
                    assert alpha[0] == 0 and 0 < alpha[1] <= 255 and "derivation" in metadata
                if policy != "opaque":
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
    ui_generation = catalog/"art/ui-generation.json"
    ui_drafts = {}
    if ui_generation.exists():
        ui_drafts = {r["asset"]:r["sha256"] for r in json.loads(ui_generation.read_text())["records"]}
    weather_path = catalog/"art/weather-generation.json"
    weather_art = {r["asset"]: r for r in json.loads(weather_path.read_text())["records"]} if weather_path.exists() else {}
    opening_path = catalog/"art/opening-generation.json"
    opening_art = {r["asset"]:r for r in json.loads(opening_path.read_text())["records"]} if opening_path.exists() else {}
    remaining_drafts_path = catalog/"art/remaining-generation-v1.json"
    remaining_drafts = {r["asset"]:r for r in json.loads(remaining_drafts_path.read_text())["records"]} if remaining_drafts_path.exists() else {}
    remaining_full_generation_path = catalog/"art/remaining-full-generation-v1.json"
    remaining_full_generation = {r["asset"]:r for r in json.loads(remaining_full_generation_path.read_text())["records"]} if remaining_full_generation_path.exists() else {}
    background_root = catalog/"graphics/static"
    background_profiles_path = background_root/"profiles.json"
    calibrated_inputs = {}
    if background_profiles_path.exists():
        background_profiles = json.loads(background_profiles_path.read_text())
        assert background_profiles["tool_sha256"] == sha(background_root/"prepare-source.txt") == sha(project/"tools/hd-background-prepare.py")
        calibrated_inputs = {r["raw_asset"]:r for r in background_profiles["records"]}
    publisher_logo_root = background_root/"logo"
    publisher_logo_profiles_path = publisher_logo_root/"profiles.json"
    if publisher_logo_profiles_path.exists():
        publisher_logo_profiles = json.loads(publisher_logo_profiles_path.read_text())
        assert publisher_logo_profiles["tool_sha256"] == sha(publisher_logo_root/"prepare-source.txt") == sha(project/"tools/hd-logo-prepare.py")
        logo_record = publisher_logo_profiles["record"]
        calibrated_inputs[logo_record["raw_asset"]] = logo_record
    for source in sorted((catalog/"art").glob("*-hd-b-v*.png")):
        name = source.name
        status = "draft; integration and review pending"
        if name in remaining_full_generation:
            assert sha(source) == remaining_full_generation[name]["sha256"]
            status = "source-referenced generated art; calibrated selection and runtime status recorded in remaining profiles"
        if name in remaining_drafts:
            assert sha(source) == remaining_drafts[name]["sha256"]
            status = "source-referenced artwork draft; black padding, geometry and runtime draw eligibility pending"
        if name in selected_art:
            assert sha(source) == selected_art[name]
            status = "normal player path verified" if name in normal_art else "reviewed source and common draw consumer integrated; normal path not yet sampled"
        elif name in reviewed_art:
            assert sha(source) == reviewed_art[name]
            status = "illustration visually reviewed; source profile and runtime integration pending"
        elif name in ui_drafts:
            assert sha(source) == ui_drafts[name]
            status = "battle UI frame visually reviewed; geometry and runtime presentation pending"
        elif name in weather_art:
            assert sha(source) == weather_art[name]["sha256"]
            status = weather_art[name]["status"]
            proof = catalog/"graphics/verification-summary.json"
            if weather_art[name]["version"] == 2 and proof.exists():
                checked = json.loads(proof.read_text())
                assert checked["verified"]
                status = "common opaque draw consumer integrated; WEATHER1 normal first battle verified" if weather_art[name]["source"] == "WEATHER1.OZM" else "common opaque draw consumer integrated; overcast normal path not yet sampled"
        elif name in opening_art:
            assert sha(source) == opening_art[name]["sha256"]
            status = opening_art[name]["status"]
        if name in calibrated_inputs:
            assert sha(source) == calibrated_inputs[name]["raw_asset_sha256"]
            status = "generation input for reviewed static source geometry; runtime uses calibrated child asset"
        picture(source, "art/"+name, source.with_suffix(".prompt.txt"), "art/"+source.with_suffix(".prompt.txt").name, status)
    revisions = catalog/"art/revisions.json"
    if revisions.exists():
        copy(revisions, "art/revisions.json", "revision-metadata")
    if remaining_path.exists():
        copy(remaining_path, "art/remaining-generation.json", "generation-metadata")
    if council:
        copy(council_path, "art/gungi-revision.json", "revision-metadata")
    ui_generation = catalog/"art/ui-generation.json"
    if ui_generation.exists():
        for record in json.loads(ui_generation.read_text())["records"]:
            assert sha(catalog/"art"/record["asset"]) == record["sha256"]
            assert sha(catalog/"art"/record["prompt"]) == record["prompt_sha256"]
        copy(ui_generation, "art/ui-generation.json", "generation-metadata")
    if weather_path.exists():
        for record in weather_art.values():
            assert sha(catalog/"art"/record["asset"]) == record["sha256"]
            assert sha(catalog/"art"/record["prompt"]) == record["prompt_sha256"]
        copy(weather_path, "art/weather-generation.json", "generation-metadata")
    if opening_path.exists():
        for record in opening_art.values():
            assert sha(catalog/"art"/record["asset"]) == record["sha256"]
            assert sha(catalog/"art"/record["prompt"]) == record["prompt_sha256"]
        copy(opening_path,"art/opening-generation.json","generation-metadata")
    if remaining_drafts_path.exists():
        for record in remaining_drafts.values():
            assert sha(catalog/"art"/record["asset"]) == record["sha256"]
            assert sha(catalog/"art"/record["prompt"]) == record["prompt_sha256"]
        copy(remaining_drafts_path,"art/remaining-generation-v1.json","generation-metadata")
    publisher_logo_generation_path = catalog/"art/logo-generation-v1.json"
    if publisher_logo_generation_path.exists():
        copy(publisher_logo_generation_path,"art/logo-generation-v1.json","generation-metadata")
    if publisher_logo_profiles_path.exists():
        record = publisher_logo_profiles["record"]
        source = publisher_logo_root/record["prepared"]
        assert sha(source) == record["asset_sha256"]
        copy(source,"art/"+record["asset"],"png",derivation=record,prompt="art/"+record["prompt"],
             status="reviewed original publisher wordmark, calibrated geometry and common opaque profile")
        for name in ["draft-manifest.json","review.json","profiles.json","prepare-source.txt"]:
            copy(publisher_logo_root/name,"art/static/logo/"+name,"preparation-metadata")
    if background_profiles_path.exists():
        for record in background_profiles["records"]:
            source = background_root/record["prepared"]
            assert sha(source) == record["asset_sha256"]
            copy(source,"art/"+record["asset"],"png",derivation=record,prompt="art/"+record["prompt"],
                 status="reviewed calibrated static source; common opaque draw consumer registered, normal case not yet sampled")
        gallery = background_root/background_profiles["gallery"]
        assert sha(gallery) == background_profiles["gallery_sha256"]
        copy(gallery,"art/static/"+gallery.name,"png",status="generated-only calibrated static art gallery")
        for name in ["draft-manifest.json","prepare-source.txt","review.json","profiles.json"]:
            copy(background_root/name,"art/static/"+name,"preparation-metadata")
    battle_frame = catalog/"battle-frame"
    for revision in (1,2):
        geometry_path = battle_frame/("battle-frame-geometry.json" if revision == 1 else "battle-frame-geometry-v2.json")
        if not geometry_path.exists():
            continue
        geometry = json.loads(geometry_path.read_text())
        source = battle_frame/geometry["prepared_asset"]
        assert sha(source) == geometry["prepared_png_sha256"]
        assert sha(catalog/"art"/geometry["asset"]) == geometry["asset_sha256"]
        prompt = catalog/"art/MAPGAMEN-hd-b-v1.prompt.txt"
        status = "superseded frame geometry draft; post projections corrected in v2" if revision == 1 else "reviewed frame geometry; runtime and GUI acceptance pending"
        proof_path = battle_frame/"verification-summary.json"
        if revision == 2 and proof_path.exists():
            proof = json.loads(proof_path.read_text())
            assert proof["geometry_sha256"] == sha(geometry_path) and proof["asset_sha256"] == sha(source)
            assert len(proof["records"]) == 3
            status = "normal pre-battle modal and first battle verified; native-index foreground and joint HD scene underlay"
        picture(source, "battle-frame/"+source.name, prompt, f"battle-frame/battle-frame-hd-b-v{revision}.prompt.txt", status)
        copy(geometry_path,"battle-frame/"+geometry_path.name,"preparation-metadata")
    old_prepare = battle_frame/"prepare-v1-source.txt"
    if old_prepare.exists():
        copy(old_prepare,"battle-frame/prepare-v1-source.txt","preparation-source-snapshot")
    terrain = catalog/"terrain/art"
    terrain_profiles = catalog/"terrain/prepared/profiles.json"
    verified_tree_asset = ""
    tree_instances = 0
    tree_summary_path = catalog/"terrain/trees/verification-summary.json"
    if tree_summary_path.exists():
        tree_summary = json.loads(tree_summary_path.read_text())
        tree_source_path = catalog/"terrain/trees/tree-source-verification.json"
        assert tree_summary["source_verification_sha256"] == sha(tree_source_path)
        assert tree_summary["normal_HD_object_instances"] == 3
        tree_source = json.loads(tree_source_path.read_text())
        verified_tree_asset = tree_source["asset"]
        assert tree_summary["asset_sha256"] == tree_source["asset_sha256"]
        tree_instances = 3
    backdrop_summary_path = catalog/"terrain/backgrounds/verification-summary.json"
    if backdrop_summary_path.exists():
        backdrop_summary = json.loads(backdrop_summary_path.read_text())
        assert backdrop_summary["normal_HD_object_instances"] == 4
        assert backdrop_summary["background_verification_sha256"] == sha(catalog/"terrain/backgrounds/background-verification.json")
        assert backdrop_summary["asset_sha256"] == tree_source["asset_sha256"]
        tree_instances = 4
    normal_unit_assets = {}
    scene_summary_path = catalog/"terrain/scene/verification-summary.json"
    if scene_summary_path.exists():
        scene_summary = json.loads(scene_summary_path.read_text())
        assert scene_summary["approved_unit_indices"] == [1,3] and scene_summary["normal_HD_unit_instances"] == 2
        receipt_path = catalog/"terrain/scene/desktop/desktop-scene.json"
        assert scene_summary["records"][0]["receipt_sha256"] == sha(receipt_path)
        receipt = json.loads(receipt_path.read_text())
        for index in scene_summary["approved_unit_indices"]:
            info = receipt["desktop"]["assets"]["terrain_units"][str(index)]
            assert info["validation"] == "ready"
            normal_unit_assets[f"unit-{index:03d}-hd-b-v1.png"] = info["hd_sha256"]
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
    unit_root = catalog/"terrain/units"
    unit_profiles_path = unit_root/"profiles.json"
    selected_units, unit_superseded = {}, set()
    object_root = catalog/"terrain/objects/expanded"
    object_profiles_path = object_root/"profiles.json"
    selected_objects = {}
    if object_profiles_path.exists():
        object_profiles = json.loads(object_profiles_path.read_text())
        assert object_profiles["tool_sha256"] == sha(project/"tools/hd-object-prepare.py")
        selected_objects = {r["asset"]:r for r in object_profiles["records"] if r["index"] in object_profiles["approved_indices"]}
    wall_root = catalog/"terrain/objects/walls"
    wall_profiles_path = wall_root/"profiles.json"
    wall_input_names = set()
    if wall_profiles_path.exists():
        wall_profiles = json.loads(wall_profiles_path.read_text())
        assert wall_profiles["tool_sha256"] == sha(wall_root/"profiles-source.txt") == sha(project/"tools/hd-wall-prepare.py")
        assert wall_profiles["approved_indices"] == list(range(24))
        selected_objects.update({r["asset"]:r for r in wall_profiles["records"] if r["asset"]})
        draft = json.loads((wall_root/wall_profiles["draft_manifest"]).read_text())
        wall_input_names = {r["asset"] for r in draft["inputs"]}
    if unit_profiles_path.exists():
        unit_profiles = json.loads(unit_profiles_path.read_text())
        assert unit_profiles["tool_sha256"] == sha(project/"tools/hd-unit-prepare.py")
        assert unit_profiles["nonempty_sources"] == 10 and unit_profiles["empty_indices"] == [10,11,12]
        selected_units = {r["asset"]:r for r in unit_profiles["records"] if r["asset"]}
        unit_superseded = {r["supersedes"] for r in json.loads((unit_root/"pose-revisions-v2.json").read_text())["records"]}
    if terrain.exists():
        for source in sorted(terrain.glob("*-hd-b-v*.png")):
            unit = source.name.startswith("unit-")
            object_draft = source.name.startswith("object-")
            status = "generated unit draft; original identity and footprint review pending" if unit else "superseded or unreviewed floor draft"
            if source.name in unit_superseded:
                status = "superseded pose or weapon-head draft; original silhouette corrected in v2"
            if source.name == "unit-006-hd-b-v1.png":
                status = "superseded pose draft; exposed blade corrected to lower left in v2"
            if source.name == "unit-006-hd-b-v2.png":
                status = "reviewed original pose correction; runtime integration pending"
            if source.name in selected_units:
                assert sha(source) == selected_units[source.name]["asset_sha256"]
                status = "reviewed source pose and alpha crop; common unit consumer integrated, normal source not yet sampled"
            if source.name in normal_unit_assets:
                assert sha(source) == normal_unit_assets[source.name]
                status = "normal first-battle unit HD verified; joint floor and forest-clump composition, native visible-owner priority"
            if object_draft:
                status = "generated object draft; cell footprint and normal HD integration pending"
                if source.name in wall_input_names:
                    status = "generated material source for calibrated wall geometry; raw draft not used directly"
                if source.name in selected_objects:
                    assert sha(source) == selected_objects[source.name]["asset_sha256"]
                    status = "reviewed independent object silhouette and alpha crop; common consumer integrated, normal source not yet sampled"
                if source.name == "object-000-hd-b-v1.png":
                    status = "superseded single-tree draft; original grove composition corrected in v2"
                if source.name == verified_tree_asset:
                    assert sha(source) == tree_source["asset_sha256"]
                    status = f"normal first-battle forest clump v2 verified in {tree_instances} scenes; observed pre-draw background at UI edge" if tree_instances == 4 else "normal first-battle forest clump v2 verified in three complete scenes; clipped edge falls back"
            if source.name in approved_floor:
                assert sha(source) == approved_floor[source.name]
                status = "normal first-battle floor verified" if source.name in normal_floor_art else "reviewed floor art and common draw consumer integrated; visible normal case not yet sampled"
            picture(source, "terrain/"+source.name, source.with_suffix(".prompt.txt"),
                    "terrain/"+source.with_suffix(".prompt.txt").name, status,
                    alpha_policy="transparent" if unit or object_draft else "opaque")
        for source in sorted(terrain.glob("*-hd-gallery-v*.png")):
            copy(source, "terrain/"+source.name, "png", status="generated-art gallery; includes drafts")
        for name in ("generation.json", "floor-generation.json", "floor-revisions.json", "object-generation.json", "object-generation-expanded.json", "object-revisions.json", "unit-revisions.json"):
            source = terrain/name
            if source.exists():
                copy(source, "terrain/"+name, "generation-metadata")
    if object_profiles_path.exists():
        for record in object_profiles["records"]:
            if not record["asset"]:
                continue
            source = object_root/record["prepared"]
            assert sha(source) == record["prepared_sha256"]
            status = "reviewed independent object crop; normal player path not yet sampled" if record["index"] in object_profiles["approved_indices"] else "wall crop draft; geometry and connecting roof heights not approved"
            copy(source,"terrain/objects/"+record["prepared"],"png",alpha_policy="transparent_sampled",derivation=record,
                 prompt="terrain/"+record["prompt"],status=status)
        gallery = object_root/object_profiles["gallery"]
        assert sha(gallery) == object_profiles["gallery_sha256"]
        copy(gallery,"terrain/objects/"+gallery.name,"png",status="generated-only object gallery; includes unapproved wall geometry drafts")
        for name in ["draft-profiles.json","review.json","profiles.json"]:
            copy(object_root/name,"terrain/objects/"+name,"preparation-metadata")
    if unit_profiles_path.exists():
        for record in unit_profiles["records"]:
            if not record["asset"]:
                continue
            source = unit_root/record["prepared"]
            assert sha(source) == record["prepared_sha256"]
            copy(source,"terrain/units/"+record["prepared"],"png",alpha_policy="transparent",derivation=record,
                 prompt="terrain/"+record["prompt"],status="source-bounded transparent unit preview; actual runtime uses registered raw art/crop")
        gallery = unit_root/unit_profiles["gallery"]
        assert sha(gallery) == unit_profiles["gallery_sha256"]
        copy(gallery,"terrain/units/"+gallery.name,"png",status="generated-only unit gallery; original references excluded")
        for name in ["pose-revisions.json","pose-revisions-v2.json","draft-profiles.json","review.json","profiles.json"]:
            copy(unit_root/name,"terrain/units/"+name,"preparation-metadata")
        attempt = unit_root/"attempt-01"
        previous = json.loads((attempt/"draft-profiles.json").read_text())
        assert previous["tool_sha256"] == sha(attempt/"prepare-source.txt")
        for record in previous["records"]:
            if not record["asset"]:
                continue
            source = attempt/record["prepared"]
            assert sha(source) == record["prepared_sha256"]
            copy(source,"terrain/units/attempt-01/"+record["prepared"],"png",alpha_policy="transparent",
                 status="first crop review; unit008 single spearhead subsequently corrected")
        copy(attempt/previous["gallery"],"terrain/units/attempt-01/"+previous["gallery"],"png",status="superseded generated-only unit review gallery")
        for name in ["draft-profiles.json","pose-revisions.json","prepare-source.txt"]:
            copy(attempt/name,"terrain/units/attempt-01/"+name,"preparation-metadata")
    if wall_profiles_path.exists():
        for folder in sorted(wall_root.glob("draft-v*")):
            manifest_path = folder/"draft-manifest.json"
            manifest = json.loads(manifest_path.read_text())
            assert manifest["tool_sha256"] == sha(folder/"prepare-source.txt")
            for record in manifest["records"]:
                source = folder/record["asset"]
                assert sha(source) == record["asset_sha256"]
                copy(source,"terrain/objects/walls/"+folder.name+"/"+source.name,"png",alpha_policy="transparent",derivation=record,
                     prompts=["terrain/"+r["prompt"] for r in manifest["inputs"]],
                     status="calibrated source-bounded wall art" if folder.name == "draft-v2" else "first geometry calibration draft; superseded by v2")
            for name in [manifest["gallery"]]+[r["HD"] for r in manifest.get("fixtures",[])]:
                copy(folder/name,"terrain/objects/walls/"+folder.name+"/"+name,"png",status="generated-only calibration gallery or offline enclosure; not normal player path")
            for name in ["draft-manifest.json","prepare-source.txt","review.json"]:
                if (folder/name).exists():
                    copy(folder/name,"terrain/objects/walls/"+folder.name+"/"+name,"preparation-metadata")
        for record in wall_profiles["records"]:
            if not record["asset"]:
                continue
            if record["index"] in wall_profiles["wall_source_indices"] and record["index"] == record["asset_source_index"]:
                source = project/record["asset_local_source"]
                assert sha(source) == record["asset_sha256"]
                copy(source,"terrain/"+record["asset"],"png",alpha_policy="transparent",derivation=record,
                     prompts=["terrain/"+r["prompt"] for r in record["material_inputs"]],
                     status="reviewed calibrated wall; common object consumer integrated, normal source not yet sampled")
            source = wall_root/record["prepared"]
            assert sha(source) == record["prepared_sha256"]
            copy(source,"terrain/objects/walls/"+record["prepared"],"png",alpha_policy="transparent_sampled",derivation=record,
                 status="registered object preview; original eight images and crops unchanged")
        copy(wall_root/wall_profiles["gallery"],"terrain/objects/walls/"+wall_profiles["gallery"],"png",
             status="generated-only complete 24 nonempty object gallery; two original blank objects preserved")
        for name in ["profiles.json","profiles-source.txt"]:
            copy(wall_root/name,"terrain/objects/walls/"+name,"preparation-metadata")
    material_root = catalog/"terrain/materials"
    if (material_root/"generation.json").exists():
        generation = json.loads((material_root/"generation.json").read_text())
        prompts = ["terrain/"+"floor-000-hd-b-v1.prompt.txt"]
        for record in generation["records"]:
            assert sha(material_root/record["asset"]) == record["sha256"]
            prompt_name = "terrain/materials/"+record["prompt"]
            prompts.append(prompt_name)
            picture(material_root/record["asset"],"terrain/materials/"+record["asset"],
                    material_root/record["prompt"],prompt_name,"visually reviewed generated terrain material")
        for version, folder, manifest_name in [(1,"drafts","draft-manifest.json"),(2,"drafts-v2","draft-manifest-v2.json")]:
            manifest_path = material_root/manifest_name
            if not manifest_path.exists():
                continue
            manifest = json.loads(manifest_path.read_text())
            if version == 2:
                assert manifest["tool_sha256"] == sha(project/"tools/hd-terrain-materials.py")
            integrated = version == 2 and (material_root/"profiles.json").exists()
            for record in manifest["records"]:
                if record["inherited"] or record["asset_source_index"] != record["index"]:
                    continue
                source = material_root/folder/record["asset"]
                assert sha(source) == record["asset_sha256"]
                name = "terrain/"+record["asset"] if integrated else "terrain/materials/drafts-v1/"+record["asset"]
                copy(source,name,"png",material_prompts=prompts,derivation=record,
                     status="reviewed source-specific material composition; common floor consumer integrated, normal case not individually sampled" if integrated else "superseded material-boundary draft; isolated color dots corrected in v2")
            copy(manifest_path,"terrain/materials/"+manifest_name,"preparation-metadata")
            gallery = material_root/manifest["gallery"]
            assert sha(gallery) == manifest["gallery_sha256"]
            copy(gallery,"terrain/materials/"+gallery.name,"png",status="generated-only floor gallery; original references excluded")
        for name in ["generation.json","review-v2.json","profiles.json","prepare-v1-source.txt"]:
            source = material_root/name
            if source.exists():
                copy(source,"terrain/materials/"+name,"preparation-metadata")
    for version in [1, 2]:
        full_opening_generation = catalog/f"art/opening-full-generation-v{version}.json"
        if full_opening_generation.exists():
            copy(full_opening_generation,f"art/opening-full-generation-v{version}.json","generation-metadata")
    for name in ["E-4-generation-v1.json", "B-6-generation-v1.json"]:
        metadata = catalog/"art"/name
        if metadata.exists():
            for record in json.loads(metadata.read_text())["records"]:
                assert sha(catalog/"art"/record["asset"]) == record["sha256"]
                assert sha(catalog/"art"/record["prompt"]) == record["prompt_sha256"]
            copy(metadata,"art/"+name,"generation-metadata")
    for version, folder in [(1, "art"), (2, "art-v2")]:
        full_opening_root = catalog/"remaining/history"/folder
        if not (full_opening_root/"profiles.json").exists():
            continue
        prepared = json.loads((full_opening_root/"profiles.json").read_text())
        assert prepared["tool_sha256"] == sha(full_opening_root/"prepare-source.txt")
        if version == 2:
            assert prepared["tool_sha256"] == sha(project/"tools/hd-opening-prepare.py")
        record = prepared["record"]
        source = full_opening_root/Path(record["asset"]).name
        assert sha(source) == record["asset_sha256"]
        copy(source,"art/"+record["asset"],"png",derivation=record,
             prompts=["art/"+record["prompt"]],status="calibrated runtime-observed source; normal HD validation separately recorded")
        for name in ["profiles.json","prepare-source.txt"]:
            target_name = name if version == 1 else ("profiles-v2.json" if name == "profiles.json" else "prepare-v2-source.txt")
            copy(full_opening_root/name,"art/opening-full/"+target_name,"preparation-metadata")
    if remaining_full_generation_path.exists():
        copy(remaining_full_generation_path,"art/remaining-full-generation-v1.json","generation-metadata")
        remaining_full_root = catalog/"remaining"
        copy(remaining_full_root/"sources.json","art/remaining/sources.json","source-metadata")
        copy(remaining_full_root/"intake-source.txt","art/remaining/intake-source.txt","preparation-metadata")
        if (remaining_full_root/"generation-inputs-v1.json").exists():
            copy(remaining_full_root/"generation-inputs-v1.json","art/remaining/generation-inputs-v1.json","generation-metadata")
        for folder in sorted(remaining_full_root.glob("draft-v*")):
            manifest_path = folder/"draft-manifest.json"
            if not manifest_path.exists():
                continue
            manifest = json.loads(manifest_path.read_text())
            assert manifest["tool_sha256"] == sha(folder/"prepare-source.txt")
            for record in manifest["records"]:
                source = folder/record["prepared"]
                assert sha(source) == record["asset_sha256"]
                copy(source,"art/remaining/"+folder.name+"/"+record["prepared"],"png",derivation=record,
                     prompts=["art/"+r["prompt"] for r in record["inputs"]],
                     status="calibrated art draft; runtime registration separately required")
                if record["name"].startswith("ANIME"):
                    for frame in record["frames"]:
                        source = folder/frame["prepared"]
                        assert sha(source) == frame["asset_sha256"]
                        copy(source,"art/remaining/"+folder.name+"/"+frame["prepared"],"png",derivation=frame,
                             status="generated animation crop or proven source mirror; original empty frame preserved")
            for gallery in manifest["galleries"]:
                source = folder/gallery["asset"]
                assert sha(source) == gallery["sha256"]
                copy(source,"art/remaining/"+folder.name+"/"+gallery["asset"],"png",status="generated-only HD overview")
            for name in ["draft-manifest.json","prepare-source.txt","review.json"]:
                if (folder/name).exists():
                    copy(folder/name,"art/remaining/"+folder.name+"/"+name,"preparation-metadata")
        remaining_full_profiles_path = remaining_full_root/"profiles.json"
        if remaining_full_profiles_path.exists():
            selected = json.loads(remaining_full_profiles_path.read_text())
            assert selected["tool_sha256"] == sha(project/"tools/hd-remaining-prepare.py")
            remaining_runtime_proof_path = remaining_full_root/"runtime-verification-summary.json"
            remaining_runtime_proof = json.loads(remaining_runtime_proof_path.read_text()) if remaining_runtime_proof_path.exists() else None
            if remaining_runtime_proof:
                assert remaining_runtime_proof["verified"]
                assert remaining_runtime_proof["runtime_profiles_sha256"] == sha(remaining_full_root/"runtime-profiles-v1.json")
            for record in selected["records"]:
                source = remaining_full_root/record["prepared"]
                assert sha(source) == record["asset_sha256"]
                status = "reviewed calibrated art; runtime source registration and normal path validation pending"
                if remaining_runtime_proof:
                    status = "normal title/menu and individual missing asset verified" if record["name"] in remaining_runtime_proof["normal_new_sources"] else "source and shared opaque consumer integrated; normal path not individually sampled"
                copy(source,"art/"+record["asset"],"png",derivation=record,
                     prompts=["art/"+r["prompt"] for r in record["inputs"]],
                     status=status)
            copy(remaining_full_profiles_path,"art/remaining/profiles.json","preparation-metadata")
            for name in ["runtime-profiles-v1.json","runtime-ready-v1.json","runtime-profile-source.txt"]:
                if (remaining_full_root/name).exists():
                    copy(remaining_full_root/name,"art/remaining/"+name,"registration-metadata")
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
