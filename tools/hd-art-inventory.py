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
            "rights": "original-derived; local only; excluded from Git and distribution"}


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
    direct = {}
    for record in decoded["records"]:
        stem = record["name"].removesuffix(".OZM")
        path = ROOT / "art" / f"{stem}-hd-b-v1.png"
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
    for directory in ("desktop", "kao-desktop"):
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
        if name == "VISUAL.OZM":
            path = PROJECT / "workplace/hd-prototype/scroll/scroll-hd-b-v1.png"
            record.update(art_status="partial", art=art_info(path, path.with_suffix(".prompt.txt")),
                          logical_region=[0, 250, 640, 150], runtime_status="scroll region verified separately; upper frame pending")
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
              "native_KAO_cases_verified": len(verified_kao), "unsupported_OZM_decodes": 5}
    result = {"classification": "local research coverage ledger; not a production asset-pack schema",
              "whole_game_HD_complete": False,
              "count_limit": "resource counts do not count atlas tiles, animation frames or UI pieces; no completion percentage",
              "dat_sha256": decoded["dat_sha256"], "tool_sha256": sha(Path(__file__)), "pillow_version": pillow_version,
              "counts": counts, "runtime_proof_sources": proof_sources, "records": records}
    (output / "art-coverage.json").write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(counts, ensure_ascii=False))


if __name__ == "__main__":
    main()
