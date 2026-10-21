"""固定 Docker 工具鏈的本機桌面封包；含原版資料，只能留在 full-local。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import zipfile

assert Path('/.dockerenv').exists() and os.getuid() == 1000
parser = argparse.ArgumentParser()
parser.add_argument('--version', required=True)
parser.add_argument('--commit', required=True)
parser.add_argument('--asset-audit', type=Path, required=True)
parser.add_argument('--platforms', choices=['linux-windows', 'macos'], required=True)
parser.add_argument('--prepared-dir', type=Path, required=True)
args = parser.parse_args()
assert re.fullmatch(r'v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}', args.version)
assert re.fullmatch(r'[0-9a-f]{40}', args.commit)
project, work, delivery = Path('/project'), Path('/output'), Path('/delivery')
assert all(p.is_dir() for p in (project, work, delivery, Path('/original'), Path('/dosgolem')))
assert all(p.stat().st_uid == 1000 for p in (work, delivery))
assert subprocess.check_output(['git', '-C', str(project), 'rev-parse', 'HEAD'], text=True).strip() == args.commit
subprocess.run(['git', '-C', str(project), 'diff', '--exit-code', 'HEAD'], check=True, stdout=subprocess.DEVNULL)
version_dir = delivery / args.version
if args.platforms == 'linux-windows':
    assert not version_dir.exists(), '拒絕覆寫既有版本'
else:
    assert version_dir.is_dir(), '先在 GUI image 建立 Linux／Windows 封包'


def sha(path):
    h = hashlib.sha256()
    with path.open('rb') as f:
        for data in iter(lambda: f.read(1024 * 1024), b''):
            h.update(data)
    return h.hexdigest()


def dump(path, value):
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')


def copy(source, target):
    assert source.is_file() and not source.is_symlink(), source
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, target)
    target.chmod(0o644)


env = dict(os.environ, GOMAXPROCS='2', GOFLAGS='-p=2')
# Preparation and full tests are reused only when every recorded input matches.
prepared = args.prepared_dir
frozen = json.loads((prepared / 'build-sources.json').read_text())
prepared_overlay = json.loads((prepared / 'startup-probe/repro-overlay.json').read_text())['Replace']
required_inputs = {str(Path(p).relative_to(project)) for p in prepared_overlay.values() if p.startswith('/project/')}
required_inputs.update('tools/' + name for name in ('dosgolem.patch', 'mark-cpu.patch', 'dos4g-exception.patch',
                      'battle-cpu.patch', 'run-probes.sh', 'run-desktop.sh', 'desktop.mod', 'desktop.sum',
                      'cpu386-integration_test.go', 'toolbar-font.hex.gz', 'LICENSE-toolbar-font.txt'))
assert required_inputs <= set(frozen), sorted(required_inputs - set(frozen))
assert all(sha(project / p) == digest for p, digest in frozen.items())
if not (work / 'engine').exists():
    shutil.copytree(prepared / 'engine', work / 'engine')
    (work / 'startup-probe').mkdir()
    copy(prepared / 'startup-probe/repro-overlay.json', work / 'startup-probe/repro-overlay.json')
    copy(prepared / 'startup-probe/platform-test.log', work / 'startup-probe/inherited-platform-test.log')
copy(project / 'tools/desktop.mod', work / 'desktop.mod')
copy(project / 'tools/desktop.sum', work / 'desktop.sum')
audit = json.loads(args.asset_audit.read_text())
assert audit['resource_root'] == '/project' and audit['original_root'] == '/original'
resources = sorted({Path(p).relative_to(project) for p in audit['paths'] if p.startswith('/project/')})
assert len(resources) > 600 and Path('assets/hd-b/art/RAINEND-hd-b-v1.png') in resources
overlay = json.loads((work / 'startup-probe/repro-overlay.json').read_text())['Replace']
source_paths = {Path(p).relative_to(project) for p in overlay.values() if p.startswith('/project/')}
source_paths.update(Path('tools') / p for p in ('dosgolem.patch', 'mark-cpu.patch', 'dos4g-exception.patch', 'battle-cpu.patch', 'run-probes.sh', 'desktop.mod', 'desktop.sum'))
source_hashes = {str(p): sha(project / p) for p in sorted(source_paths)}
source_hashes['tools/package-desktop.py'] = sha(project / 'tools/package-desktop.py')
dump(work / 'build-sources.json', source_hashes)
for path in source_paths:
    copy(project / path, work / 'source-snapshot' / path)

license_sources = {'Go': Path('/usr/local/go/LICENSE'), 'dosgolem': Path('/dosgolem/LICENSE'),
                   'GNU-Unifont': project / 'tools/LICENSE-toolbar-font.txt'}
modules = {'Ebitengine': 'github.com/hajimehoshi/ebiten/v2@v2.9.9',
           'Oto': 'github.com/ebitengine/oto/v3@v3.4.0',
           'PureGo': 'github.com/ebitengine/purego@v0.9.0',
           'HideConsole': 'github.com/ebitengine/hideconsole@v1.0.0',
           'XGB': 'github.com/jezek/xgb@v1.1.1',
           'x-sync': 'golang.org/x/sync@v0.17.0', 'x-sys': 'golang.org/x/sys@v0.36.0'}
license_sources.update({k: Path('/go/pkg/mod') / v / 'LICENSE' for k, v in modules.items()})
version_dir.mkdir(exist_ok=args.platforms == 'macos')
full_local = version_dir / 'full-local'
full_local.mkdir(exist_ok=args.platforms == 'macos')
(version_dir / 'smoke').mkdir(exist_ok=args.platforms == 'macos')
packages = []
if args.platforms == 'linux-windows':
    targets = [('linux', 'amd64', '1'), ('windows', 'amd64', '0')]
else:
    recorded = json.loads((version_dir / 'SHA256SUMS.json').read_text())
    assert recorded['version'] == args.version and recorded['commit'] == args.commit
    packages = recorded['packages']
    assert {(p['platform'], p['arch']) for p in packages} == {('linux', 'amd64'), ('windows', 'amd64')}
    assert all(sha(version_dir / p['file']) == p['sha256'] for p in packages)
    targets = [('darwin', 'amd64', '1'), ('darwin', 'arm64', '1')]
for goos, arch, cgo in targets:
    platform = 'macos' if goos == 'darwin' else goos
    name = f'tokugawa-dosgolem-{args.version}-{platform}-{arch}'
    stage = work / 'stage' / name
    assert not stage.exists()
    stage.mkdir(parents=True)
    if goos == 'darwin':
        contents = stage / 'Tokugawa.app/Contents'
        executable = contents / 'MacOS/tokugawa-dosgolem'
        base = contents / 'Resources'
        contents.mkdir(parents=True, exist_ok=True)
        short_version = args.version.split('-')[0][2:]
        (contents / 'Info.plist').write_text('<?xml version="1.0" encoding="UTF-8"?>\n'
            '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n'
            '<plist version="1.0"><dict><key>CFBundleExecutable</key><string>tokugawa-dosgolem</string>'
            '<key>CFBundleIdentifier</key><string>io.github.wicanr2.tokugawa-dosgolem</string>'
            '<key>CFBundleName</key><string>德川家康</string><key>CFBundlePackageType</key><string>APPL</string>'
            f'<key>CFBundleShortVersionString</key><string>{short_version}</string>'
            f'<key>CFBundleVersion</key><string>{short_version}</string>'
            f'<key>TokugawaBuildVersion</key><string>{args.version}</string>'
            '<key>LSMinimumSystemVersion</key><string>12.0</string><key>NSHighResolutionCapable</key><true/>'
            '</dict></plist>\n')
    else:
        base = stage
        executable = stage / ('tokugawa-dosgolem.exe' if goos == 'windows' else 'tokugawa-dosgolem')
    executable.parent.mkdir(parents=True, exist_ok=True)
    ldflags = f'-X main.buildVersion={args.version} -X main.buildCommit={args.commit}'
    command = ['go', 'build', '-trimpath', '-buildvcs=false', '-mod=readonly', '-modfile=' + str(work / 'desktop.mod'),
               '-tags', 'desktop', '-overlay', str(work / 'startup-probe/repro-overlay.json'),
               '-ldflags', ldflags, '-o', str(executable), './cmd/leprobe']
    with (work / (platform + '-' + arch + '-build.log')).open('wb') as log:
        build_env = dict(env, GOOS=goos, GOARCH=arch, CGO_ENABLED=cgo)
        if goos == 'darwin':
            build_env.update(CC='o64-clang' if arch == 'amd64' else 'oa64-clang',
                             CXX='o64-clang++' if arch == 'amd64' else 'oa64-clang++', MACOSX_DEPLOYMENT_TARGET='12.0')
        subprocess.run(command, cwd='/dosgolem', env=build_env,
                       stdout=log, stderr=subprocess.STDOUT, check=True, timeout=480)
    executable.chmod(0o755)
    for relative in resources:
        copy(project / relative, base / 'runtime' / relative)
    # Provenance is queried by the runtime receipt; keep the exact compiled inputs.
    for relative in source_paths:
        copy(project / relative, base / 'runtime' / relative)
    copy(work / 'engine/internal/cpu386/cpu.go', base / 'runtime/engine/internal/cpu386/cpu.go')
    for filename in ('DX5.EXE', 'DX5.DAT', 'DX5.cue', 'DX5.bin', 'MARK.EXE.BAK'):
        copy(Path('/original') / filename, base / 'game' / filename)
    copy(project / 'LICENSE', stage / 'LICENSE')
    for label, source in license_sources.items():
        copy(source, stage / 'licenses' / (label + '.txt'))
    readme = f'''德川家康 × dosgolem {args.version}

本機完整版，含原版資料，不得上傳或轉散布。
Windows：啟動 tokugawa-dosgolem.exe。
Linux：執行 ./tokugawa-dosgolem。需要圖形桌面、OpenGL 與 ALSA。
macOS：開啟 Tokugawa.app。此包未簽章；尚未以 macOS 真機驗收。

啟動時先選原版或 HD，依序播放出版商標誌與進入遊戲。
滑鼠移到最上方或按 Esc 可展開功能列，切換原版／HD。
F6 切換 HD，F7 暫停，F9 截圖。音樂沿用原版 CD。
預設執行速度為 4×；功能列「執行」可選 1×、2×、4×、8×。
音樂依音效裝置固定速度播放，不隨遊戲加速改變音調或丟棄樣本。
遊戲內正常儲存進度即可，資料寫入使用者目錄，重新啟動可讀回。
Linux：$XDG_DATA_HOME/tokugawa-dosgolem，未設定時 ~/.local/share/tokugawa-dosgolem。
Windows：%LOCALAPPDATA%\\tokugawa-dosgolem。
macOS：~/Library/Application Support/tokugawa-dosgolem。
不要移動或刪除 runtime、game 或應用程式包內的資源。
原版文字仍為繁體中文；簡中、英文、日文尚未接入。

自製程式採 source-available RRSAL-1.0，詳見 LICENSE。
原版資料與第三方元件各依原權利人條件，不由本專案授權。
來源：https://github.com/wicanr2/tokugawa-dosgolem
來源提交：{args.commit}
'''
    if goos == 'windows':
        (stage / 'README.txt').write_bytes(b'\xef\xbb\xbf' + readme.replace('\n', '\r\n').encode())
    else:
        (stage / 'README.txt').write_text(readme)
    files = {str(p.relative_to(stage)): {'size': p.stat().st_size, 'sha256': sha(p)}
             for p in sorted(stage.rglob('*')) if p.is_file()}
    manifest = {'version': args.version, 'commit': args.commit, 'platform': platform, 'arch': arch,
                'toolchain': subprocess.check_output(['go', 'version'], text=True).strip(),
                'base_dosgolem_commit': 'a9714ebdab2ad6b529f81225472680f2b11f2842',
                'classification': 'full-local; contains original copyrighted data; do not distribute',
                'runtime_resource_count': len(resources), 'runtime_audit_sha256': sha(args.asset_audit),
                'CGO_ENABLED': cgo, 'compiler': build_env.get('CC', 'system gcc' if cgo == '1' else 'Go'),
                'inherited_test_inputs': frozen, 'inherited_test_log_sha256': sha(prepared / 'startup-probe/platform-test.log'),
                'build_sources': source_hashes, 'files': files,
                'normal_playthrough_verified': False, 'normal_playthrough_required': False,
                'macos_native_verified': False, 'listening_verified': False}
    dump(stage / 'manifest.json', manifest)
    archive = full_local / (name + '.zip')
    with zipfile.ZipFile(archive, 'x', zipfile.ZIP_DEFLATED, compresslevel=6) as z:
        for path in sorted(stage.rglob('*')):
            if path.is_file():
                z.write(path, name + '/' + str(path.relative_to(stage)))
    with zipfile.ZipFile(archive) as z:
        assert z.testzip() is None
        assert all(i.flag_bits & 0x800 for i in z.infolist() if not i.filename.isascii())
    copy(stage / 'manifest.json', full_local / (name + '-manifest.json'))
    packages.append({'file': str(archive.relative_to(version_dir)), 'size': archive.stat().st_size,
                     'sha256': sha(archive), 'platform': platform, 'arch': arch,
                     'manifest_sha256': sha(stage / 'manifest.json')})
    print('PACKAGED', name, flush=True)
dump(version_dir / 'SHA256SUMS.json', {'version': args.version, 'commit': args.commit, 'packages': packages})
assert source_hashes == {p: sha(project / p) for p in source_hashes}
print('PASS: requested local packages; platform smoke validation is separate', flush=True)
