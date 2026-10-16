"""由 r27 凍結來源建立剩餘圖像測試副本；不修改正式執行入口。"""
import hashlib
import json
import os
from pathlib import Path
import shutil
import struct
import subprocess

from PIL import Image

assert Path('/.dockerenv').is_file() and os.getuid() == 1000
project, output = Path('/project'), Path('/output')
assert output.stat().st_uid == 1000
research = project/'workplace/hd-prototype/catalog/remaining/consumers/battle'
base, control = research/'r27/runtime', research/'r29/control'
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
read = lambda p: json.loads(p.read_text())
sources = read(base/'build-sources.json')
receipt = read(control/'receipt.json')
assert len(sources) == 106 and receipt['integration_input_sha256'] == sources
assert receipt['successful_steps'] == 80000000 and receipt['stop'] == 'instruction_budget'
assert all(sha(project/p) == value for p, value in sources.items())
memory = (control/'startup-memory.bin').read_bytes()
entry_hash = hashlib.sha256(memory).hexdigest()
title = Image.open(control/Path(receipt['frame']['path']).relative_to('workplace')).convert('RGB')
assert title.size==(640,400)
assert max(max(pixel) for pixel in title.getdata())==251
plans = [(0,0,0,10),(1,0,0,11),(0,1,0,12),(0,3,0,13),
         (0,3,2,14),(1,3,3,15),(0,1,2,16),(1,1,3,17)]
for side, index, flags, resource in plans:
    address = (0x37134 if side else 0x37114)+index*4
    if flags == (3 if side else 2):
        address += 16
    assert struct.unpack_from('<I', memory, address)[0] == resource
