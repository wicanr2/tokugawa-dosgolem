"""核對批次劇情來源及兩次 750M 正常玩家路徑收據。"""
import hashlib
import json
import os
from pathlib import Path


def sha(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024*1024), b""):
            h.update(block)
    return h.hexdigest()


def main():
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000
    project = Path("/project")
    root = project/"workplace/hd-prototype/catalog/story-expanded"
    output = Path("/output")
    assert output.stat().st_uid == 1000
    profiles_path = root/"profiles.json"
    profiles = json.loads(profiles_path.read_text())
    assert profiles["source_count"] == len(profiles["records"]) == 75 and profiles["art_count"] == 60
    assert profiles["tool_sha256"] == sha(project/"tools/hd-story-prepare.py")
    assert profiles["decoder_sha256"] == sha(project/"tools/hd-ozm-probe.py")
    baseline_path = project/"workplace/desktop-prototype/headless/startup-probe/latest.json"
    baseline = json.loads(baseline_path.read_text())
    fields = ["successful_steps", "instruction_budget", "stop", "stop_instruction_linear_eip", "trace_tail",
              "interrupt_counts", "dpmi_calls", "dpmi_unimplemented", "frame", "cd_pcm", "mounted_disc",
              "mouse_input_events", "input_boundary_frames", "real_mode_history", "real_mode_last"]
    records = []
    for folder_name, mode in (("desktop", "portraits"), ("desktop-missing", "portraits-missing")):
        folder = root/folder_name
        receipt_path = folder/("desktop-"+mode+".json")
        proof_path = folder/("verification-"+mode+".json")
        receipt = json.loads(receipt_path.read_text())
        proof = json.loads(proof_path.read_text())
        assert proof["verified"] and proof["receipt_sha256"] == sha(receipt_path)
        assert receipt["successful_steps"] == 750_000_000 and receipt["stop"] == "instruction_budget"
        assert all(receipt[k] == baseline[k] for k in fields), [k for k in fields if receipt[k] != baseline[k]]
        ram_sha = sha(folder/"startup-memory.bin")
        assert ram_sha == "4e873135fd0c22e2e8bb8330438fcb6320ecb28fc13a8f9551c984c7724ed5c9"
        assert receipt["fd2_startup_calls"] == 0 and len(receipt["mouse_input_events"]) == 390
        for name, digest in receipt["integration_input_sha256"].items():
            assert sha(project/name) == digest, ("stale integration source", name)
        desktop = receipt["desktop"]
        toggles = [e for e in desktop["events"] if e["kind"] == "toggle"]
        assert len(toggles) == 21 and all(e["guest_unchanged"] and e["before"] == e["after"] for e in toggles)
        loaded = desktop["assets"]["story_portraits"]
        assert len(loaded) == 74
        for source in profiles["records"]:
            assert sha(project/source["source"]) == source["source_png_sha256"]
            asset_path = project/"assets/hd-b/art"/source["asset"]
            if source["resource"] == "A-0.OZM":
                asset_path = project/"assets/hd-b/A-0-hd-b-v1.png"
                info = desktop["assets"]
            else:
                info = loaded[source["resource"]]
            assert sha(asset_path) == source["asset_sha256"]
            assert info["source_sha256"] == source["source_png_sha256"]
            if mode == "portraits" or source["resource"] == "A-0.OZM":
                assert info["hd_sha256"] == source["asset_sha256"] and not info.get("hd_error")
            else:
                assert "hd_sha256" not in info
        assert len(proof["portrait_checks"]) == 9 and len(proof["kao_checks"]) == 4
        for picture in proof["pictures"]:
            path = folder/Path(picture["actual_png"]).relative_to("/output")
            assert sha(path) == picture["sha256"] and picture["window_pixels_equal"]
        records.append({"mode": mode, "receipt_sha256": sha(receipt_path), "verification_sha256": sha(proof_path),
                        "binary_sha256": sha(folder/"tokugawa-desktop-prototype"), "ram_sha256": ram_sha,
                        "verifier_snapshot_sha256": sha(folder/"verify-desktop.py"), "guest_unchanged": True,
                        "normal_story_cases": len(proof["portrait_checks"]), "normal_kao_cases": len(proof["kao_checks"]),
                        "toggles": len(toggles)})
    assert records[0]["binary_sha256"] == records[1]["binary_sha256"]
    result = {"classification": "75 reviewed sources and shared draw consumer verified; normal player path sampled for nine story images",
              "whole_game_HD_complete": False, "source_profiles_sha256": sha(profiles_path),
              "baseline_sha256": sha(baseline_path), "compared_fields": fields, "records": records,
              "source_count": 75, "normal_story_cases_verified": 9, "not_sampled_on_normal_path": 66,
              "platform_test_log_sha256": sha(root/"startup-probe/platform-test.log"), "tool_sha256": sha(Path(__file__))}
    (output/"verification-summary.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps({"verified": True, "sources": 75, "normal_story_cases": 9,
                      "summary_sha256": sha(output/"verification-summary.json")}))


if __name__ == "__main__":
    main()
