"""在 Docker 內離線核對下載與解包副本，輸出驗證收據。"""

import datetime
import hashlib
import json
import os
import pathlib
import platform
import stat
import struct
import zipfile


assert pathlib.Path("/.dockerenv").is_file(), "請在 Docker 容器內執行"
root = pathlib.Path("/project")
output = pathlib.Path("/output")
assert output.is_dir() and output.stat().st_uid == os.getuid()
source = json.loads((root / "original/source.json").read_text())


def sha256(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


archive = root / source["archive_path"]
assert archive.stat().st_size == source["archive_bytes"]
assert sha256(archive) == source["expected_sha256"] == source["archive_sha256"]
with zipfile.ZipFile(archive) as bundle:
    assert bundle.testzip() is None
    entries = len(bundle.infolist())
    assert {e.filename for e in bundle.infolist() if not e.is_dir()} == {
        m["path"] for m in source["members"]
    }

for member in source["members"]:
    path = root / "original/game" / member["path"]
    assert path.stat().st_size == member["bytes"], member["path"]
    assert sha256(path) == member["sha256"], member["path"]
    assert path.stat().st_uid == os.getuid(), member["path"]
    assert path.stat().st_mode & 0o222 == 0, member["path"]

assert archive.stat().st_uid == os.getuid() and archive.stat().st_mode & 0o222 == 0
bad_paths = []
for path in root.rglob("*"):
    entry = path.lstat()
    if entry.st_uid == 0 or (stat.S_ISDIR(entry.st_mode) and path.name.endswith(".md")):
        bad_paths.append(str(path.relative_to(root)))
assert not bad_paths, bad_paths

exe = (root / "original/game/DX5.EXE").read_bytes()
header_offset = struct.unpack_from("<I", exe, 0x3C)[0]
assert exe[:2] == b"MZ" and exe[header_offset:header_offset + 2] == b"LE"
receipt = {
    "verified_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "result": "passed",
    "archive_sha256": source["archive_sha256"],
    "archive_bytes": archive.stat().st_size,
    "zip_crc_check": "passed",
    "zip_entries": entries,
    "verified_files": len(source["members"]),
    "unpacked_bytes": sum(m["bytes"] for m in source["members"]),
    "owner_uid_gid": [os.getuid(), os.getgid()],
    "root_owned_or_md_directories": bad_paths,
    "le_header_file_offset": hex(header_offset),
    "python": platform.python_version(),
    "verification_script_sha256": sha256(pathlib.Path(__file__)),
    "download_script_sha256": sha256(root / "tools/fetch-jsdos.py"),
    "leprobe_receipts": {
        name: sha256(root / "workplace/leprobe" / name)
        for name in ("stdout.log", "stderr.log", "toolchain.txt")
    },
}
dosgolem = pathlib.Path("/dosgolem")
if dosgolem.is_dir():
    receipt["dosgolem_source_sha256"] = {
        name: sha256(dosgolem / name)
        for name in ("go.mod", "cmd/leprobe/main.go", "cmd/probe/main.go")
    }
(output / "intake-verification.json").write_text(
    json.dumps(receipt, ensure_ascii=False, indent=2) + "\n"
)
print(json.dumps(receipt, ensure_ascii=False, indent=2))
