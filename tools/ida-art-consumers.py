"""匯出繪圖資源讀取端的有界 caller 視窗，不改名、不附加推測語意。"""
import hashlib
import json
from pathlib import Path

import ida_auto
import ida_bytes
import ida_funcs
import ida_kernwin
import ida_loader
import ida_nalt
import idautils
import idc


ida_auto.auto_wait()
source = Path(ida_nalt.get_input_file_path())
digest = hashlib.sha256(source.read_bytes()).hexdigest()
assert digest == "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a"
rows = {}
refs = {}
targets = tuple(int(value, 0) for value in idc.ARGV[1].split(",")) if len(idc.ARGV) > 1 else (0x149E0, 0x34108, 0x35438, 0x35644)
assert 1 <= len(targets) <= 16 and all(0x10000 <= target < 0x5C5F0 for target in targets)
for target in targets:
    direct = list(idautils.XrefsTo(target))
    assert len(direct) <= 128
    refs[hex(target)] = [{"ida_linear_ea": hex(x.frm), "type": x.type} for x in direct]
    for ref in direct:
        func = ida_funcs.get_func(ref.frm)
        if not func:
            continue
        start = ref.frm
        for _ in range(18):
            previous = idc.prev_head(start, func.start_ea)
            if previous == idc.BADADDR:
                break
            start = previous
        for ea in idautils.Heads(start, min(ref.frm + 12, func.end_ea)):
            rows[ea] = {"ida_linear_ea": hex(ea), "file_offset": hex(ida_loader.get_fileregion_offset(ea)),
                        "original_ida_name": idc.get_name(ea), "original_ida_function": hex(func.start_ea),
                        "bytes": (ida_bytes.get_bytes(ea, min(ida_bytes.get_item_size(ea), 32)) or b"").hex(),
                        "listing": idc.generate_disasm_line(ea, 0)}
result = {"input_file": source.name, "input_sha256": digest, "tool": "IDA Pro", "version": ida_kernwin.get_kernel_version(),
          "address_space": "IDA LE loader linear EA; file offsets recorded separately", "function_count": len(list(idautils.Functions())),
          "custom_renames": [], "semantic_level": "raw tool evidence; names are navigation only", "direct_xrefs": refs,
          "rows": [rows[ea] for ea in sorted(rows)]}
suffix = "-" + "-".join(f"{target:x}" for target in targets) if len(idc.ARGV) > 1 else ""
Path("/output/art-consumers" + suffix + ".json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
idc.qexit(0)
