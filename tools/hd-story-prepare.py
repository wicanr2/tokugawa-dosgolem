"""從固定 DAT 與已審查手繪圖建立劇情來源對照，不是正式素材包格式。"""
import argparse
import hashlib
import importlib.util
import json
import os
import sys
from pathlib import Path

from PIL import Image, __version__ as pillow_version


REVIEWED = ("11", "19", "28", "43", "6", "A-0", "A-1", "A-2", "A-3", "A-4", "A-5", "A-6", "A-7", "A-8",
            "B-1", "B-2", "B-3", "B-4", "C-1", "C-1'", "C-2", "C-4", "C-5", "C-6", "C-7", "C-8", "C-9", "C-11",
            "D-1", "D-2", "D-3", "D-4", "D-8", "E-1", "E-2", "E-3", "E-6", "F-1", "F-2", "F-5", "G-1", "G-2",
            "D-6", "E-5", "END-10", "END-8", "END-9", "F-3", "F-6", "F-7", "F-8", "G-4", "G-5", "G-7",
            "GUNGI", "H-1", "H-2", "H-3", "KOYO", "YOULOSE")
VARIANTS = {"D-2.OZM": "D-2-hd-b-v2.png", "E-1.OZM": "E-1-hd-b-v2.png", "GUNGI.OZM": "GUNGI-hd-b-v2.png"}
LABELS = {
    "A-6.OZM": [(249, 112, 305, 136)],
    "A-7.OZM": [(102, 54, 146, 82), (160, 35, 202, 61), (110, 112, 171, 140)],
    "B-1.OZM": [(100, 82, 149, 128)],
    "B-2.OZM": [(100, 82, 149, 128), (156, 37, 198, 61)],
    "C-5.OZM": [(249, 0, 308, 27), (252, 55, 318, 84)],
    "C-8.OZM": [(173, 12, 256, 37), (105, 99, 184, 127)],
    "E-2.OZM": [(226, 16, 310, 49)],
    "F-6.OZM": [(246, 109, 307, 139)],
    "H-1.OZM": [(246, 109, 307, 139)],
    "H-2.OZM": [(246, 109, 307, 139)],
}


def sha(data):
    return hashlib.sha256(data).hexdigest()


