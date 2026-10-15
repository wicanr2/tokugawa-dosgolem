"""產生隔離 MARK 入口研究工具。沒有正式啟動接入，也不假裝執行 COMMAND.COM。"""
import hashlib
import json
from pathlib import Path
import os
import shutil
import subprocess

assert Path('/.dockerenv').exists() and os.getuid() == 1000
root = Path('/output')
assert root.stat().st_uid == 1000
baseline = Path('/project/workplace/hd-prototype/catalog/remaining/partial-story/runtime')
source = Path('/project/tools/startup-probe.go').read_text()
source = source.replace('flag.Parse()', '''flag.Parse()
    if *program != "mark" || *desktop {
        panic("this research tool requires -program mark and headless mode")
    }''')
source = source.replace('os.ReadFile("/project/tools/startup-probe.go")', 'os.ReadFile("/output/mark-startup-probe.go")')
source = source.replace('"tool":             "dosgolem native LE + bounded DOS4G-compatible startup research probe"',
                        '"tool": "DRAFT standalone MARK native LE research; not restored game startup"')
path = root / 'mark-startup-probe.go'
with path.open('x') as f:
    f.write(source)
overlay = json.loads((baseline / 'startup-probe/repro-overlay.json').read_text())
for original, target in overlay['Replace'].items():
    if target.startswith('/output/engine/'):
        destination = root / target.removeprefix('/output/')
        destination.parent.mkdir(parents=True, exist_ok=True)
        assert destination.parent.stat().st_uid == 1000
        shutil.copyfile(baseline / target.removeprefix('/output/'), destination)
overlay['Replace']['/dosgolem/cmd/leprobe/main.go'] = '/output/mark-startup-probe.go'
shutil.copyfile('/project/tools/cpu386-integration_test.go', root / 'engine/internal/cpu386/integration_test.go')
shutil.copyfile('/dosgolem/internal/machine/dpmi_test.go', root / 'engine/internal/machine/dpmi_test.go')
overlay['Replace']['/dosgolem/internal/machine/dpmi_test.go'] = '/output/engine/internal/machine/dpmi_test.go'
subprocess.run(['git', 'apply', '--check', '/project/tools/mark-cpu.patch'], cwd=root / 'engine', check=True)
subprocess.run(['git', 'apply', '/project/tools/mark-cpu.patch'], cwd=root / 'engine', check=True)
subprocess.run(['git', 'apply', '--check', '/project/tools/dos4g-exception.patch'], cwd=root / 'engine', check=True)
subprocess.run(['git', 'apply', '/project/tools/dos4g-exception.patch'], cwd=root / 'engine', check=True)
overlay['Replace']['/dosgolem/internal/machine/dos4g_vector.go'] = '/project/tools/dos4g-vector.go'
overlay['Replace']['/dosgolem/cmd/leprobe/dos4g_exception.go'] = '/project/tools/dos4g-exception.go'
overlay['Replace']['/dosgolem/cmd/leprobe/dos4g_exception_test.go'] = '/project/tools/dos4g-exception_test.go'
overlay['Replace']['/dosgolem/cmd/leprobe/mark_trace.go'] = '/project/tools/mark-trace.go'
overlay['Replace']['/dosgolem/internal/machine/vbe_timing.go'] = '/project/tools/vbe-timing.go'
for filename, tool in [('program.go','program.go'),('hd_publisher.go','hd-publisher.go'),('hd_publisher_test.go','hd-publisher_test.go')]:
    overlay['Replace']['/dosgolem/cmd/leprobe/'+filename] = '/project/tools/'+tool
(root / 'startup-probe').mkdir()
with (root / 'startup-probe/repro-overlay.json').open('x') as f:
    json.dump(overlay, f, indent=2)
with (root / 'preparation.json').open('x') as f:
    json.dump({'classification': 'DRAFT standalone executable research; no startup integration',
               'input': 'MARK.EXE.BAK', 'probe_sha256': hashlib.sha256(path.read_bytes()).hexdigest(),
               'baseline_overlay_sha256': hashlib.sha256((baseline / 'startup-probe/repro-overlay.json').read_bytes()).hexdigest(),
               'all_game_callbacks_disabled': True}, f, indent=2)
with (root / 'mark-tool-inputs.json').open('x') as f:
    json.dump({name:hashlib.sha256(Path('/project/tools',name).read_bytes()).hexdigest()
               for name in ['mark-probe-prepare.py','mark-trace.go','vbe-timing.go','dos4g-exception.go','dos4g-exception.patch']}, f, indent=2)
print(path)
