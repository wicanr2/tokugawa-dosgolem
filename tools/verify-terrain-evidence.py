"""獨立核對地面來源、兩次正常 GUI 與當時的建置輸入。"""
import argparse
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify_original_cursor(project, terrain):
    """從原始索引獨立合成前景及游標，解釋離線 64 格與桌面 65 格的差異。"""
    from PIL import Image
    trace = terrain/'objects/trace'
    evidence = json.loads((trace/'verification.json').read_text())
    receipt = json.loads((trace/'startup-probe/latest.json').read_text())
    assert sha(trace/'startup-probe/latest.json') == evidence['receipt_sha256']
    profiles = json.loads((terrain/'objects/source-profiles.json').read_text())
    idx_path = terrain.parent/'decoded/MAP_CHIP.OZM.idx'
    idx = idx_path.read_bytes()
    assert sha(idx_path) == profiles['source_indices_sha256']
    palette_path = terrain/'trace/native-floor-palette-verification.json'
    assert sha(palette_path) == profiles['palette_verification_sha256']
    palette = json.loads(palette_path.read_text())['palette']
    native_path = project/receipt['frame']['path']
    assert sha(native_path) == receipt['frame']['png_sha256']
    native = Image.open(native_path).convert('RGB')
    cursor = receipt['mouse_cursor']
    last = receipt['mouse_input_events'][-1]
    assert cursor['Defined'] and cursor['DisplayCount'] >= 0
    cx, cy = last['x']-cursor['HotX'], last['y']-cursor['HotY']
    before_matches = after_matches = foreground_pixels = equal_colour = cursor_differences = 0
    for floor in evidence['current_floors']:
        fx, fy = floor['index']%13*48, floor['index']//13*32
        base = bytearray(idx[(fy+y)*640+fx+x] for y in range(32) for x in range(48))
        combined, mask = base[:], bytearray(48*32)
        x0,y0,x1,y1 = floor['rect']
        for sprite in evidence['sprites']:
            if sprite['order'] <= floor['order']:
                continue
            sx0,sy0,sx1,sy1 = sprite['rect']
            ax = sprite['index']%13*48
            ay = 256+sprite['index']//13*48 if sprite['group']=='objects' else 352
            for y in range(max(y0,sy0), min(y1,sy1)):
                for x in range(max(x0,sx0), min(x1,sx1)):
                    value = idx[(ay+y-sy0)*640+ax+x-sx0]
                    if value:
                        n = (y-y0)*48+x-x0
                        combined[n], mask[n] = value, 1
        rgb_before = bytes(c for value in combined for c in palette[value])
        actual = native.crop(floor['rect']).tobytes()
        before_matches += rgb_before == actual
        foreground_pixels += sum(mask)
        equal_colour += sum(opaque and palette[combined[n]]==palette[base[n]] for n,opaque in enumerate(mask))
        for y in range(max(y0,cy), min(y1,cy+16)):
            for x in range(max(x0,cx), min(x1,cx+16)):
                bit, row = 0x8000>>(x-cx), y-cy
                n = (y-y0)*48+x-x0
                if not cursor['Mask'][row]&bit:
                    combined[n] = 0
                if cursor['Mask'][row+16]&bit:
                    combined[n] ^= 15
        rgb_after = bytes(c for value in combined for c in palette[value])
        assert rgb_after == actual, ('original cursor composition mismatch',floor)
        after_matches += 1
        cursor_differences += sum(rgb_before[n:n+3] != rgb_after[n:n+3] for n in range(0,len(rgb_after),3))
    assert (before_matches,after_matches,foreground_pixels)==(64,65,7067)
    return {'verified': True, 'native_frame_sha256': sha(native_path), 'original_trace_sha256': sha(trace/'verification.json'),
            'matched_before_cursor': before_matches, 'matched_with_original_cursor': after_matches,
            'original_foreground_pixels': foreground_pixels, 'same_colour_foreground_pixels': equal_colour,
            'cursor_pixels_changed': cursor_differences, 'cursor_origin': [cx,cy],
            'method': 'raw atlas indices, original draw order, index-zero alpha, original 16x16 AND/XOR cursor'}


