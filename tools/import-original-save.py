"""保存原版程式正常寫入的 FILE 3 工作副本，不建立新存檔格式。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil

assert Path('/.dockerenv').exists() and os.getuid() == 1000
project, root = Path('/project'), Path('/output')
assert root.stat().st_uid == 1000
parser = argparse.ArgumentParser()
parser.add_argument('--source-root', type=Path, required=True)
parser.add_argument('--base-manifest', type=Path, required=True)
args = parser.parse_args()
read = lambda p: json.loads(p.read_text())
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
receipt = read(root / 'receipt.json')
sources = read(root / 'build-sources.json')
assert receipt['integration_input_sha256'] == sources and len(sources) == 106
assert all(sha(args.source_root / name) == digest for name, digest in sources.items())
assert receipt['input_sha256'] == '2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a'
assert receipt['fd2_startup_calls'] == receipt['native_runtime_instructions_skipped'] == 0
assert receipt['original_read_only'] and receipt['stop'] in ['instruction_budget', 'desktop_closed']
schedule = read(root / 'player-path.json')
assert receipt['mouse_input_schedule'] == schedule and len(receipt['mouse_input_events']) == len(schedule)
assert all(event['actual_step'] == request['step'] for event, request in zip(receipt['mouse_input_events'], schedule))
state_path = Path(receipt['file_state_directory'])
assert state_path.parent == Path('/output/startup-probe')
saved_dat = root / 'startup-probe' / state_path.name / 'DX5.DAT'
original_dat = project / 'original/game/DX5.DAT'
old, new = original_dat.read_bytes(), saved_dat.read_bytes()
assert sha(original_dat) == 'de092d5ef59a3e790fd89c323dde1b447c8fe93f430043d1384a1565f27cef37'
inventory = read(project / 'workplace/hd-prototype/asset-inventory.json')
assert inventory['input_sha256'] == sha(original_dat) and inventory['record_count'] == 201
assert len(new) == len(old) and new[:inventory['candidate_table_end']] == old[:inventory['candidate_table_end']]
changed = []
for item in inventory['records']:
    start, length = item['candidate_file_offset'], item['candidate_length']
    before, after = old[start:start + length], new[start:start + length]
    if before != after:
        changed.append(dict(resource=item['name'], file_offset=start, length=length,
                            changed_bytes=sum(a != b for a, b in zip(before, after))))
assert len(changed) == 1 and changed[0]['resource'] == 'I003.SAV'
base = read(args.base_manifest)
assert base['classification'] == 'original program data plus original-game save; not downloaded baseline DAT'
assert base['original_dat_sha256'] == sha(original_dat) and base['saved_dat_sha256'] != sha(saved_dat)
assert len(base['files']) == 12
for item in base['files']:
    path = Path(item['path'])
    assert not path.is_absolute() and '..' not in path.parts
    assert sha(args.base_manifest.parent / 'game' / path) == item['sha256']
    assert sha(project / 'original/game' / path) == item['download_sha256']
target = root / 'save-import'
assert not target.exists(), 'refusing to overwrite an existing original save import'
target.mkdir()
game = target / 'game'
game.mkdir()
manifest = copy.deepcopy(base)
manifest.update(saved_dat_origin=str(saved_dat), saved_dat_sha256=sha(saved_dat),
                parent_manifest_project_path=str(args.base_manifest.relative_to(project)),
                parent_manifest_sha256=sha(args.base_manifest),
                parent_saved_dat_sha256=base['saved_dat_sha256'],
                save_origin_receipt_sha256=sha(root / 'receipt.json'),
                save_origin_input_sha256=sha(root / 'player-path.json'),
                changed_resources_from_download=changed, unchanged_resources=200,
                save_origin_frame=receipt['frame'], import_tool_sha256=sha(Path(__file__)))
for item in manifest['files']:
    source = saved_dat if item['path'] == 'DX5.DAT' else args.base_manifest.parent / 'game' / item['path']
    dest = game / item['path']
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, dest)
    dest.chmod(0o444)
    item['sha256'], item['bytes'] = sha(dest), dest.stat().st_size
    assert dest.stat().st_uid == 1000 and item['sha256'] == sha(source)
for directory in sorted((p for p in game.rglob('*') if p.is_dir()), reverse=True):
    directory.chmod(0o555)
game.chmod(0o555)
(target / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(dict(result='PASS', saved_dat_sha256=manifest['saved_dat_sha256'],
                      changed_resources=changed, unchanged_resources=200), ensure_ascii=False))
