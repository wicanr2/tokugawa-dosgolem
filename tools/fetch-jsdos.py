"""在 Docker 內取得固定版本 jsdos，驗證後保留原始檔與解包副本。"""

import datetime
import hashlib
import json
import os
import pathlib
import platform
import shutil
import stat
import urllib.request
import zipfile


assert pathlib.Path("/.dockerenv").is_file(), "請在 Docker 容器內執行"
root = pathlib.Path("/project")
assert root.is_dir() and root.stat().st_uid == os.getuid() == 1000
original = root / "original"
original.mkdir(exist_ok=True)
assert original.is_dir() and original.stat().st_uid == os.getuid()
archive = original / "德川家康.jsdos.zip"
assert not archive.exists(), "不覆寫既有原始檔"
assert not (original / "game").exists(), "不覆寫既有解包目錄"

page_url = "https://dos.zczc.cz/games/%E5%BE%B7%E5%B7%9D%E5%AE%B6%E5%BA%B7/"
api_url = "https://api-workers.dos.lol/v1/games/%E5%BE%B7%E5%B7%9D%E5%AE%B6%E5%BA%B7"
with urllib.request.urlopen(api_url, timeout=30) as response:
    metadata = json.load(response)
expected = "e35bb236ae48f095a6dbda3236a59e5a5b903c82fe9d5564233c15c1793b425a"
assert metadata["identifier"] == "德川家康"
assert metadata["binary_jsdos"] == expected + ".zip", "網站版本已變，請先核對來源"
url = "https://binary.dos.lol/" + metadata["binary_jsdos"]
partial = archive.with_name(archive.name + ".part")
expected_size = 214665581
with urllib.request.urlopen(urllib.request.Request(url, method="HEAD"), timeout=30) as response:
    assert int(response.headers["Content-Length"]) == expected_size
assert shutil.disk_usage(original).free > expected_size + 2 * 1024**3
size = partial.stat().st_size if partial.exists() else 0
assert 0 <= size < expected_size
digest = hashlib.sha256()
if partial.exists():
    assert partial.is_file() and partial.stat().st_uid == os.getuid()
    with partial.open("rb") as prefix:
        while chunk := prefix.read(1024 * 1024):
            digest.update(chunk)
request = urllib.request.Request(url, headers={"Range": f"bytes={size}-"} if size else {})
with urllib.request.urlopen(request, timeout=30) as response:
    if size:
        assert response.status == 206
        assert response.headers["Content-Range"] == f"bytes {size}-{expected_size - 1}/{expected_size}"
    headers = dict(response.headers)
    final_url = response.url
    with partial.open("ab" if size else "xb") as output:
        while chunk := response.read(1024 * 1024):
            size += len(chunk)
            assert size <= expected_size
            digest.update(chunk)
            output.write(chunk)
assert size == expected_size
actual = digest.hexdigest()
assert actual == expected, (actual, expected)

with zipfile.ZipFile(partial) as bundle:
    entries = bundle.infolist()
    assert len(entries) <= 10000
    assert sum(e.file_size for e in entries) <= 2 * 1024**3
    names = set()
    for entry in entries:
        path = pathlib.PurePosixPath(entry.filename)
        assert not path.is_absolute() and ".." not in path.parts
        assert "\\" not in entry.filename
        assert not stat.S_ISLNK(entry.external_attr >> 16)
        assert entry.filename.casefold() not in names, entry.filename
        names.add(entry.filename.casefold())
    assert bundle.testzip() is None
    inventory = []
    game = original / "game"
    game.mkdir()
    for entry in entries:
        target = game / entry.filename
        if entry.is_dir():
            target.mkdir(parents=True, exist_ok=True)
            continue
        target.parent.mkdir(parents=True, exist_ok=True)
        member_digest = hashlib.sha256()
        with bundle.open(entry) as source, target.open("xb") as output:
            while chunk := source.read(1024 * 1024):
                member_digest.update(chunk)
                output.write(chunk)
        target.chmod(0o444)
        sha256 = member_digest.hexdigest()
        inventory.append({"path": entry.filename, "bytes": entry.file_size, "sha256": sha256})
        print(entry.filename, entry.file_size, sha256)
        if entry.filename.lower().endswith((".conf", ".bat", ".json")) and entry.file_size < 30000:
            print(target.read_bytes().decode("utf-8", errors="replace"))

partial.rename(archive)
archive.chmod(0o444)
receipt = {
    "retrieved_at_utc": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "page_url": page_url,
    "api_url": api_url,
    "api_response": metadata,
    "frontend_script_url": "https://dos.zczc.cz/assets/index-Bsp10LlZ.js",
    "download_url": url,
    "final_url": final_url,
    "response_headers": headers,
    "archive_path": "original/" + archive.name,
    "archive_bytes": size,
    "expected_sha256": expected,
    "archive_sha256": actual,
    "zip_crc_check": "passed",
    "tool": {
        "python": platform.python_version(),
        "image": "python@sha256:5f55cdf0c5d9dc1a415637a5ccc4a9e18663ad203673173b8cda8f8dcacef689",
        "script_sha256": hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),
    },
    "members": inventory,
}
(original / "source.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
print("Archive:", size, "bytes;", len(entries), "entries; SHA256:", actual)
print("Owner:", archive.stat().st_uid, archive.stat().st_gid)
