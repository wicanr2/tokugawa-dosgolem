"""離線核對四個桌面 ZIP 的內容、平台、架構、授權與雜湊，不冒稱真機試玩。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import plistlib
import struct
import zipfile

assert Path('/.dockerenv').exists() and os.getuid() == 1000
p = argparse.ArgumentParser()
p.add_argument('--version-root', type=Path, required=True)
p.add_argument('--output', type=Path, required=True)
args = p.parse_args()
root = args.version_root
index = json.loads((root / 'SHA256SUMS.json').read_text())
assert {(v['platform'], v['arch']) for v in index['packages']} == {
    ('linux', 'amd64'), ('windows', 'amd64'), ('macos', 'amd64'), ('macos', 'arm64')}
original = json.loads(Path('/project/original/source.json').read_text())
original_hashes = {v['path']: v['sha256'] for v in original['members']}


def digest(stream):
    h = hashlib.sha256()
    for block in iter(lambda: stream.read(1024 * 1024), b''):
        h.update(block)
    return h.hexdigest()


rows = []
for entry in index['packages']:
    archive = root / entry['file']
    with archive.open('rb') as f:
        assert digest(f) == entry['sha256']
    assert archive.stat().st_size == entry['size']
    with zipfile.ZipFile(archive) as z:
        infos = z.infolist()
        assert len(infos) == len({i.filename for i in infos})
        assert all(i.flag_bits & 0x800 for i in infos if not i.filename.isascii())
        tops = {i.filename.split('/')[0] for i in infos}
        assert len(tops) == 1
        prefix = tops.pop() + '/'
        manifest_data = z.read(prefix + 'manifest.json')
        assert hashlib.sha256(manifest_data).hexdigest() == entry['manifest_sha256']
        m = json.loads(manifest_data)
        assert m['version'] == index['version'] and m['commit'] == index['commit']
        assert m['platform'] == entry['platform'] and m['arch'] == entry['arch']
        assert m['normal_playthrough_required'] is False and m['normal_playthrough_verified'] is False
        tested = m['inherited_test_inputs']
        compiled = {name: value for name, value in m['build_sources'].items() if name != 'tools/package-desktop.py'}
        assert compiled and set(compiled) <= set(tested)
        assert all(tested[name] == value for name, value in compiled.items())
        assert set(i.filename[len(prefix):] for i in infos) == set(m['files']) | {'manifest.json'}
        for path, expected in m['files'].items():
            info = z.getinfo(prefix + path)
            assert info.file_size == expected['size']
            with z.open(info) as f:
                assert digest(f) == expected['sha256'], path
        assert z.read(prefix + 'LICENSE') == Path('/project/LICENSE').read_bytes()
        assert len([n for n in m['files'] if n.startswith('licenses/')]) >= 10
        mac = entry['platform'] == 'macos'
        base = 'Tokugawa.app/Contents/Resources/' if mac else ''
        for name in ('DX5.EXE', 'DX5.DAT', 'DX5.cue', 'DX5.bin', 'MARK.EXE.BAK'):
            assert m['files'][base + 'game/' + name]['sha256'] == original_hashes[name], name
        if mac:
            executable = 'Tokugawa.app/Contents/MacOS/tokugawa-dosgolem'
            plist = plistlib.loads(z.read(prefix + 'Tokugawa.app/Contents/Info.plist'))
            assert plist['TokugawaBuildVersion'] == index['version'] and plist['LSMinimumSystemVersion'] == '12.0'
        else:
            executable = 'tokugawa-dosgolem.exe' if entry['platform'] == 'windows' else 'tokugawa-dosgolem'
        data = z.read(prefix + executable)
        assert index['version'].encode() in data and index['commit'].encode() in data
        if entry['platform'] == 'windows':
            assert data[:2] == b'MZ'
            pe = struct.unpack_from('<I', data, 0x3c)[0]
            assert data[pe:pe+4] == b'PE\0\0' and struct.unpack_from('<H', data, pe + 4)[0] == 0x8664
            text = z.read(prefix + 'README.txt')
            assert text.startswith(b'\xef\xbb\xbf') and b'\n' not in text.replace(b'\r\n', b'')
            binary_format = 'PE32+ AMD64'
        elif mac:
            assert data[:4] == b'\xcf\xfa\xed\xfe'
            cpu = struct.unpack_from('<I', data, 4)[0]
            assert cpu == {'amd64': 0x01000007, 'arm64': 0x0100000c}[entry['arch']]
            count, cursor, minimum = struct.unpack_from('<I', data, 16)[0], 32, None
            for _ in range(count):
                command, size = struct.unpack_from('<II', data, cursor)
                assert size >= 8 and cursor + size <= len(data)
                if command == 0x32:
                    minimum = struct.unpack_from('<I', data, cursor + 12)[0]
                elif command == 0x24:
                    minimum = struct.unpack_from('<I', data, cursor + 8)[0]
                cursor += size
            assert minimum == 12 << 16, minimum
            binary_format = 'Mach-O 64 ' + entry['arch']
        else:
            assert data[:6] == b'\x7fELF\x02\x01' and struct.unpack_from('<H', data, 18)[0] == 62
            binary_format = 'ELF64 x86-64'
        if entry['platform'] != 'windows':
            assert z.getinfo(prefix + executable).external_attr >> 16 & 0o111
        rows.append({'platform': entry['platform'], 'arch': entry['arch'], 'format': binary_format,
                     'archive_sha256': entry['sha256'], 'size': entry['size'], 'verified_files': len(m['files']),
                     'original_data_hashes_match': True, 'licenses_present': True, 'version_consistent': True,
                     'verified_test_inputs': len(tested)})
summary = {'result': 'PASS', 'version': index['version'], 'commit': index['commit'], 'packages': rows,
           'classification': 'archive and binary format verification; no native macOS execution',
           'native_macOS_verified': False, 'normal_playthrough_required': False}
args.output.write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(summary, ensure_ascii=False), flush=True)
