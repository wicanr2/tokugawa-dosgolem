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


def recorded_verifier(name, expected, snapshot):
    path = PROJECT / 'tools' / name
    if sha(path) != expected:
        path = ROOT / snapshot
    assert sha(path) == expected, (name, path, expected)
    return {'tool': name, 'sha256': expected, 'source': str(path.relative_to(PROJECT))}


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
    verifier_sources = []
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
    remaining_path = ROOT/'art/remaining-generation-v1.json'
    remaining_art = {r['source']:r for r in json.loads(remaining_path.read_text())['records']} if remaining_path.exists() else {}
    pending_generation_path = ROOT/'art/remaining-full-generation-v1.json'
    pending_generation = {r['source']:r for r in json.loads(pending_generation_path.read_text())['records']} if pending_generation_path.exists() else {}
    pending_profile_path = ROOT/'remaining/profiles.json'
    pending_profiles = {}
    pending_art_proof = None
    pending_runtime = {}
    pending_runtime_proof = None
    pending_runtime_path = ROOT/'remaining/runtime-profiles-v1.json'
    pending_runtime_proof_path = ROOT/'remaining/runtime-verification-summary.json'
    if pending_profile_path.exists():
        pending_selected = json.loads(pending_profile_path.read_text())
        assert pending_selected['tool_sha256'] == sha(PROJECT/'tools/hd-remaining-prepare.py')
        pending_profiles = {r['name']:r for r in pending_selected['records']}
        proof_path = ROOT/'remaining'/pending_selected['selected_draft']/'art-verification.json'
        pending_art_proof = json.loads(proof_path.read_text())
        assert pending_art_proof['verified'] and pending_art_proof['source_count'] == 20
        assert pending_selected['art_verification_sha256'] == sha(proof_path)
        assert pending_art_proof['tool_sha256'] == sha(PROJECT/'tools/verify-hd-remaining.py')
    if pending_runtime_path.exists():
        pending_runtime = json.loads(pending_runtime_path.read_text())
        assert pending_runtime['go_profiles_sha256'] == sha(PROJECT/'tools/hd-graphics-remaining-profiles.go')
        assert pending_runtime['art_profiles_sha256'] == sha(pending_profile_path)
        assert set(pending_runtime['registered_source_names']) == set(pending_profiles)
    if pending_runtime_proof_path.exists():
        pending_runtime_proof = json.loads(pending_runtime_proof_path.read_text())
        assert pending_runtime_proof['verified'] and pending_runtime_proof['runtime_profiles_sha256'] == sha(pending_runtime_path)
        verifier_sources.append(recorded_verifier('verify-hd-remaining-runtime.py', pending_runtime_proof['tool_sha256'],
                                                  'remaining/desktop-title/verify-hd-remaining-runtime.py'))
    static_profiles = {}
    for relative, tool in [('graphics/static/profiles.json', 'hd-background-prepare.py'),
                           ('graphics/static/logo/profiles.json', 'hd-logo-prepare.py')]:
        path = ROOT/relative
        if path.exists():
            data = json.loads(path.read_text())
            assert data['tool_sha256'] == sha(PROJECT/'tools'/tool)
            for profile in data.get('records', [data.get('record')]):
                assert profile and profile['name'] not in static_profiles
                static_profiles[profile['name']] = profile
    static_proof_path = ROOT/'graphics/static/verification-summary.json'
    static_proof = json.loads(static_proof_path.read_text()) if static_proof_path.exists() else None
    if static_proof:
        assert static_proof['verified']
        verifier_sources.append(recorded_verifier('verify-hd-background.py', static_proof['tool_sha256'],
                                                  'graphics/static/desktop-logo/palette/ready/verify-hd-background.py'))
        assert static_proof['profiles_sha256'] == sha(ROOT/'graphics/static/profiles.json')
        assert static_proof['logo_profiles_sha256'] == sha(ROOT/'graphics/static/logo/profiles.json')
        assert set(static_proof['reviewed_sources']) == set(static_profiles)
    history = ROOT/'remaining/history'
    full_profile_path = history/'art-v2/profiles.json'
    full_profile = json.loads(full_profile_path.read_text())['record'] if full_profile_path.exists() else None
    opening_proof_path = history/'verification-summary.json'
    opening_proof = json.loads(opening_proof_path.read_text()) if opening_proof_path.exists() else None
    prefix_art = {}
    for name in ['E-4-generation-v1.json', 'B-6-generation-v1.json']:
        path = ROOT/'art'/name
        if path.exists():
            for record in json.loads(path.read_text())['records']:
                for source in record['aliases']:
                    prefix_art[source] = record
    prefix_story_path = ROOT/'remaining/partial-story/profiles.json'
    prefix_story = {}
    if prefix_story_path.exists():
        data = json.loads(prefix_story_path.read_text())
        review = json.loads((prefix_story_path.parent/'review.json').read_text())
        assert data['tool_sha256'] == sha(PROJECT/'tools/hd-prefix-prepare.py')
        assert review['profiles_sha256'] == sha(prefix_story_path)
        prefix_story = {r['resource']: r for r in data['records']}
        assert set(prefix_story) == set(review['approved_resources'])
    prefix_runtime_path = ROOT/'remaining/partial-story/verification-summary.json'
    prefix_runtime = json.loads(prefix_runtime_path.read_text()) if prefix_runtime_path.exists() else None
    if prefix_runtime:
        assert prefix_runtime['verified']
        verifier_sources.append(recorded_verifier('verify-hd-prefix-runtime.py', prefix_runtime['tool_sha256'],
                                                  'remaining/partial-story/desktop/source-snapshot/tools/verify-hd-prefix-runtime.py'))
        assert prefix_runtime['profiles_sha256'] == sha(prefix_story_path)
    if opening_proof:
        assert opening_proof['verified'] and opening_proof['tool_sha256'] == sha(PROJECT/'tools/verify-hd-opening-summary.py')
        assert opening_proof['art_profiles_sha256'] == sha(full_profile_path)
    partial_proof_path = history/'partial/verification-summary.json'
    partial_proof = json.loads(partial_proof_path.read_text()) if partial_proof_path.exists() else None
    if partial_proof:
        assert partial_proof['verified'] and partial_proof['tool_sha256'] == sha(PROJECT/'tools/verify-hd-opening-partial-captures.py')
        assert partial_proof['capture_proof_sha256'] == sha(history/'partial/capture-verification.json')
    shift_proof_path = history/'scrolling/verification-summary.json'
    shift_proof = json.loads(shift_proof_path.read_text()) if shift_proof_path.exists() else None
    if shift_proof:
        assert shift_proof['verified'] and shift_proof['tool_sha256'] == sha(PROJECT/'tools/verify-hd-opening-shift-summary.py')
        assert shift_proof['source_proof_sha256'] == sha(history/'scrolling/source-verification.json')
    frame_summary_path = ROOT/'battle-frame/verification-summary.json'
    frame_summary = json.loads(frame_summary_path.read_text()) if frame_summary_path.exists() else None
    if frame_summary:
        assert frame_summary['records'][0]['GUI_verification_sha256'] == sha(ROOT/'battle-frame/desktop/verification-battle-frame.json')
        proof_sources_frame = {'verification':str(frame_summary_path.relative_to(PROJECT)),'sha256':sha(frame_summary_path)}
    direct = {}
    for record in decoded["records"]:
        if record['name'] in ['OP-FULL.OZM', 'B-6.OZM', 'B-7.OZM', 'C-10.OZM', 'E-4.OZM']:
            continue
        if record['name'] in ui_drafts or record['name'] in weather or record['name'] in opening or record['name'] in remaining_art or record['name'] in static_profiles or record['name'] in pending_generation or record['name'] in pending_profiles:
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
    if prefix_runtime:
        proof_sources.append({'verification':str(prefix_runtime_path.relative_to(PROJECT)), 'sha256':sha(prefix_runtime_path)})
    if opening_proof:
        proof_sources.append({'verification':str(opening_proof_path.relative_to(PROJECT)), 'sha256':sha(opening_proof_path)})
    if partial_proof:
        proof_sources.append({'verification':str(partial_proof_path.relative_to(PROJECT)), 'sha256':sha(partial_proof_path)})
    if shift_proof:
        proof_sources.append({'verification':str(shift_proof_path.relative_to(PROJECT)), 'sha256':sha(shift_proof_path)})
    if pending_runtime_proof:
        proof_sources.append({'verification':str(pending_runtime_proof_path.relative_to(PROJECT)),'sha256':sha(pending_runtime_proof_path)})
    if static_proof:
        proof_sources.append({'verification':str(static_proof_path.relative_to(PROJECT)), 'sha256':sha(static_proof_path)})
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

    if partial_proof:
        captures = json.loads((history/'partial/capture-verification.json').read_text())
        complete = json.loads((history/'partial/full/desktop-full.json').read_text())
        for name in partial_proof['normal_story_sources']:
            check = next(r for r in captures['runs']['partial']['checks'] if r['source'] == name and r['mode'] == 'hd')
            assert complete['desktop']['assets']['story_portraits'][name]['hd_sha256'] == direct[name]['sha256']
            verified_story[name] = check
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
    wall_profiles_path = ROOT/'terrain/objects/walls/profiles.json'
    if wall_profiles_path.exists():
        wall_profiles = json.loads(wall_profiles_path.read_text())
        assert wall_profiles['tool_sha256'] == sha(PROJECT/'tools/hd-wall-prepare.py') == sha(wall_profiles_path.parent/'profiles-source.txt')
        assert wall_profiles['approved_indices'] == list(range(24)) and wall_profiles['empty_indices'] == [24,25]
        object_drafts += len(reviewed_object_indices)-24
        reviewed_object_indices = wall_profiles['approved_indices']
        wall_proof_path = wall_profiles_path.parent/'verification-summary.json'
        if wall_proof_path.exists():
            wall_proof = json.loads(wall_proof_path.read_text())
            assert wall_proof['verified'] and wall_proof['profiles_sha256'] == sha(wall_profiles_path)
            assert wall_proof['tool_sha256'] == sha(PROJECT/'tools/verify-hd-walls.py')
            proof_sources.append({'verification':str(wall_proof_path.relative_to(PROJECT)),'sha256':sha(wall_proof_path)})
    combat_path = ROOT/'remaining/consumers/battle/r17/verification-summary.json'
    combat = json.loads(combat_path.read_text()) if combat_path.exists() else None
    combat_hd_sources = set()
    if combat:
        assert combat['result'] == 'PASS' and combat['native_steps'] == 600000000
        assert len(combat['native_fields_equal']) == 17
        assert combat['mixed_OR_native_pixels_preserved'] == 18
        assert combat['original_black_foreground_pixels_preserved'] == 3012
        assert combat['missing_ground_native_output_pixels'] == 82300
        verifier_sources.append(recorded_verifier('verify-hd-combat.py', combat['tool_sha256'],
                                                  'remaining/consumers/battle/r17/verify-hd-combat.py'))
        for case, proof in combat['cases'].items():
            assert proof['receipt_sha256'] == sha(combat_path.parent/case/'receipt.json')
        state = json.loads((combat_path.parent/'desktop-animation/state-action-2.json').read_text())
        combat_hd_sources = {r['resource'] for r in state['graphics'] if r['mode'] == 'hd'}
        assert combat_hd_sources == {'BATTLE02.OZM', 'BATTLE.OZM', 'JIMEN_0.OZM', 'JIMEN_1.OZM', 'ANIME08.OZM'}
        proof_sources.append({'verification': str(combat_path.relative_to(PROJECT)), 'sha256': sha(combat_path)})
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
            if approved_floor_count == 104 and len(reviewed_unit_indices) == 10 and len(reviewed_object_indices) == 24:
                data = (ROOT/'decoded/MAP_CHIP.OZM.idx').read_bytes()
                assert len(data) == 640*400 and not any(data[y*640+x] for y in range(400) for x in range(624,640))
                record.update(art_status="prepared", wall_profiles_sha256=sha(wall_profiles_path),
                              complete_art_scope="104 floor, 24 nonempty object and 10 nonempty unit sources; five blank sprite cells and zero right padding preserved",
                              runtime_status="complete atlas art registered with common consumers; normal six-floor scene, object-000 and units-001/003 sampled; remaining sources not individually sampled")
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
        if name in remaining_art:
            painting = remaining_art[name]
            path = ROOT/'art'/painting['asset']
            assert sha(path) == painting['sha256']
            record.update(art_status='draft',art=art_info(path,ROOT/'art'/painting['prompt']),
                          runtime_status='source-referenced art draft; black padding, geometry and consumer registration pending')
        if name in static_profiles:
            profile = static_profiles[name]
            path = PROJECT/'assets/hd-b/art'/profile['asset']
            assert sha(path) == profile['asset_sha256']
            assert profile['source']['indices_sha256'] == source['indices_sha256']
            record.update(kind='calibrated static illustration or publisher wordmark', art_status='prepared',
                          art=art_info(path,ROOT/'art'/profile['prompt']), source_profile=profile,
                          runtime_status='reviewed source and common opaque consumer integrated; normal path not yet sampled')
            if static_proof and name in static_proof['normal_new_source_names']:
                record.update(runtime_status='normal idle opening HD and original toggles verified; complete 200M original state comparison',
                              runtime_proof_sha256=sha(static_proof_path))
        if name in pending_generation:
            generation = pending_generation[name]
            path = ROOT/'art'/generation['asset']
            assert sha(path) == generation['sha256']
            record.update(art_status='draft',art=art_info(path,ROOT/'art'/generation['prompt']),
                          runtime_status='generated art; source geometry and runtime registration pending')
        if name in pending_profiles:
            profile = pending_profiles[name]
            path = ROOT/'remaining'/profile['prepared']
            assert sha(path) == profile['asset_sha256']
            assert source['indices_sha256'] == profile['source']['indices_sha256']
            record.update(art_status='prepared',art=art_info(path,ROOT/'art'/profile['inputs'][0]['prompt']),
                          source_profile=profile,art_proof_sha256=pending_selected['art_verification_sha256'],
                          runtime_status='reviewed calibrated art only; new runtime consumer registration pending')
            if name in pending_runtime.get('registered_source_names',[]):
                record.update(runtime_status='source and shared opaque consumer registered; fixed native foreground and later observed occlusion; normal case not yet sampled',
                              runtime_profiles_sha256=sha(pending_runtime_path))
            if pending_runtime_proof and name in pending_runtime_proof['normal_new_sources']:
                record.update(runtime_status='normal title/menu HD, individual missing asset and 200M original state comparison verified',
                              runtime_proof_sha256=sha(pending_runtime_proof_path))
            if partial_proof and name in partial_proof['normal_graphics_sources']:
                record.update(runtime_status='ordinary history partial draw, exact source and palette, HD/original/restored and missing asset verified; OP9 normal HD X11 samples include four eyes and two of five portraits',
                              runtime_proof_sha256=sha(partial_proof_path))
        if name == 'OP-FULL.OZM' and full_profile:
            path = PROJECT/'assets/hd-b/art'/full_profile['asset']
            assert sha(path) == full_profile['asset_sha256']
            record.update(art_status='prepared', kind='runtime-observed complete opening illustration',
                          art=art_info(path, ROOT/'art'/full_profile['prompt']), source_profile=full_profile,
                          runtime_status='exact runtime source and full-copy consumer registered; normal HD verification pending')
        if name in prefix_art:
            painting = prefix_art[name]
            path = ROOT/'art'/painting['asset']
            assert sha(path) == painting['sha256']
            record.update(art_status='draft', art=art_info(path,ROOT/'art'/painting['prompt']),
                          art_source_resource=painting['source'], source_scope='47992 known pixels; eight unknown pixels remain excluded',
                          runtime_status='generated from known-region reference; geometry and partial-source qualification pending')
        if name in prefix_story:
            prepared = prefix_story[name]
            path = PROJECT/'assets/hd-b/art'/prepared['asset']
            assert sha(path) == prepared['asset_sha256']
            assert sha(prefix_story_path.parent/prepared['known_indices']) == prepared['known_indices_sha256']
            record.update(art_status='prepared', art=art_info(path,ROOT/'art'/prepared['prompt']), source_profile=prepared,
                          source_scope='47992 known art pixels; eight source-unknown pixels remain native; static decode still incomplete',
                          runtime_status='known-region common story consumer implemented; runtime regression pending')
            if prefix_runtime:
                record.update(runtime_status='known-region source/shared-consumer tests and existing history/battle regression verified; this scene not normally sampled',
                              runtime_proof_sha256=sha(prefix_runtime_path))
        if opening_proof and name in opening_proof['normal_new_sources']:
            record.update(runtime_status='ordinary history HD/original/restored, six fade-in and two palette-cycle samples verified; full 600M state comparison',
                          runtime_proof_sha256=sha(opening_proof_path))
        if shift_proof and name in shift_proof['normal_static_sources']:
            record.update(runtime_status='ordinary history scrolling HD/original/restored and unknown tail preservation sampled; normal/missing full 600M state comparison',
                          runtime_proof_sha256=sha(shift_proof_path))
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
    pcx_record = {"source": "RAINEND.PCX", "kind": "publisher logo and credits", "source_sha256": pcx["payload_sha256"],
                  "art_status": "missing", "runtime_status": "not integrated"}
    pcx_draft_path = ROOT/'art/RAINEND-generation-v1.json'
    if pcx_draft_path.exists():
        draft = json.loads(pcx_draft_path.read_text())
        assert draft['source_payload_sha256'] == pcx['payload_sha256']
        path = ROOT/'art'/draft['asset']
        assert sha(path) == draft['asset_sha256'] and sha(ROOT/'art'/draft['prompt']) == draft['prompt_sha256']
        pcx_record.update(art_status='draft',art=art_info(path,ROOT/'art'/draft['prompt']),
                          runtime_status='publisher logo draft; caller, display mode and original copyright composition pending',
                          native_text_preserve_rect=draft['planned_native_preserve_rect'])
        publisher_proof_path = Path('/project/workplace/desktop-prototype/publisher/verification-summary.json')
        if publisher_proof_path.exists():
            proof = json.loads(publisher_proof_path.read_text())
            assert proof['result'] == 'PASS' and {item['mode'] for item in proof['mark']} == {'hd', 'missing'}
            assert proof['independent_source']['payload_sha256'] == pcx['payload_sha256']
            pcx_record.update(art_status='prepared',
                              runtime_status='standalone original MARK desktop HD verified; game startup chaining pending',
                              runtime_check={'summary_sha256': sha(publisher_proof_path),
                                             'scope': 'standalone MARK native entry, display, HD switching, fallback and exit',
                                             'game_startup_integrated': False})
            chain_path = Path('/project/workplace/desktop-prototype/startup-chain/verification-summary.json')
            if chain_path.exists():
                chain = json.loads(chain_path.read_text())
                assert chain['result'] == 'PASS' and chain['game_startup_integrated']
                assert set(chain['normal_last_displays']) == {'original', 'hd'}
                assert {item['mode'] for item in chain['cases']} == {'original', 'hd', 'cancel'}
                pcx_record.update(runtime_status='default startup MARK HD and last display setting verified through DX5 first battle',
                                  runtime_check={'summary_sha256': sha(chain_path),
                                                 'standalone_summary_sha256': sha(publisher_proof_path),
                                                 'scope': 'default selection, MARK original/HD, native exit, DX5 first battle and window close',
                                                 'game_startup_integrated': True})
    for record in records:
        if record['source'] in combat_hd_sources:
            record.update(runtime_status='normal battle HD/original/restored and native state verified; composite masks, original text, individual missing ground and region restoration verified; top unit icons remain original',
                          runtime_proof_sha256=sha(combat_path))
    records.append(pcx_record)
    runtime_tails = []
    tail_path = ROOT / 'remaining/source-boundary/OP-FULL-runtime/receipt.json'
    if tail_path.exists():
        tail = json.loads(tail_path.read_text())
        assert tail['source'] == 'OP-FULL.OZM' and tail['matching_candidates'] == [170]
        assert sha(tail_path.parent / 'OP-FULL-runtime.idx') == tail['indexed_sha256']
        target = next(record for record in records if record['source'] == tail['source'])
        target['runtime_source_observation'] = {
            'classification': 'one natural history-menu RAM tail uniquely matched; static decode still unsupported',
            'receipt_sha256': sha(tail_path), 'indexed_sha256': tail['indexed_sha256'],
            'normal_load_step': tail['normal_load_event']['step'],
            'art_and_HD_consumer_verified': bool(opening_proof and tail['source'] in opening_proof['normal_new_sources']),
        }
        runtime_tails.append(tail['source'])
    counts = {"OZM_resources": 112, "KAO_portraits": 81, "PCX_resources": 1,
              "unsupported_OZM_with_one_runtime_tail_observation": len(runtime_tails),
              "generated_story_illustrations": len(direct),
              "generated_UI_drafts": len(ui_drafts)-(1 if frame_summary else 0),
              "generated_UI_art": len(ui_drafts),"approved_UI_art":1 if frame_summary else 0,
              "OZM_sources_with_prepared_art_including_exact_reuse": sum(r["art_status"] == "prepared" for r in records[:112]),
              "OZM_partial_art": sum(r["art_status"] == "partial" for r in records[:112]),
              "prepared_KAO_portraits": 81, "native_story_cases_verified": len(verified_story),
              "native_KAO_cases_verified": len(verified_kao), "unsupported_OZM_decodes": 5,
              "reviewed_story_source_profiles": len(profiles)+len(prefix_story),
              "generated_weather_sources":len(weather), "approved_weather_sources":len(weather_profiles),
              "normal_weather_sources_verified":1 if weather_proof else 0,
              "generated_opening_drafts":len(set(opening)-set(static_profiles)),
              "generated_remaining_OZM_drafts":len(set(remaining_art)-set(static_profiles))+len(set(pending_generation)-set(pending_profiles)),
              "approved_remaining_art_sources":len(pending_profiles),
              "remaining_animation_atlases_prepared":8 if pending_art_proof else 0,
              "remaining_animation_frames_prepared":48 if pending_art_proof else 0,
              "remaining_sources_runtime_registered":len(pending_runtime.get('registered_source_names',[])),
              "remaining_sources_normal_verified":len(set(pending_runtime_proof['normal_new_sources'] if pending_runtime_proof else []) | set(partial_proof['normal_graphics_sources'] if partial_proof else []) | (combat_hd_sources & set(pending_profiles))),
              "generated_partial_source_art_resources":len(prefix_art),
              "prepared_partial_source_story_resources":len(prefix_story),
              "partial_source_story_resources_runtime_registered":len(prefix_runtime['registered_sources']) if prefix_runtime else 0,
              "approved_static_sources":len(static_profiles),
              "normal_static_sources_verified":len(set(static_proof['normal_new_source_names'] if static_proof else []) | (set(opening_proof['normal_new_sources']) & set(static_profiles) if opening_proof else set()) | set(shift_proof['normal_static_sources'] if shift_proof else []) | (combat_hd_sources & set(static_profiles))),
              "runtime_observed_opening_art_prepared":1 if full_profile else 0,
              "runtime_observed_opening_sources_normal_verified":1 if opening_proof else 0,
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
              "counts": counts, "runtime_proof_sources": proof_sources, "verification_tool_sources": verifier_sources, "records": records}
    (output / "art-coverage.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(counts, ensure_ascii=False))


if __name__ == "__main__":
    main()
