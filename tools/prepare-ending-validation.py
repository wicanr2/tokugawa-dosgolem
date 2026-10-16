"""建立隔離結局測試副本；正式 startup-probe.go 與原版資料不改。"""
import hashlib
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess

assert Path('/.dockerenv').is_file() and os.getuid() == 1000
project, output = Path('/project'), Path('/output')
assert output.stat().st_uid == 1000
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
parser = argparse.ArgumentParser()
parser.add_argument('--base', type=Path, required=True)
parser.add_argument('--control', type=Path, required=True)
args = parser.parse_args()
base, control = args.base, args.control
research = project/'workplace/hd-prototype/catalog/remaining/consumers/battle'
assert base.is_dir() and control.is_dir() and base.is_relative_to(research) and control.is_relative_to(research)
receipt = json.loads((control/'receipt.json').read_text())
assert receipt['successful_steps'] == 40000000 and receipt['stop'] == 'instruction_budget'
assert receipt['frame']['raw_indexed_sha256'] == '39a6813ba03cf0517b43d9d04536ff0dbd264f556339959ebca376cb57ff61c8'
original_sources = json.loads((base/'build-sources.json').read_text())
assert len(original_sources) == 106 and receipt['integration_input_sha256'] == original_sources
assert all(sha(project/p) == h for p, h in original_sources.items())
original = (project/'tools/startup-probe.go').read_text()
snapshot = output/'source-snapshot'
snapshot.mkdir(exist_ok=False)
for name in original_sources:
    target = snapshot/name
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy(project/name, target)
shutil.copy(project/'tools/ending-source-probe.go', snapshot/'tools/ending-source-probe.go')
entry_hash = sha(control/'startup-memory.bin')
declaration = '''
	// 僅由 prepare-ending-validation.py 產生的隔離測試副本。
	endingEntry := map[string]any{"validation_only":true,"normal_playthrough_verified":false,
		"classification":"controlled original ending; user authorized acceptance change on 2026-10-16"}
	endingStages := []map[string]any{}
	endingSeen := map[string]bool{}
	endingGate, endingEntered := 0, false
	endingFinishAt, endingFinishSent := -1, false
'''
entry = '''
			if endingGate != 0 && *desktop {
				if _, err := os.Stat(fmt.Sprintf("/output/ending-continue-%03d",endingGate)); os.IsNotExist(err) { return true } else if err != nil { panic(err) }
				endingGate=0
			}
			if steps == 40000000 && !endingEntered {
				if profile.Publisher || fmt.Sprintf("%x",sha256.Sum256(m.Mem)) != "ENTRY_HASH" ||
					hex.EncodeToString(m.Mem[0x101a4:0x101a9]) != "e8fa840000" || hex.EncodeToString(m.Mem[0x186a3:0x186a8]) != "681c000000" { panic("controlled ending entry contract mismatch") }
				endingEntry["step"],endingEntry["before_eip"],endingEntry["after_eip"]=steps,m.CPU.EIP,uint32(0x101a4)
				endingEntry["registers"],endingEntry["segments"],endingEntry["flags"]=m.CPU.R,m.CPU.Seg,m.CPU.EFlags
				endingEntry["ram_sha256_before"]=fmt.Sprintf("%x",sha256.Sum256(m.Mem))
				m.CPU.EIP=0x101a4
				endingEntry["ram_sha256_after"]=fmt.Sprintf("%x",sha256.Sum256(m.Mem))
				endingEntry["ram_bytes_changed"]=0
				endingEntered=true
			}
			if endingEntered && steps==endingFinishAt && !endingFinishSent {
				e:=scheduledMouse{Step:steps,X:320,Y:300,Buttons:1}
				applyMouseEvent(mouseServices,e)
				inputEvents=append(inputEvents,map[string]any{"source":"controlled_validation_finish_input","actual_step":steps,"requested_x":320,"requested_y":300,"buttons":1})
				endingEntry["finish_input_step"]=steps
				endingEntry["finish_input_contract"]="ordinary left button after original final-copy return plus 1000000 successful instructions"
				endingFinishSent=true
			}
'''.replace('ENTRY_HASH', entry_hash)
stages = '''
			if endingEntered {
				label := ""
				switch before {
				case 0x184d0,0x18824:
					if len(graphics.Records)>0 {
						draw:=graphics.Records[len(graphics.Records)-1]
						if draw.Kind=="ending_shift" && draw.Verified && (draw.SourceShift==0 || draw.SourceShift==120 || draw.SourceShift==240) {
							label=fmt.Sprintf("END-2-shift-%03d",draw.SourceShift)
						}
					}
				case 0x1f883:
					name,_:=assetTrace.LastReturn["name"].(string)
					if strings.HasPrefix(name,"END-") && uint64(m.CPU.R[cpu386.ESP])+4<=uint64(len(m.Mem)) {
						ret:=binary.LittleEndian.Uint32(m.Mem[m.CPU.R[cpu386.ESP]:])
						if ret>=0x1872d && ret<0x18db4 { label=fmt.Sprintf("%s-delay-%X",name,ret) }
					}
				case 0x18daf:
					label="END-11B-final-copy"
				}
				if label!="" && !endingSeen[label] {
					if len(endingStages)>=64 { panic("ending checkpoint limit exceeded") }
					endingSeen[label]=true
					snap,err:=snapshotNative(screen,cursor,int(mouseServices.Mouse.X),int(mouseServices.Mouse.Y)); if err!=nil { panic(err) }
					id:=len(endingStages)+1
					path:=fmt.Sprintf("ending-stage-%03d.idx",id)
					if err=os.WriteFile("/output/"+path,snap.Indexed,0600); err!=nil { panic(err) }
					frame,err:=captureFrame(screen,&cursor,int(mouseServices.Mouse.X),int(mouseServices.Mouse.Y)); if err!=nil { panic(err) }
					stage:=map[string]any{"id":id,"label":label,"step":steps,"eip":before,"registers":m.CPU.R,"segments":m.CPU.Seg,"flags":m.CPU.EFlags,
						"ram_sha256":fmt.Sprintf("%x",sha256.Sum256(m.Mem)),"indices_path":path,"indices_sha256":fmt.Sprintf("%x",sha256.Sum256(snap.Indexed)),
						"palette":snap.Palette,"DAC":snap.DAC,"frame":frame,"validation_only":true,"normal_playthrough_verified":false}
					if len(graphics.Records)>0 { stage["last_graphics_return"]=graphics.Records[len(graphics.Records)-1] }
					endingStages=append(endingStages,stage)
					if label=="END-11B-final-copy" { endingFinishAt=steps+1000000 }
					data,err:=json.MarshalIndent(stage,"","  "); if err!=nil { panic(err) }; if err=os.WriteFile("/output/ending-stage.json",data,0600); err!=nil { panic(err) }
					if *desktop { endingGate=id; return true }
				}
			}
'''
replacements = [
    ('// 有界 LE 啟動研究探針。只提供平台 ABI，不跳過遊戲或 C runtime 指令。',
     '// 隔離結局驗收：40M 邊界只改 EIP；非正常通關，不供正式啟動。'),
    ('\tflag.Parse()\n', '\tflag.Parse()\n\t*hdDiagnostic, *graphicsDiagnostic = true, true\n'),
    ('\tadvance := func(maxSteps int) bool {', declaration+'\tadvance := func(maxSteps int) bool {'),
    ('\t\tfor steps < *budget && steps < end {', '\t\tfor steps < *budget && steps < end {'+entry),
    ('\t\t\trow := rawTrace{step: steps,', stages+'\t\t\trow := rawTrace{step: steps,'),
    ('\t\t"normal_player_path_verified": false, "game_screen_verified": false,',
     '\t\t"ending_validation":endingEntry,"ending_validation_stages":endingStages,\n\t\t"normal_player_path_verified": false, "game_screen_verified": false,'),
    ('"native_runtime_instructions_skipped": false', '"native_runtime_instructions_skipped": endingEntered'),
    ('"dosgolem native LE + bounded DOS4G-compatible startup research probe"',
     '"controlled original ending validation; not normal playthrough"'),
]
modified = original
for before, after in replacements:
    assert modified.count(before) == 1, before
    modified = modified.replace(before, after)
