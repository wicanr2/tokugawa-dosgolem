"""匯出 LE 啟動區的原始 IDA 定位，不改名、不覆蓋語意。"""

import hashlib
import json
import pathlib

import ida_auto
import ida_bytes
import ida_funcs
import ida_ida
import ida_kernwin
import ida_loader
import ida_nalt
import ida_segment
import idautils
import idc


ida_auto.auto_wait()
path = pathlib.Path(ida_nalt.get_input_file_path())
digest = hashlib.sha256(path.read_bytes()).hexdigest()
assert digest in {
    "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a",
    "1ea6dc335b42b39542ba8dec35c8ffd103e4b495d30700c4100e60dbcbecaa60",
}
entry = ida_ida.inf_get_start_ea()
segments = []
for ea in idautils.Segments():
    seg = ida_segment.getseg(ea)
    segments.append({"name": ida_segment.get_segm_name(seg), "start": hex(seg.start_ea),
                     "end": hex(seg.end_ea), "bitness": seg.bitness})

rows = []
range_start, range_end = entry, entry + 0x1000
output_name = "intake.json"
if len(idc.ARGV) > 1:
    range_start, range_end = (int(part, 0) for part in idc.ARGV[1].split(":"))
    assert 0x10000 <= range_start < range_end <= 0x5C5F0
    assert range_end - range_start <= 0x4000
    output_name = f"range-{range_start:x}-{range_end:x}.json"
for ea in idautils.Heads(range_start, range_end):
    size = ida_bytes.get_item_size(ea)
    func = ida_funcs.get_func(ea)
    file_offset = ida_loader.get_fileregion_offset(ea)
    rows.append({
        "ida_linear_ea": hex(ea), "file_offset": hex(file_offset),
        "bytes": (ida_bytes.get_bytes(ea, min(size, 32)) or b"").hex(),
        "original_ida_name": idc.get_name(ea),
        "original_ida_function": hex(func.start_ea) if func else None,
        "listing": idc.generate_disasm_line(ea, 0),
        "incoming_xrefs": [{"ida_linear_ea": hex(x.frm), "type": x.type}
                           for x in idautils.XrefsTo(ea)],
    })

receipt = {
    "input_file": path.name, "input_sha256": digest,
    "tool": "IDA Pro", "version": ida_kernwin.get_kernel_version(),
    "address_space": "IDA LE loader linear EA; file offsets recorded separately",
    "entry_ida_linear_ea": hex(entry), "function_count": len(list(idautils.Functions())),
    "segments": segments, "rows": rows,
    "requested_ida_linear_range": [hex(range_start), hex(range_end)],
    "custom_renames": [], "semantic_level": "raw tool evidence; names are navigation only",
}
pathlib.Path("/output", output_name).write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
idc.qexit(0)
