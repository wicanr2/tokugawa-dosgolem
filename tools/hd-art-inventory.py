"""重生本機美術覆蓋證據；資源筆數不代表圖集內所有圖塊已完成。"""
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, __version__ as pillow_version


PROJECT = Path("/project")
ROOT = PROJECT / "workplace/hd-prototype/catalog"


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def art_info(path, prompt):
    assert path.is_file() and prompt.is_file(), (path, prompt)
    with Image.open(path) as im:
        assert im.convert("RGBA").getextrema()[3] == (255, 255), path
        size = list(im.size)
    return {"path": str(path.relative_to(PROJECT)), "sha256": sha(path), "size": size,
            "prompt": str(prompt.relative_to(PROJECT)), "prompt_sha256": sha(prompt),
            "generator": "built-in image_gen.imagegen", "style": "B 戰國手繪風",
            "rights": "generated derivative; user authorized private Git archive on 2026-10-14; public distribution pending"}


def main():
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000
    assert pillow_version == "9.4.0"
    output = Path("/output")
    assert output.stat().st_uid == 1000
    decoded = json.loads((ROOT / "decoded/inventory.json").read_text())
    extraction = json.loads((ROOT / "kao/kao-extraction.json").read_text())
    selected = json.loads((ROOT / "kao/selected/manifest.json").read_text())
    inventory = json.loads((PROJECT / "workplace/hd-prototype/asset-inventory.json").read_text())
    assert len(decoded["records"]) == 112 and len(extraction["records"]) == len(selected["records"]) == 81
    originals = {r["name"]: r for r in inventory["records"]}
    profile_path = ROOT / "story-expanded/profiles.json"
    profiles = {}
    if profile_path.exists():
        prepared = json.loads(profile_path.read_text())
        assert prepared["tool_sha256"] == sha(PROJECT / "tools/hd-story-prepare.py")
        profiles = {r["resource"]: r for r in prepared["records"]}
    direct = {}
    for record in decoded["records"]:
        stem = record["name"].removesuffix(".OZM")
        path = ROOT / "art" / f"{stem}-hd-b-v1.png"
        if record["name"] == "GUNGI.OZM" and (ROOT / "art/GUNGI-hd-b-v2.png").exists():
            path = ROOT / "art/GUNGI-hd-b-v2.png"
        if record["name"] in profiles and profiles[record["name"]]["art_source_resource"] == record["name"]:
            path = ROOT / "art" / profiles[record["name"]]["asset"]
        if stem == "A-0":
            path = PROJECT / "workplace/hd-prototype/A-0-hd-b-v1.png"
        if path.exists():
            direct[record["name"]] = art_info(path, path.with_suffix(".prompt.txt"))
    shared = {}
    for record in decoded["records"]:
        if record["name"] in direct and record.get("indices_sha256"):
            shared.setdefault(record["indices_sha256"], record["name"])

    verified_story = {}
    verified_kao = {}
    proof_sources = []
    for directory in ("desktop", "kao-desktop", "story/desktop", "story-expanded/desktop"):
        verify_path = ROOT / directory / "verification-portraits.json"
        receipt_path = ROOT / directory / "desktop-portraits.json"
        if not verify_path.exists():
            continue
        verify = json.loads(verify_path.read_text())
        assert verify["verified"] and verify["receipt_sha256"] == sha(receipt_path)
        receipt = json.loads(receipt_path.read_text())
        assert all(e["guest_unchanged"] for e in receipt["desktop"]["events"] if e["kind"] == "toggle")
        proof_sources.append({"verification": str(verify_path.relative_to(PROJECT)), "sha256": sha(verify_path)})
        for check in verify["portrait_checks"]:
            if check["mode"] == "hd":
                name = check["resource"]
                assets = receipt["desktop"]["assets"]
                loaded = assets if name == "A-0.OZM" else assets["story_portraits"][name]
                if name in direct and loaded["hd_sha256"] == direct[name]["sha256"]:
                    verified_story[name] = check
        for check in verify.get("kao_checks", []):
            if check["mode"] == "hd":
                verified_kao[check["index"]] = check

    terrain_path = ROOT / "terrain/prepared/profiles.json"
    terrain_profiles = json.loads(terrain_path.read_text()) if terrain_path.exists() else None
    verified_floors = set()
    unit_drafts = 0
    object_drafts = 0
    foreground_summary = None
    tree_summary = None
    tree_summary_path = ROOT/'terrain/trees/verification-summary.json'
    if tree_summary_path.exists():
        tree_summary = json.loads(tree_summary_path.read_text())
        assert tree_summary['normal_HD_object_cases']==1 and tree_summary['normal_HD_object_instances']==3
        assert tree_summary['records'][0]['GUI_verification_sha256']==sha(ROOT/'terrain/trees/desktop/verification-trees.json')
        proof_sources.append({'verification':str(tree_summary_path.relative_to(PROJECT)),'sha256':sha(tree_summary_path)})
    terrain_proof = ROOT / "terrain/runtime/desktop/verification-terrain.json"
    foreground_path = ROOT / "terrain/objects/verification-summary.json"
    if foreground_path.exists():
        foreground_summary = json.loads(foreground_path.read_text())
        terrain_proof = ROOT / "terrain/objects/desktop/verification-terrain.json"
        assert foreground_summary['source_sprite_count'] == 39
        assert foreground_summary['records'][0]['GUI_verification_sha256'] == sha(terrain_proof)
        proof_sources.append({'verification': str(foreground_path.relative_to(PROJECT)), 'sha256': sha(foreground_path)})
    object_generation = ROOT / 'terrain/art/object-generation.json'
    if object_generation.exists():
        for draft in json.loads(object_generation.read_text())['records']:
            assert sha(ROOT/'terrain/art'/draft['asset']) == draft['asset_sha256']
            object_drafts += 1
    if terrain_profiles:
        assert terrain_profiles["tool_sha256"] == sha(PROJECT / "tools/hd-terrain-prepare.py")
        assert terrain_profiles["source_count"] == 104 and terrain_profiles["reviewed_HD_count"] == 8
        generation = json.loads((ROOT / "terrain/art/generation.json").read_text())
        for draft in generation["records"]:
            if draft["name"].startswith("unit-"):
                assert sha(ROOT / "terrain/art" / draft["asset"]) == draft["sha256"]
                unit_drafts += 1
        assert unit_drafts == 10
        if terrain_proof.exists():
            proof = json.loads(terrain_proof.read_text())
            assert proof["verified"] and proof["receipt_sha256"] == sha(terrain_proof.with_name("desktop-terrain.json"))
            verified_floors = {d["index"] for c in proof["terrain_checks"] for d in c["active_rectangles"]}
            proof_sources.append({"verification": str(terrain_proof.relative_to(PROJECT)), "sha256": sha(terrain_proof)})
    records = []
    for source in decoded["records"]:
        name = source["name"]
        record = {"source": name, "kind": "OZM resource; atlas subdivision may be required",
                  "source_sha256": source["payload_sha256"], "source_file_offset": source["dat_file_offset"],
                  "decoder_status": source["status"], "normal_path_load_matches": source.get("normal_path_matches", 0),
                  "art_status": "missing", "runtime_status": "not integrated"}
        owner = name if name in direct else shared.get(source.get("indices_sha256"))
        if owner:
            record.update(art_status="prepared", art=direct[owner], art_source_resource=owner)
            record["reuse_basis"] = "own illustration" if owner == name else "identical decoded four-plane indices"
        if name in verified_story:
            record.update(runtime_status="normal player path verified", runtime_check=verified_story[name])
        if name in profiles:
            profile = profiles[name]
            assert record["art"]["sha256"] == profile["asset_sha256"]
            record.update(source_profile=profile)
            if name not in verified_story:
                record.update(runtime_status="reviewed source and common draw consumer integrated; normal path not yet sampled")
        if name == "VISUAL.OZM":
            path = PROJECT / "workplace/hd-prototype/scroll/scroll-hd-b-v1.png"
            record.update(art_status="partial", art=art_info(path, path.with_suffix(".prompt.txt")),
                          logical_region=[0, 250, 640, 150], runtime_status="scroll region verified separately; upper frame pending")
            frame_root = PROJECT / "workplace/hd-prototype/scroll/frame"
            frame_proof = frame_root / "desktop/verification-frame.json"
            if frame_proof.exists():
                proof = json.loads(frame_proof.read_text())
                frame_receipt = frame_root / "desktop/desktop-frame.json"
                assert proof["verified"] and proof["receipt_sha256"] == sha(frame_receipt)
                info = art_info(frame_root / "frame-hd-b-v2.png", path.with_suffix(".prompt.txt"))
                loaded = json.loads(frame_receipt.read_text())["desktop"]["assets"]["frame"]
                assert loaded["validation"] == "ready" and loaded["hd_sha256"] == info["sha256"]
                record.update(frame_art=info, frame_logical_region=[0, 0, 640, 250],
                              preserve_central_logical_region=[160, 50, 320, 150],
                              runtime_status="upper decorative frame and scroll verified; central scene is a separate layer",
                              frame_runtime_check=proof["frame_checks"], frame_proof_sha256=sha(frame_proof))
        if name == "WINDOW.OZM":
            window_root = ROOT / "window"
            path = window_root / "window-skin-hd-b-v1.png"
            if path.exists():
                record.update(kind="nine-tile window UI skin", tile_count=9, tile_size=[16, 16],
                              art_status="prepared", art=art_info(path, path.with_suffix(".prompt.txt")))
            proof_path = window_root / "desktop-window-modal/verification-window-modal.json"
            receipt_path = window_root / "desktop-window-modal/desktop-window-modal.json"
            if proof_path.exists():
                proof = json.loads(proof_path.read_text())
                assert proof["verified"] and proof["receipt_sha256"] == sha(receipt_path)
                loaded = json.loads(receipt_path.read_text())["desktop"]["assets"]["window"]
                assert loaded["validation"] == "ready" and loaded["hd_sha256"] == record["art"]["sha256"]
                record.update(runtime_status="normal player path verified; character selection, confirmation and status windows",
                              runtime_check=proof["window_checks"], runtime_proof_sha256=sha(proof_path),
                              source_extraction_sha256=sha(window_root / "window-extraction.json"))
                proof_sources.append({"verification": str(proof_path.relative_to(PROJECT)), "sha256": sha(proof_path)})
        if name == "MAP_CHIP.OZM" and terrain_profiles:
            record.update(art_status="partial", source_grid_counts={"floor": 104, "object": 26, "unit": 13},
                          approved_floor_art=8, generated_unit_drafts=unit_drafts, approved_unit_art=0,
                          generated_object_drafts=object_drafts, approved_object_art=1 if tree_summary else 0,
                          normal_object_indices_verified=[0] if tree_summary else [], normal_object_instances_verified=3 if tree_summary else 0,
                          normal_floor_indices_verified=sorted(verified_floors),
                          runtime_status="floor consumer with native index-zero foreground preservation integrated; HD sprites and whole atlas pending",
                          floor_profiles_sha256=sha(terrain_path),
                          source_extraction_sha256=sha(ROOT / "terrain/terrain-extraction.json"))
        records.append(record)
    for source, prepared in zip(extraction["records"], selected["records"]):
        index = source["index"]
        assert prepared["index"] == index
        path = ROOT / "kao/selected" / prepared["asset"]
        assert sha(path) == prepared["asset_sha256"]
        record = {"source": f"KAO.GRP:{index:02d}", "kind": "80x80 portrait", "index": index,
                  "source_sha256": source["raw_planar_sha256"], "source_png_sha256": source["source_png_sha256"],
                  "art_status": "prepared", "art": art_info(path, ROOT / "kao" / prepared["prompt"]),
                  "crop_provenance": prepared, "runtime_status": "common draw consumer implemented; native case not yet sampled"}
        if index in verified_kao:
            record.update(runtime_status="normal player path verified", runtime_check=verified_kao[index])
        records.append(record)
    pcx = originals["RAINEND.PCX"]
    records.append({"source": "RAINEND.PCX", "kind": "publisher logo and credits", "source_sha256": pcx["payload_sha256"],
                    "art_status": "missing", "runtime_status": "not integrated"})
    counts = {"OZM_resources": 112, "KAO_portraits": 81, "PCX_resources": 1,
              "generated_story_illustrations": len(direct),
              "OZM_sources_with_prepared_art_including_exact_reuse": sum(r["art_status"] == "prepared" for r in records[:112]),
              "OZM_partial_art": sum(r["art_status"] == "partial" for r in records[:112]),
              "prepared_KAO_portraits": 81, "native_story_cases_verified": len(verified_story),
              "native_KAO_cases_verified": len(verified_kao), "unsupported_OZM_decodes": 5,
              "reviewed_story_source_profiles": len(profiles),
              "approved_MAP_CHIP_floor_art": 8 if terrain_profiles else 0,
              "normal_MAP_CHIP_floor_cases_verified": len(verified_floors),
              "generated_MAP_CHIP_unit_drafts": unit_drafts,
              "generated_MAP_CHIP_object_drafts": object_drafts,
              "native_MAP_CHIP_sprite_source_profiles": 39 if foreground_summary else 0,
              "approved_MAP_CHIP_object_art": 1 if tree_summary else 0,
              "normal_MAP_CHIP_object_cases_verified": 1 if tree_summary else 0,
              "normal_MAP_CHIP_object_instances_verified": 3 if tree_summary else 0}
    result = {"classification": "local research coverage ledger; not a production asset-pack schema",
              "whole_game_HD_complete": False,
              "count_limit": "resource counts do not count atlas tiles, animation frames or UI pieces; no completion percentage",
              "dat_sha256": decoded["dat_sha256"], "tool_sha256": sha(Path(__file__)), "pillow_version": pillow_version,
              "counts": counts, "runtime_proof_sources": proof_sources, "records": records}
    (output / "art-coverage.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(counts, ensure_ascii=False))


if __name__ == "__main__":
    main()