target = snapshot/'tools/startup-probe.go'
target.write_text(modified)
subprocess.run(['gofmt', '-w', str(target)], check=True)
sources = {p: sha(snapshot/p) for p in original_sources}
(output/'build-sources.json').write_text(json.dumps(sources, indent=2)+'\n')
shutil.copytree(base/'engine', output/'engine')
(output/'startup-probe').mkdir()
shutil.copy(base/'startup-probe/repro-overlay.json', output/'startup-probe/repro-overlay.json')
manifest = dict(validation_only=True, normal_playthrough_verified=False, entry_step=40000000,
                control_relative=str(control.relative_to(project)), base_relative=str(base.relative_to(project)),
                control_ram_sha256=entry_hash, original_callsite_linear=0x101a4, original_ending_linear=0x186a3,
                mutation='CPU EIP only; no RAM, EXE, DAT or save writes by the test hook',
                finish_input=dict(after_final_copy_instructions=1000000, x=320, y=300, buttons=1,
                                  classification='ordinary deterministic mouse input; original sub_1BB48 exits naturally'),
                original_startup_sha256=original_sources['tools/startup-probe.go'],
                controlled_startup_sha256=sources['tools/startup-probe.go'],
                prepare_tool_sha256=sha(Path(__file__)), original_compiled_sources=original_sources,
                controlled_compiled_sources=sources)
(output/'validation-manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n')
print('controlled source snapshot prepared; production sources unchanged')