def write_checked(path, data):
    assert path.parent.stat().st_uid == 1000
    if path.exists():
        assert path.stat().st_uid == 1000
        assert path.read_bytes() == data, ("refusing changed original export", path)
    else:
        path.write_bytes(data)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--go-output", type=Path)
    args = parser.parse_args()
    assert Path("/.dockerenv").is_file() and os.getuid() == 1000 and pillow_version == "9.4.0"
    project = Path("/project")
    catalog = project/"workplace/hd-prototype/catalog"
    output = Path("/output")
    assert output.stat().st_uid == 1000
    root = output/"story-expanded"
    root.mkdir(exist_ok=True)
    sources = root/"sources"
    sources.mkdir(exist_ok=True)
    assert root.stat().st_uid == sources.stat().st_uid == 1000
    dat = Path("/original/DX5.DAT").read_bytes()
    assert sha(dat) == "de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37"
    decoder_path = project/"tools/hd-ozm-probe.py"
    spec = importlib.util.spec_from_file_location("ozm_decoder", decoder_path)
    decoder = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(decoder)
    inventory = json.loads((catalog/"decoded/inventory.json").read_text())
    assert inventory["tool_sha256"] == sha(decoder_path.read_bytes()) and inventory["dat_sha256"] == sha(dat)
    decoded = {r["name"]: r for r in inventory["records"]}
    capture_path = next((catalog/"hd-assets").glob("A-0*.json"))
    capture = json.loads(capture_path.read_text())
    palette = [tuple(c[k] for k in ("R", "G", "B")) for c in capture["palette_at_loader_return"]]
    captures = {json.loads(p.read_text())["event"]["name"]: p for p in (catalog/"hd-assets").glob("*.json")}
    direct = {stem+".OZM": VARIANTS.get(stem+".OZM", stem+"-hd-b-v1.png") for stem in REVIEWED}
    shared = {}
    for name in direct:
        shared.setdefault(decoded[name]["indices_sha256"], name)
    records = []
    for name, source in sorted(decoded.items()):
        owner = name if name in direct else shared.get(source.get("indices_sha256"))
        if owner is None:
            continue
        assert source["status"] == "decoded" and source["width"] in (320, 328) and source["height"] == 150
        payload = dat[source["dat_file_offset"]:source["dat_file_offset"]+source["payload_bytes"]]
        assert sha(payload) == source["payload_sha256"]
        cols, rows, planes, indices, _, _ = decoder.decode(payload)
        assert [sha(p) for p in planes] == source["plane_sha256"] and sha(indices) == source["indices_sha256"]
        assert indices == (catalog/"decoded"/(name+".idx")).read_bytes()
        assert source["indices_sha256"] == decoded[owner]["indices_sha256"]
        assert source["plane_sha256"] == decoded[owner]["plane_sha256"]
        original = Image.frombytes("RGB", (cols*8, rows), bytes(c for n in indices for c in palette[n]))
        assert original.tobytes() == Image.open(catalog/"previews"/(name+".png")).convert("RGB").tobytes()
        visible = original.crop((0, 0, 320, 150))
        path = sources/(name.removesuffix(".OZM")+"-original.png")
        import io
        buf = io.BytesIO()
        visible.save(buf, format="PNG")
        write_checked(path, buf.getvalue())
        source_name = "workplace/hd-prototype/catalog/story-expanded/sources/"+path.name
        source_sha = sha(path.read_bytes())
        level = "static decoder and shared original draw consumer verified; normal path not yet sampled"
        if name in captures:
            captured = json.loads(captures[name].read_text())
            assert captured["indices_sha256"] == source["indices_sha256"]
            source_name = str(captures[name].with_suffix(".png").relative_to(project))
            source_sha = captured["png_sha256"]
            assert sha((project/source_name).read_bytes()) == source_sha
            assert Image.open(project/source_name).convert("RGB").tobytes() == visible.tobytes()
            level = "original normal-path source captured; presentation proof recorded separately"
        asset = (project/"workplace/hd-prototype/A-0-hd-b-v1.png") if owner == "A-0.OZM" else catalog/"art"/direct[owner]
        with Image.open(asset) as hd:
            w, h = hd.size
            assert hd.convert("RGBA").getextrema()[3] == (255, 255)
            assert w >= 320 and h >= 150 and max(w, h) <= 8192 and abs(w*150-h*320)*100 <= h*320
        labels = LABELS.get(owner, [])
        assert all(0 <= x0 < x1 <= 320 and 0 <= y0 < y1 <= 150 for x0, y0, x1, y1 in labels)
        records.append({"resource": name, "source": source_name, "source_png_sha256": source_sha,
                        "columns": cols, "plane_sha256": source["plane_sha256"], "indices_sha256": source["indices_sha256"],
                        "asset": direct[owner], "asset_sha256": sha(asset.read_bytes()), "art_source_resource": owner,
                        "label_rects": labels, "visible_source_crop": [0, 0, 320, 150], "evidence_level": level})
    assert len(records) == 75 and len(direct) == 60
    lines = ['// Code generated by tools/hd-story-prepare.py; DO NOT EDIT.',
             '// 原版來源與共同繪圖端對照。正常路徑驗證範圍另見 CONTEXT。', 'package main', 'import "image"',
             'type hdPortraitProfile struct {', 'Source, SourceSHA, Asset string', 'Columns uint32', 'Planes [4]string', '}',
             'var hdPortraitProfiles = map[string]hdPortraitProfile{']
    for r in records:
        q = lambda s: json.dumps(s, ensure_ascii=False)
        lines.append(f'{q(r["resource"])}: {{Source: {q("/project/"+r["source"])}, SourceSHA: {q(r["source_png_sha256"])}, Asset: {q(r["asset"])}, Columns: {r["columns"]}, Planes: [4]string{{'+','.join(q(s) for s in r['plane_sha256'])+'}},')
    lines += ['}', 'var portraitLabelRects = map[string][]image.Rectangle{']
    for r in records:
        if r["label_rects"]:
            lines.append(json.dumps(r['resource'])+': {'+','.join('image.Rect('+','.join(map(str,v))+')' for v in r['label_rects'])+'},')
    lines += ['}', '']
    go_output = args.go_output or root/"hd-portraits.go"
    assert go_output.parent.stat().st_uid == 1000
    assert not go_output.exists() or go_output.stat().st_uid == 1000
    go_output.write_text('\n'.join(lines))
    result = {"classification": "reviewed story sources and compiled prototype profiles; not a production pack schema",
              "dat_sha256": sha(dat), "tool_sha256": sha(Path(__file__).read_bytes()), "decoder_sha256": sha(decoder_path.read_bytes()),
              "python_version": sys.version, "pillow_version": pillow_version, "source_count": len(records), "art_count": len(direct),
              "caller_evidence_sha256": sha((project/"workplace/hd-prototype/range-17fb0-180a0.json").read_bytes()),
              "copy_evidence_sha256": sha((project/"workplace/hd-prototype/range-1bf73-1c550.json").read_bytes()), "records": records}
    target = root/"profiles.json"
    assert not target.exists() or target.stat().st_uid == 1000
    target.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({"sources": len(records), "art": len(direct), "go_output": str(go_output)}))


if __name__ == "__main__":
    main()
