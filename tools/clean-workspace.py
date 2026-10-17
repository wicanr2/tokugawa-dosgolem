"""Docker 內列出或清除可重建快取與已封存的封包副本，保留研究證據。"""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess

assert Path('/.dockerenv').exists() and os.getuid() == 1000
parser = argparse.ArgumentParser()
parser.add_argument('--apply', action='store_true')
args = parser.parse_args()
root, output = Path('/project'), Path('/output')
assert output.stat().st_uid == output.stat().st_gid == 1000


def sha(path):
    with path.open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()


def files(path):
    for base, dirs, names in os.walk(path, followlinks=False):
        for name in names:
            yield Path(base) / name


def measure(path):
    size, count = 0, 0
    assert path.is_dir() and not path.is_symlink()
    for f in files(path):
        s = f.lstat()
        assert s.st_uid == 1000, f
        size += s.st_size
        count += 1
    return size, count


rows, skipped = [], []
for prefix in ['workplace', 'dist-all']:
    for base, dirs, names in os.walk(root / prefix, followlinks=False):
        for name in list(dirs):
            if name not in ['gocache', 'go-cache', 'wine-prefix']:
                continue
            path = Path(base) / name
            dirs.remove(name)
            if path.is_symlink():
                continue
            if name == 'wine-prefix' and any(
                f.suffix.lower() == '.sav' or f.name.upper() == 'DX5.DAT'
                for f in files(path)):
                skipped.append({'path': str(path.relative_to(root)), 'reason': 'contains potential game save'})
                continue
            size, count = measure(path)
            rows.append({'path': str(path.relative_to(root)), 'kind': 'rebuildable_cache', 'bytes': size, 'files': count})

# A package directory is disposable only when its exact manifest and every
# file match a retained, hashed ZIP and its sidecar manifest.
indexes = {}
for version_root in (root / 'dist-all').iterdir():
    index = version_root / 'SHA256SUMS.json'
    if index.is_file():
        for entry in json.loads(index.read_text())['packages']:
            archive = version_root / entry['file']
            indexes[archive.stem] = (archive, entry)
candidates = list((root / 'workplace/desktop-prototype/package').glob('build-v*/stage/*'))
candidates += list((root / 'dist-all').glob('*/smoke/*/unpacked/*'))
for path in candidates:
    if not path.is_dir() or path.name not in indexes:
        continue
    archive, entry = indexes[path.name]
    manifest = path / 'manifest.json'
    try:
        assert manifest.is_file() and sha(manifest) == entry['manifest_sha256']
        assert sha(archive) == entry['sha256']
        m = json.loads(manifest.read_text())
        actual = {str(f.relative_to(path)) for f in files(path)}
        assert actual == set(m['files']) | {'manifest.json'}
        for name, record in m['files'].items():
            f = path / name
            assert f.is_file() and not f.is_symlink()
            assert f.stat().st_size == record['size'] and sha(f) == record['sha256']
        size, count = measure(path)
    except (AssertionError, FileNotFoundError) as error:
        skipped.append({'path': str(path.relative_to(root)), 'reason': 'package copy not proven recoverable'})
        continue
    rows.append({'path': str(path.relative_to(root)), 'kind': 'verified_package_copy', 'bytes': size,
                 'files': count, 'retained_archive': str(archive.relative_to(root)), 'archive_sha256': entry['sha256']})

assert len({r['path'] for r in rows}) == len(rows)
removed = []
if args.apply:
    active = subprocess.check_output(['git', '-c', 'safe.directory=/project', '-C', str(root),
                                      'status', '--porcelain'], text=True)
    assert not active, '先提交清理工具與文件，避免誤判工作樹'
    for row in rows:
        path = root / row['path']
        assert path.is_relative_to(root / 'workplace') or path.is_relative_to(root / 'dist-all')
        assert path.stat().st_uid == path.stat().st_gid == 1000 and not path.is_symlink()
        shutil.rmtree(path)
        assert not path.exists()
        removed.append(row['path'])
report = {'result': 'APPLIED' if args.apply else 'DRY_RUN',
          'time_utc': datetime.datetime.now(datetime.timezone.utc).isoformat(),
          'selected': rows, 'selected_bytes': sum(r['bytes'] for r in rows),
          'selected_files': sum(r['files'] for r in rows), 'removed': removed, 'skipped': skipped,
          'preserved': ['original', 'assets', 'source snapshots', 'IDA databases', 'receipts',
                        'game saves', 'screenshots', 'recordings', 'ZIP archives']}
target = output / ('workspace-cleanup-v100.json' if args.apply else 'workspace-cleanup-v100-plan.json')
assert not target.exists(), '拒絕覆寫既有清理收據'
target.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
print(json.dumps({k: v for k, v in report.items() if k not in ['selected', 'skipped', 'preserved']}, ensure_ascii=False), flush=True)