def main():
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000
    parser = argparse.ArgumentParser()
    parser.add_argument('--foreground', action='store_true', help='核對保留原始透明前景的地面切片')
    args = parser.parse_args()
    project = Path("/project")
    terrain = project/"workplace/hd-prototype/catalog/terrain"
    runtime = terrain/("objects" if args.foreground else "runtime")
    output = Path("/output")
    assert output.stat().st_uid == 1000
    profile_path = terrain/"prepared/profiles.json"
    profiles = json.loads(profile_path.read_text())
    assert profiles["source_count"] == len(profiles["records"]) == 104
    assert profiles["reviewed_HD_count"] == 8
    assert profiles["tool_sha256"] == sha(project/"tools/hd-terrain-prepare.py")
    for source in profiles["records"]:
        assert sha(terrain/source["source"]) == source["source_png_sha256"]
        if source["asset"]:
            assert sha(project/"assets/hd-b/terrain"/source["asset"]) == source["asset_sha256"]
    baseline_path = project/"workplace/desktop-prototype/headless/startup-probe/latest.json"
    baseline = json.loads(baseline_path.read_text())
    fields = ["successful_steps", "instruction_budget", "stop", "stop_instruction_linear_eip", "trace_tail",
              "interrupt_counts", "dpmi_calls", "dpmi_unimplemented", "frame", "cd_pcm", "mounted_disc",
              "mouse_input_events", "input_boundary_frames", "real_mode_history", "real_mode_last"]
    records = []
    for folder_name, mode in (("desktop", "terrain"), ("desktop-missing", "terrain-missing")):
        folder = runtime/folder_name
        receipt_path, proof_path = folder/("desktop-"+mode+".json"), folder/("verification-"+mode+".json")
        receipt, proof = json.loads(receipt_path.read_text()), json.loads(proof_path.read_text())
        assert proof["verified"] and proof["receipt_sha256"] == sha(receipt_path)
        assert all(receipt[k] == baseline[k] for k in fields), [k for k in fields if receipt[k] != baseline[k]]
        assert receipt["successful_steps"] == 750_000_000 and receipt["fd2_startup_calls"] == 0
        assert len(receipt["mouse_input_events"]) == 390
        assert sha(folder/"startup-memory.bin") == "4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9"
        for name, digest in receipt["integration_input_sha256"].items():
            assert sha(project/name) == digest, ("stale runtime input", name)
        toggles = [e for e in receipt["desktop"]["events"] if e["kind"] == "toggle"]
        assert len(toggles) == 3 and all(e["guest_unchanged"] and e["before"] == e["after"] for e in toggles)
        state = receipt["desktop"]["hd_observation"]["terrain"]
        assert state["pending_return"] is None and state["completed_returns"] == 134 and len(state["draws"]) == 65
        if args.foreground:
            assert state['completed_sprite_returns'] == 11 and len(state['sprites']) == 11 and state['draw_sequence'] == 145
            assert sorted({(d['group'], d['index']) for d in state['sprites']}) == [('objects', 0), ('units', 1), ('units', 3)]
            sprite_profiles_path = terrain/'objects/source-profiles.json'
            sprite_profiles = json.loads(sprite_profiles_path.read_text())
            assert sprite_profiles['tool_sha256'] == sha(project/'tools/hd-terrain-object-prepare.py')
            assert len(sprite_profiles['records']) == len(receipt['desktop']['assets']['terrain_sprite_sources']) == 39
            for source in sprite_profiles['records']:
                assert sha(terrain/'objects'/source['source']) == source['source_png_sha256']
                key = f'{source["group"]}-{source["index"]:03d}'
                info = receipt['desktop']['assets']['terrain_sprite_sources'][key]
                assert info['source_sha256'] == source['source_png_sha256'] and info['validation'] == 'ready_native_mask'
        loaded = receipt["desktop"]["assets"]["terrain"]
        assert len(loaded) == 104
        for source in profiles["records"]:
            info = loaded[str(source["index"])]
            assert info["source_sha256"] == source["source_png_sha256"]
            if mode == "terrain" and source["asset"]:
                assert info["hd_sha256"] == source["asset_sha256"] and info["validation"] == "ready"
            else:
                assert info["validation"] == "fallback_missing_asset"
        assert len(proof["terrain_checks"]) == 1
        check = proof["terrain_checks"][0]
        indices = sorted({d["index"] for d in check["active_rectangles"]})
        assert indices == ([0, 2, 6, 13, 15, 22] if mode == "terrain" else [])
        assert check["outside_terrain_unchanged"] and check["guest_unchanged"]
        if args.foreground:
            assert check['original_foreground_unchanged']
            assert len(check['active_rectangles']) == (65 if mode == 'terrain' else 0)
            assert check['foreground_native_pixels'] == (7067 if mode == 'terrain' else 0)
            assert check['foreground_output_pixels'] == check['foreground_native_pixels']*4
        for picture in proof["pictures"]:
            assert sha(folder/Path(picture["actual_png"]).relative_to("/output")) == picture["sha256"]
            assert picture["window_pixels_equal"]
        records.append({"mode": mode, "receipt_sha256": sha(receipt_path), "GUI_verification_sha256": sha(proof_path),
                        "binary_sha256": sha(folder/"tokugawa-desktop-prototype"),
                        "verifier_snapshot_sha256": sha(folder/"verify-desktop.py"),
                        "normal_floor_indices": indices, "HD_rectangles": len(check["active_rectangles"]),
                        "changed_pixels": check["changed_pixels"], "guest_unchanged": True})
        if args.foreground:
            records[-1]['foreground_native_pixels'] = check['foreground_native_pixels']
            records[-1]['original_foreground_unchanged'] = True
    assert records[0]["binary_sha256"] == records[1]["binary_sha256"]
    result = {"classification": "approved floor-source slice and first-battle GUI verification; not whole-game HD",
              "whole_game_HD_complete": False, "source_count": 104, "approved_HD_floor_count": 8,
              "normal_HD_floor_cases": 6, "source_profiles_sha256": sha(profile_path),
              "baseline_sha256": sha(baseline_path), "compared_fields": fields, "records": records,
              "platform_test_log_sha256": sha(runtime/"startup-probe/platform-test.log"), "tool_sha256": sha(Path(__file__))}
    if args.foreground:
        result['classification'] = 'normal first-battle HD floors with source-index-zero foreground preservation; not HD sprites or whole-game HD'
        result['source_sprite_count'] = 39
        result['source_sprite_profiles_sha256'] = sha(sprite_profiles_path)
        result['original_draw_trace_verification_sha256'] = sha(terrain/'objects/trace/verification.json')
        result['original_cursor_verification'] = verify_original_cursor(project, terrain)
    target = output/"verification-summary.json"
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps({"verified": True, "normal_HD_floor_cases": 6, "summary_sha256": sha(target)}))


if __name__ == "__main__":
    main()
