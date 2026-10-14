"""核對共用視窗的五次真實 GUI 收據、建置來源與原版狀態。"""
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
    root = project / "workplace/hd-prototype/catalog/window"
    output = Path("/output")
    assert output.stat().st_uid == 1000
    baseline_path = project / "workplace/hd-prototype/asset-capture/startup-probe/latest.json"
    baseline = json.loads(baseline_path.read_text())
    fields = ["successful_steps", "instruction_budget", "stop", "stop_instruction_linear_eip", "trace_tail",
              "interrupt_counts", "dpmi_calls", "dpmi_unimplemented", "frame", "cd_pcm", "mounted_disc",
              "mouse_input_events", "input_boundary_frames", "real_mode_history", "real_mode_last"]
    records = []
    for mode in ("window-only", "window", "window-missing", "window-modal", "toolbar-window-only"):
        toolbar = mode.startswith("toolbar")
        folder = root / ("toolbar-window-only" if toolbar else "desktop-"+mode)
        receipt_path = folder / ("windowbar-window-only.json" if toolbar else "desktop-"+mode+".json")
        proof_path = folder / ("verification-windowbar-window-only.json" if toolbar else "verification-"+mode+".json")
        proof = json.loads(proof_path.read_text())
        receipt = json.loads(receipt_path.read_text())
        assert proof["verified"] and proof["receipt_sha256"] == sha(receipt_path), mode
        assert receipt["successful_steps"] == 150_000_000 and receipt["stop"] == "instruction_budget"
        assert all(receipt[k] == baseline[k] for k in fields), (mode, [k for k in fields if receipt[k] != baseline[k]])
        assert sha(folder/"startup-memory.bin") == "0d9123e52923ba852b125ea04918e1007138a33e0a8e4d856b078865cb33562f"
        assert receipt["fd2_startup_calls"] == 0
        for name, digest in receipt["integration_input_sha256"].items():
            assert sha(project/name) == digest, (mode, "stale build source", name)
        desktop = receipt["desktop"]
        toggles = [e for e in desktop["events"] if e["kind"] == "toggle"]
        assert toggles and all(e["guest_unchanged"] and e["before"] == e["after"] for e in toggles)
        observation = desktop["hd_observation"]["window"]
        assert all(observation[k] is None for k in ("pending_draw", "pending_save", "pending_restore"))
        kinds = [e["kind"] for e in observation["events"]]
        assert kinds.count("window_save_return") > 0 and kinds.count("window_restore_return") > 0
        assets = desktop["assets"]["window"]
        assert assets["source_sha256"] == "8a4c3c581d76b131644f69733b61783fafde14541263cfde1d3dfe594d66978b"
        assert assets["validation"] == ("fallback_missing_asset" if mode == "window-missing" else "ready")
        if mode != "window-missing":
            assert assets["hd_sha256"] == sha(project/"assets/hd-b/window/window-skin-hd-b-v1.png")
        script = folder/("verify-windowbar.py" if toolbar else "verify-desktop.py")
        for picture in proof["pictures"]:
            path = folder/Path(picture.get("path", picture.get("actual_png"))).relative_to("/output")
            assert sha(path) == picture["sha256"], (mode, path)
            if not toolbar:
                assert picture["window_pixels_equal"]
        record = {"mode": mode, "receipt_sha256": sha(receipt_path), "verification_sha256": sha(proof_path),
                  "binary_sha256": sha(folder/"tokugawa-desktop-prototype"), "verifier_snapshot_sha256": sha(script),
                  "ram_sha256": sha(folder/"startup-memory.bin"), "guest_unchanged": True,
                  "window_save_returns": kinds.count("window_save_return"),
                  "window_restore_returns": kinds.count("window_restore_return"), "checks": proof.get("window_checks", [])}
        records.append(record)
    assert len({r["binary_sha256"] for r in records}) == 1
    assert len(records[3]["checks"]) == 3
    result = {"classification": "WINDOW.OZM common UI slice CONFORMED; Linux container only",
              "whole_game_HD_complete": False, "baseline_sha256": sha(baseline_path), "compared_fields": fields,
              "source_extraction_sha256": sha(root/"window-extraction.json"),
              "hd_asset_sha256": sha(project/"assets/hd-b/window/window-skin-hd-b-v1.png"),
              "platform_tests_sha256": sha(root/"startup-probe/platform-test.log"),
              "tool_sha256": sha(Path(__file__)), "records": records}
    (output/"verification-summary.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps({"verified": True, "modes": len(records), "summary_sha256": sha(output/"verification-summary.json")}, ensure_ascii=False))


if __name__ == "__main__":
    main()
