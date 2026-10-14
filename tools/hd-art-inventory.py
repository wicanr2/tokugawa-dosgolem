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
    ui_path = ROOT/'art/ui-generation.json'
    ui_drafts = {r['source']:r for r in json.loads(ui_path.read_text())['records']} if ui_path.exists() else {}
    weather_path = ROOT/'art/weather-generation.json'
    weather = {r['source']:r for r in json.loads(weather_path.read_text())['records'] if r['version'] == 2} if weather_path.exists() else {}
    weather_profile_path = ROOT/'graphics/weather-profiles.json'
    weather_profiles = {r['source']:r for r in json.loads(weather_profile_path.read_text())['records']} if weather_profile_path.exists() else {}
    weather_proof_path = ROOT/'graphics/verification-summary.json'
    weather_proof = json.loads(weather_proof_path.read_text()) if weather_proof_path.exists() else None
    opening_path = ROOT/'art/opening-generation.json'
    opening = {r['source']:r for r in json.loads(opening_path.read_text())['records']} if opening_path.exists() else {}
    frame_summary_path = ROOT/'battle-frame/verification-summary.json'
    frame_summary = json.loads(frame_summary_path.read_text()) if frame_summary_path.exists() else None
    if frame_summary:
        assert frame_summary['records'][0]['GUI_verification_sha256'] == sha(ROOT/'battle-frame/desktop/verification-battle-frame.json')
        proof_sources_frame = {'verification':str(frame_summary_path.relative_to(PROJECT)),'sha256':sha(frame_summary_path)}
    direct = {}
    for record in decoded["records"]:
        if record['name'] in ui_drafts or record['name'] in weather or record['name'] in opening:
            continue
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
    if frame_summary:
        proof_sources.append(proof_sources_frame)
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
    material_profiles_path = ROOT/'terrain/materials/profiles.json'
    if material_profiles_path.exists():
        expanded = json.loads(material_profiles_path.read_text())
        assert expanded['source_count'] == expanded['reviewed_HD_count'] == 104
        assert expanded['tool_sha256'] == sha(PROJECT/'tools/hd-terrain-materials.py')
        assert expanded['baseline_sha256'] == sha(terrain_path)
        terrain_profiles = expanded
        materials_proof_path = ROOT/'terrain/materials/verification-summary.json'
        if materials_proof_path.exists():
            materials_proof = json.loads(materials_proof_path.read_text())
            assert materials_proof['verified'] and materials_proof['reviewed_source_count'] == 104
            assert materials_proof['profiles_sha256'] == sha(material_profiles_path)
            assert materials_proof['tool_sha256'] == sha(PROJECT/'tools/verify-hd-terrain-materials.py')
            proof_sources.append({'verification':str(materials_proof_path.relative_to(PROJECT)),'sha256':sha(materials_proof_path)})
    approved_floor_count = sum(bool(r['asset']) for r in terrain_profiles['records']) if terrain_profiles else 0
    verified_floors = set()
    unit_drafts = 0
    object_drafts = 0
    foreground_summary = None
    tree_summary = None
    scene_summary = None
    tree_summary_path = ROOT/'terrain/trees/verification-summary.json'
    if tree_summary_path.exists():
        tree_summary = json.loads(tree_summary_path.read_text())
        assert tree_summary['normal_HD_object_cases']==1 and tree_summary['normal_HD_object_instances']==3
        assert tree_summary['records'][0]['GUI_verification_sha256']==sha(ROOT/'terrain/trees/desktop/verification-trees.json')
        proof_sources.append({'verification':str(tree_summary_path.relative_to(PROJECT)),'sha256':sha(tree_summary_path)})
    backdrop_summary_path = ROOT/'terrain/backgrounds/verification-summary.json'
    if backdrop_summary_path.exists():
        tree_summary = json.loads(backdrop_summary_path.read_text())
        assert tree_summary['normal_HD_object_cases'] == 1 and tree_summary['normal_HD_object_instances'] == 4
        assert tree_summary['background_verification_sha256'] == sha(ROOT/'terrain/backgrounds/background-verification.json')
        assert tree_summary['records'][0]['GUI_verification_sha256'] == sha(ROOT/'terrain/backgrounds/desktop/verification-trees.json')
        proof_sources.append({'verification':str(backdrop_summary_path.relative_to(PROJECT)),'sha256':sha(backdrop_summary_path)})
    terrain_proof = ROOT / "terrain/runtime/desktop/verification-terrain.json"
    scene_summary_path = ROOT/'terrain/scene/verification-summary.json'
    if scene_summary_path.exists():
        scene_summary = json.loads(scene_summary_path.read_text())
        assert scene_summary['approved_unit_indices'] == [1,3] and scene_summary['normal_HD_floor_draws'] == 67
        assert scene_summary['records'][0]['GUI_verification_sha256'] == sha(ROOT/'terrain/scene/desktop/verification-scene.json')
        proof_sources.append({'verification':str(scene_summary_path.relative_to(PROJECT)),'sha256':sha(scene_summary_path)})
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
    expanded_objects = ROOT/'terrain/art/object-generation-expanded.json'
    if expanded_objects.exists():
        for draft in json.loads(expanded_objects.read_text())['records']:
            assert sha(ROOT/'terrain/art'/draft['asset']) == draft['asset_sha256']
            object_drafts += 1
    if terrain_profiles:
        assert terrain_profiles["tool_sha256"] == sha(PROJECT / ("tools/hd-terrain-materials.py" if material_profiles_path.exists() else "tools/hd-terrain-prepare.py"))
        expected_floor_count = 104 if material_profiles_path.exists() else 8
        assert terrain_profiles["source_count"] == 104 and terrain_profiles["reviewed_HD_count"] == expected_floor_count
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
    generated_units, generated_objects = unit_drafts, object_drafts
    approved_units = scene_summary['approved_unit_indices'] if scene_summary else []
    reviewed_unit_indices = approved_units
    unit_profiles_path = ROOT/'terrain/units/profiles.json'
    if unit_profiles_path.exists():
        unit_profiles = json.loads(unit_profiles_path.read_text())
        assert unit_profiles['tool_sha256'] == sha(PROJECT/'tools/hd-unit-prepare.py')
        assert unit_profiles['source_slots'] == 13 and unit_profiles['nonempty_sources'] == 10 and unit_profiles['empty_indices'] == [10,11,12]
        reviewed_unit_indices = [r['index'] for r in unit_profiles['records'] if r['asset']]
        assert reviewed_unit_indices == list(range(10))
        unit_proof_path = ROOT/'terrain/units/verification-summary.json'
        if unit_proof_path.exists():
            unit_proof = json.loads(unit_proof_path.read_text())
            assert unit_proof['verified'] and unit_proof['profiles_sha256'] == sha(unit_profiles_path)
            assert unit_proof['tool_sha256'] == sha(PROJECT/'tools/verify-hd-units.py')
            proof_sources.append({'verification':str(unit_proof_path.relative_to(PROJECT)),'sha256':sha(unit_proof_path)})
    unit_drafts -= len(reviewed_unit_indices)
    reviewed_object_indices = [0] if tree_summary else []
    object_profiles_path = ROOT/'terrain/objects/expanded/profiles.json'
    if object_profiles_path.exists():
        object_profiles = json.loads(object_profiles_path.read_text())
        assert object_profiles['tool_sha256'] == sha(PROJECT/'tools/hd-object-prepare.py')
        assert object_profiles['source_slots'] == 26 and object_profiles['empty_indices'] == [24,25]
        reviewed_object_indices = object_profiles['approved_indices']
        object_proof_path = object_profiles_path.parent/'verification-summary.json'
        if object_proof_path.exists():
            object_proof = json.loads(object_proof_path.read_text())
            assert object_proof['verified'] and object_proof['profiles_sha256'] == sha(object_profiles_path)
            assert object_proof['tool_sha256'] == sha(PROJECT/'tools/verify-hd-objects.py')
            proof_sources.append({'verification':str(object_proof_path.relative_to(PROJECT)),'sha256':sha(object_proof_path)})
    object_drafts -= len(reviewed_object_indices)
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
                          approved_floor_art=approved_floor_count, generated_unit_drafts=unit_drafts, approved_unit_art=len(reviewed_unit_indices),
                          reviewed_unit_source_indices=reviewed_unit_indices, original_empty_unit_indices=[10,11,12],
                          normal_unit_indices_verified=approved_units, normal_unit_instances_verified=2 if scene_summary else 0,
                          generated_object_drafts=object_drafts, approved_object_art=len(reviewed_object_indices),
                          reviewed_object_source_indices=reviewed_object_indices, original_empty_object_indices=[24,25],
                          normal_object_indices_verified=[0] if tree_summary else [], normal_object_instances_verified=tree_summary['normal_HD_object_instances'] if tree_summary else 0,
                          normal_floor_indices_verified=sorted(verified_floors),
                          runtime_status=f"all 104 floor and 10 nonempty unit profiles, {len(reviewed_object_indices)} reviewed object sources integrated; original blank sources preserved; normal six-floor scene, object-000 and units-001/003 sampled; wall geometry drafts pending" if approved_floor_count == 104 and len(reviewed_unit_indices) == 10 else "full-floor journal and reviewed foreground sources integrated; remaining atlas art pending",
                          normal_floor_draws_verified=67 if scene_summary else 65 if foreground_summary else 0,
                          floor_profiles_sha256=sha(material_profiles_path if approved_floor_count == 104 else terrain_path),
                          source_extraction_sha256=sha(ROOT / "terrain/terrain-extraction.json"))
        if name in ui_drafts:
            draft = ui_drafts[name]
            path = ROOT/'art'/draft['asset']
            assert sha(path) == draft['sha256']
            record.update(kind="battle interface frame", art_status="draft",
                          art=art_info(path, ROOT/'art'/draft['prompt']),
                          runtime_status="visually reviewed; geometry and integration pending")
            if name == 'MAPGAMEN.OZM' and frame_summary:
                geometry_path = ROOT/'battle-frame/battle-frame-geometry-v2.json'
                geometry = json.loads(geometry_path.read_text())
                assert frame_summary['geometry_sha256'] == sha(geometry_path)
                prepared = ROOT/'battle-frame'/geometry['prepared_asset']
                assert sha(prepared) == frame_summary['asset_sha256']
                record.update(art_status='prepared',art=art_info(prepared,ROOT/'art'/draft['prompt']),
                              runtime_status='normal pre-battle modal and first battle verified; whole-region weather/crest guards, indexed foreground and joint HD scene underlay',
                              runtime_proof_sha256=sha(frame_summary_path),geometry_sha256=sha(geometry_path))
        if name in opening:
            painting = opening[name]
            path = ROOT/'art'/painting['asset']
            assert sha(path) == painting['sha256']
            record.update(kind='opening panoramic illustration', art_status='draft',
                          art=art_info(path,ROOT/'art'/painting['prompt']),runtime_status=painting['status'])
        if name in weather:
            painting = weather[name]
            path = ROOT/'art'/painting['asset']
            assert sha(path) == painting['sha256']
            record.update(kind='opaque mountain and cloud weather painting', art_status='draft',
                          art=art_info(path, ROOT/'art'/painting['prompt']),
                          runtime_status='v2 rock spires visually reviewed; source and integration pending')
            if name in weather_profiles:
                profile = weather_profiles[name]
                assert profile['asset_sha256'] == painting['sha256']
                record.update(art_status='prepared',source_profile=profile,
                              runtime_status='exact source and common opaque draw consumer integrated; normal path not yet sampled')
            if weather_proof and name == 'WEATHER1.OZM':
                assert weather_proof['verified'] and weather_proof['profiles_sha256'] == sha(weather_profile_path)
                record.update(runtime_status='normal first-battle weather HD, original toggle and missing asset verified',
                              runtime_proof_sha256=sha(weather_proof_path))
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
              "generated_UI_drafts": len(ui_drafts)-(1 if frame_summary else 0),
              "generated_UI_art": len(ui_drafts),"approved_UI_art":1 if frame_summary else 0,
              "OZM_sources_with_prepared_art_including_exact_reuse": sum(r["art_status"] == "prepared" for r in records[:112]),
              "OZM_partial_art": sum(r["art_status"] == "partial" for r in records[:112]),
              "prepared_KAO_portraits": 81, "native_story_cases_verified": len(verified_story),
              "native_KAO_cases_verified": len(verified_kao), "unsupported_OZM_decodes": 5,
              "reviewed_story_source_profiles": len(profiles),
              "generated_weather_sources":len(weather), "approved_weather_sources":len(weather_profiles),
              "normal_weather_sources_verified":1 if weather_proof else 0,
              "generated_opening_drafts":len(opening),
              "approved_MAP_CHIP_floor_art": approved_floor_count,
              "normal_MAP_CHIP_floor_cases_verified": len(verified_floors),
              "generated_MAP_CHIP_unit_drafts": unit_drafts,
              "generated_MAP_CHIP_unit_art": generated_units,
              "approved_MAP_CHIP_unit_art": len(reviewed_unit_indices),
              "normal_MAP_CHIP_unit_cases_verified": len(approved_units),
              "normal_MAP_CHIP_unit_instances_verified": 2 if scene_summary else 0,
              "generated_MAP_CHIP_object_drafts": object_drafts,
              "generated_MAP_CHIP_object_art": generated_objects,
              "native_MAP_CHIP_sprite_source_profiles": 39 if foreground_summary else 0,
              "approved_MAP_CHIP_object_art": len(reviewed_object_indices),
              "normal_MAP_CHIP_object_cases_verified": 1 if tree_summary else 0,
              "normal_MAP_CHIP_object_instances_verified": tree_summary['normal_HD_object_instances'] if tree_summary else 0}
    result = {"classification": "local research coverage ledger; not a production asset-pack schema",
              "whole_game_HD_complete": False,
              "count_limit": "resource counts do not count atlas tiles, animation frames or UI pieces; no completion percentage",
              "dat_sha256": decoded["dat_sha256"], "tool_sha256": sha(Path(__file__)), "pillow_version": pillow_version,
              "counts": counts, "runtime_proof_sources": proof_sources, "records": records}
    (output / "art-coverage.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(counts, ensure_ascii=False))


if __name__ == "__main__":
    main()