assert memory[0x2387e:0x23885].hex() == '6a00e890fbffff'
assert memory[0x2388a:0x23891].hex() == '6a01e884fbffff'
assert memory[0x10b51:0x10b58].hex() == '6a69e81bb40000'
snapshot = output/'source-snapshot'
snapshot.mkdir(exist_ok=False)
for name in sources:
    target = snapshot/name
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(project/name, target)
declaration = '''
	// 隔離美術驗收專用，由 prepare-art-validation.py 產生。
	artPlans := [][4]uint32{{0,0,0,10},{1,0,0,11},{0,1,0,12},{0,3,0,13},{0,3,2,14},{1,3,3,15},{0,1,2,16},{1,1,3,17}}
	artEntries := []map[string]any{}
	artStages := []map[string]any{}
	artSeen := map[string]bool{}
	artCase, artGate := -1, 0
	artNext, artOverride := false, false
	artInitialized, artRepaint := false, false
	artBaseline := map[string]any{}
	artSaveStage := func(label string) {
		if len(artStages)>=64 { panic("controlled art checkpoint quota") }
		snap,err:=snapshotNative(screen,cursor,int(mouseServices.Mouse.X),int(mouseServices.Mouse.Y)); if err!=nil { panic(err) }
		id:=len(artStages)+1
		path:=fmt.Sprintf("ending-stage-%03d.idx",id)
		if err=os.WriteFile("/output/"+path,snap.Indexed,0600); err!=nil { panic(err) }
		frame,err:=captureFrame(screen,&cursor,int(mouseServices.Mouse.X),int(mouseServices.Mouse.Y)); if err!=nil { panic(err) }
		stage:=map[string]any{"id":id,"case":artCase,"label":label,"step":steps,"eip":m.CPU.EIP,"registers":m.CPU.R,"segments":m.CPU.Seg,"flags":m.CPU.EFlags,
			"ram_sha256":fmt.Sprintf("%x",sha256.Sum256(m.Mem)),"indices_path":path,"indices_sha256":fmt.Sprintf("%x",sha256.Sum256(snap.Indexed)),
			"palette":snap.Palette,"DAC":snap.DAC,"frame":frame,"validation_only":true,"normal_playthrough_verified":false}
		if len(graphics.Records)>0 { stage["last_graphics_return"]=graphics.Records[len(graphics.Records)-1] }
		artStages=append(artStages,stage)
		data,err:=json.MarshalIndent(stage,"","  "); if err!=nil { panic(err) }; if err=os.WriteFile("/output/ending-stage.json",data,0600); err!=nil { panic(err) }
		if *desktop { artGate=id }
	}
'''
entry = '''
			if artGate!=0 && *desktop {
				if _,err:=os.Stat(fmt.Sprintf("/output/ending-continue-%03d",artGate)); os.IsNotExist(err) { return true } else if err!=nil { panic(err) }
				artGate=0
			}
			if steps==80000000 && !artInitialized {
				if fmt.Sprintf("%x",sha256.Sum256(m.Mem))!="ENTRY_HASH" { panic("controlled art initial RAM") }
				{
					snap,err:=snapshotNative(screen,cursor,int(mouseServices.Mouse.X),int(mouseServices.Mouse.Y)); if err!=nil { panic(err) }
					for i,want:=range battleFrameNativePalette { if snap.Palette[snap.DAC[i]]!=[3]byte{want.R,want.G,want.B} { panic("controlled art requires complete stable original palette") } }
				}
				artBaseline=map[string]any{"step":steps,"before_eip":m.CPU.EIP,"after_eip":uint32(0x10b51),"registers":m.CPU.R,"segments":m.CPU.Seg,"flags":m.CPU.EFlags,"ram_sha256_before":fmt.Sprintf("%x",sha256.Sum256(m.Mem)),"ram_bytes_changed":0}
				m.CPU.EIP=0x10b51; artInitialized=true; artRepaint=true
			}
			if artNext {
				artNext=false; artCase++
				if artCase>len(artPlans) { stop="controlled_art_complete"; return false }
				e:=map[string]any{"case":artCase,"step":steps,"before_eip":m.CPU.EIP,"registers":m.CPU.R,"segments":m.CPU.Seg,"flags":m.CPU.EFlags,"ram_sha256_before":fmt.Sprintf("%x",sha256.Sum256(m.Mem))}
				writes:=[]map[string]any{}
				if artCase<len(artPlans) {
					p:=artPlans[artCase]
					for i,address:=range []uint32{0x3777a,0x376b9,0x37732} {
						value:=p[1]; if i==0 { value=p[2] }
						old:=binary.LittleEndian.Uint32(m.Mem[address:]); binary.LittleEndian.PutUint32(m.Mem[address:],value)
						writes=append(writes,map[string]any{"linear":address,"before":old,"after":value,"bytes":4})
					}
					m.CPU.EIP=0x2387e; if p[0]==1 { m.CPU.EIP=0x2388a }
					e["resource_id"]=p[3]
				} else { m.CPU.EIP=0x10b51 }
				e["after_eip"],e["writes"],e["ram_sha256_after"]=m.CPU.EIP,writes,fmt.Sprintf("%x",sha256.Sum256(m.Mem))
				artEntries=append(artEntries,e)
			}
			if artCase==len(artPlans) && m.CPU.EIP==0x1bf73 && !artOverride {
				a:=m.CPU.R[cpu386.ESP]+4
				if uint64(a)+4>uint64(len(m.Mem)) || binary.LittleEndian.Uint32(m.Mem[a:])!=105 || binary.LittleEndian.Uint32(m.Mem[a-4:])!=0x10b58 { panic("OP6 original loader argument") }
				e:=artEntries[len(artEntries)-1]
				e["loader_override"]=map[string]any{"step":steps,"eip":m.CPU.EIP,"linear":a,"before":uint32(105),"after":uint32(121),"bytes":4,"ram_sha256_before":fmt.Sprintf("%x",sha256.Sum256(m.Mem))}
				binary.LittleEndian.PutUint32(m.Mem[a:],121)
				e["loader_override"].(map[string]any)["ram_sha256_after"]=fmt.Sprintf("%x",sha256.Sum256(m.Mem))
				artOverride=true
			}
'''.replace('ENTRY_HASH', entry_hash)
stages = '''
			if artRepaint && before==0x10b60 {
				if len(graphics.Records)==0 { panic("original title backdrop absent") }
				r:=graphics.Records[len(graphics.Records)-1]
				if !r.Verified || r.Name!="OP5.OZM" || r.Kind!="opaque_full" { panic("original title backdrop mismatch") }
				artSaveStage("control-title-repaint"); artRepaint=false; artNext=true
				if *desktop { return true }; continue
			}
			if artCase>=0 {
				label:=""
				if artCase<len(artPlans) {
					name:=fmt.Sprintf("ANIME%02d.OZM",artPlans[artCase][3]-9)
					if before==0x1f883 && binary.LittleEndian.Uint32(m.Mem[m.CPU.R[cpu386.ESP]:])==0x23516 && len(graphics.Records)>0 {
						r:=graphics.Records[len(graphics.Records)-1]
						if !r.Verified || r.Name!=name || r.Return!=0x2350c { panic("controlled animation original draw mismatch") }
						label=fmt.Sprintf("%s-frame-%d-%d",name,r.SourceRect.Min.X,r.SourceRect.Min.Y)
					}
					if (artPlans[artCase][0]==0 && before==0x23888) || (artPlans[artCase][0]==1 && before==0x23894) { label=name+"-restored" }
				} else if before==0x10b60 {
					if !artOverride || len(graphics.Records)==0 { panic("OP6 draw absent") }
					r:=graphics.Records[len(graphics.Records)-1]
					if !r.Verified || r.Name!="OP6.OZM" || r.Kind!="opaque_full" { panic("OP6 original full draw mismatch") }
					label="OP6.OZM-full"
				}
				if label!="" && !artSeen[label] {
					artSeen[label]=true; artSaveStage(label)
					if strings.HasSuffix(label,"-restored") || label=="OP6.OZM-full" { artNext=true }
					if *desktop { return true }
					if artNext { continue }
				}
			}
'''
original = (project/'tools/startup-probe.go').read_text()
replacements = [
    ('\tflag.Parse()\n', '\tflag.Parse()\n\t*hdDiagnostic, *graphicsDiagnostic = true, true\n'),
    ('\tadvance := func(maxSteps int) bool {', declaration+'\tadvance := func(maxSteps int) bool {'),
    ('\t\tfor steps < *budget && steps < end {', '\t\tfor steps < *budget && steps < end {'+entry),
    ('\t\t\trow := rawTrace{step: steps,', stages+'\t\t\trow := rawTrace{step: steps,'),
    ('\t\t"normal_player_path_verified": false, "game_screen_verified": false,',
     '\t\t"art_validation_baseline":artBaseline,"art_validation_entries":artEntries,"art_validation_stages":artStages,\n\t\t"normal_player_path_verified": false, "game_screen_verified": false,'),
    ('"native_runtime_instructions_skipped": false', '"native_runtime_instructions_skipped": artCase>=0'),
    ('"dosgolem native LE + bounded DOS4G-compatible startup research probe"', '"controlled original art validation; not normal player path"'),
]
for before, after in replacements:
    assert original.count(before) == 1, before
    original = original.replace(before, after)
