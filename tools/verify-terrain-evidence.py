"""獨立核對地面來源、兩次正常 GUI 與當時的建置輸入。"""
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000
    project = Path("/project")
    terrain = project/"workplace/hd-prototype/catalog/terrain"
    runtime = terrain/"runtime"
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
        for picture in proof["pictures"]:
            assert sha(folder/Path(picture["actual_png"]).relative_to("/output")) == picture["sha256"]
            assert picture["window_pixels_equal"]
        records.append({"mode": mode, "receipt_sha256": sha(receipt_path), "GUI_verification_sha256": sha(proof_path),
                        "binary_sha256": sha(folder/"tokugawa-desktop-prototype"),
                        "verifier_snapshot_sha256": sha(folder/"verify-desktop.py"),
                        "normal_floor_indices": indices, "HD_rectangles": len(check["active_rectangles"]),
                        "changed_pixels": check["changed_pixels"], "guest_unchanged": True})
    assert records[0]["binary_sha256"] == records[1]["binary_sha256"]
    result = {"classification": "approved floor-source slice and first-battle GUI verification; not whole-game HD",
              "whole_game_HD_complete": False, "source_count": 104, "approved_HD_floor_count": 8,
              "normal_HD_floor_cases": 6, "source_profiles_sha256": sha(profile_path),
              "baseline_sha256": sha(baseline_path), "compared_fields": fields, "records": records,
              "platform_test_log_sha256": sha(runtime/"startup-probe/platform-test.log"), "tool_sha256": sha(Path(__file__))}
    target = output/"verification-summary.json"
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps({"verified": True, "normal_HD_floor_cases": 6, "summary_sha256": sha(target)}))


if __name__ == "__main__":
    main()
