"""輸出資源入口的原始 IDA 證據，不改名或寫入語意。"""

import hashlib
import json
import pathlib

import ida_auto
import ida_bytes
import ida_funcs
import ida_kernwin
import ida_loader
import ida_nalt
import idautils
import idc


ida_auto.auto_wait()
source = pathlib.Path(ida_nalt.get_input_file_path())
data = source.read_bytes()
digest = hashlib.sha256(data).hexdigest()
assert digest == "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a"
anchors = []
functions = set()
for needle in (b"DX5.DAT", b"Ken Lin", b"KAO.GRP"):
    start = 0
    while True:
        offset = data.find(needle, start)
        if offset < 0:
            break
        start = offset + len(needle)
        ea = ida_loader.get_fileregion_ea(offset)
        assert ida_bytes.get_bytes(ea, len(needle)) == needle
        refs = []
        for x in idautils.XrefsTo(ea):
            func = ida_funcs.get_func(x.frm)
            refs.append({"ida_linear_ea": hex(x.frm), "type": x.type,
                         "function_start": hex(func.start_ea) if func else None,
                         "listing": idc.generate_disasm_line(x.frm, 0)})
            if func:
                functions.add(func.start_ea)
        anchors.append({"bytes_hex": needle.hex(), "text": needle.decode(),
                        "file_offset": hex(offset), "ida_linear_ea": hex(ea), "direct_xrefs": refs})

# 只匯出錨點直接使用者，不遞迴掃描整個 EXE。
rows = []
limits = []
for start in sorted(functions):
    func = ida_funcs.get_func(start)
    assert func and func.end_ea - start <= 0x4000
    limits.append({"original_function": ida_funcs.get_func_name(start),
                   "start": hex(start), "end": hex(func.end_ea)})
    for ea in idautils.FuncItems(start):
        size = ida_bytes.get_item_size(ea)
        rows.append({"ida_linear_ea": hex(ea), "file_offset": hex(ida_loader.get_fileregion_offset(ea)),
                     "bytes": (ida_bytes.get_bytes(ea, min(size, 32)) or b"").hex(),
                     "original_ida_name": idc.get_name(ea), "original_function": ida_funcs.get_func_name(start),
                     "listing": idc.generate_disasm_line(ea, 0),
                     "incoming_xrefs": [{"ida_linear_ea": hex(x.frm), "type": x.type} for x in idautils.XrefsTo(ea)]})
assert len(rows) < 10000
result = {"input_file": source.name, "input_sha256": digest, "tool": "IDA Pro",
          "version": ida_kernwin.get_kernel_version(),
          "address_space": "IDA LE loader linear EA; file offsets recorded separately",
          "semantic_level": "raw tool evidence; names are navigation only", "custom_renames": [],
          "probe_sha256": hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),
          "anchors": anchors, "function_ranges": limits, "rows": rows}
pathlib.Path("/output/asset-xrefs.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
idc.qexit(0)