target = snapshot/'tools/startup-probe.go'
target.write_text(original)
subprocess.run(['gofmt', '-w', str(target)], check=True)
controlled = {name:sha(snapshot/name) for name in sources}
(output/'build-sources.json').write_text(json.dumps(controlled, indent=2)+'\n')
shutil.copytree(base/'engine', output/'engine')
(output/'startup-probe').mkdir()
shutil.copyfile(base/'startup-probe/repro-overlay.json', output/'startup-probe/repro-overlay.json')
manifest = dict(validation_only=True, normal_playthrough_verified=False, entry_step=80000000,
                control_relative=str(control.relative_to(project)), control_ram_sha256=sha(control/'startup-memory.bin'),
                control_native_indices_sha256=receipt['frame']['raw_indexed_sha256'],
                original_compiled_sources=sources, controlled_compiled_sources=controlled,
                animation_plans=plans, OP6_test_resource_id=121, prepare_tool_sha256=sha(Path(__file__)),
                backdrop_precondition='original OP5 loader/full draw via 0x10b51 clears the title menu layer before animation tests',
                mutation='isolated test EIP and three listed animation dwords; OP6 loader stack parameter 105 to 121; original code and disk data unchanged')
(output/'validation-manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n')
print('controlled art snapshot prepared; production 106 inputs unchanged')
