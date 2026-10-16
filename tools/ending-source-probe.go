package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
)

// 隔離的來源證據；不啟動遊戲，不產生正常玩家路徑收據。
type endingProbeBus struct {
	mem    []byte
	writes int
}

func (b *endingProbeBus) Read8(addr uint32) (byte, error) {
	if uint64(addr) >= uint64(len(b.mem)) {
		return 0, fmt.Errorf("read outside bus: %X", addr)
	}
	return b.mem[addr], nil
}
func (b *endingProbeBus) Write8(addr uint32, value byte) error {
	for bit := 0; bit < 4; bit++ {
		base := uint32(0x100000 + bit*0x10000)
		if addr >= base && addr < base+12000 {
			b.mem[addr] = value
			b.writes++
			return nil
		}
	}
	return fmt.Errorf("unexpected write outside four 12000-byte regions: %X", addr)
}

func endingDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }
func endingMustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}
func endingWrite(name string, data []byte) {
	f, err := os.OpenFile(filepath.Join("/output", name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	if _, err = f.Write(data); err != nil {
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
}

func main() {
	cpuPath := flag.String("cpu-source", "", "exact patched CPU source")
	flag.Parse()
	if *cpuPath == "" {
		panic("cpu-source required")
	}
	exe := endingMustRead("/original/DX5.EXE")
	if endingDigest(exe) != "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a" {
		panic("EXE mismatch")
	}
	loaded, err := machine.LoadLE(exe)
	if err != nil {
		panic(err)
	}
	proofs := []map[string]any{}
	for _, name := range []string{"range-184d0-185a7.json", "range-185a7-189c8.json", "range-11b60-11ba6.json"} {
		path := "/project/workplace/hd-prototype/catalog/remaining/consumers/ida/" + name
		data := endingMustRead(path)
		var doc struct {
			InputSHA      string `json:"input_sha256"`
			Tool, Version string
			Rows          []struct {
				EA    string `json:"ida_linear_ea"`
				Bytes string
			}
		}
		if json.Unmarshal(data, &doc) != nil || doc.InputSHA != endingDigest(exe) || doc.Tool != "IDA Pro" || doc.Version != "9.4" || len(doc.Rows) == 0 {
			panic("IDA schema mismatch")
		}
		for _, row := range doc.Rows {
			ea, e := strconv.ParseUint(row.EA, 0, 64)
			code, x := hex.DecodeString(row.Bytes)
			if e != nil || x != nil || ea+uint64(len(code)) > uint64(len(loaded.Mem)) || string(loaded.Mem[ea:ea+uint64(len(code))]) != string(code) {
				panic("IDA / loaded bytes mismatch: " + row.EA)
			}
		}
		proofs = append(proofs, map[string]any{"path": path, "sha256": endingDigest(data), "loaded_rows_verified": len(doc.Rows)})
	}
	original := endingMustRead("/project/workplace/hd-prototype/catalog/decoded/END-2.OZM.idx")
	if len(original) != 256000 || endingDigest(original) != "b2aa5040cba120939881b9c1ae7bafa1de492aece744bce745df66c3f83b7682" {
		panic("source mismatch")
	}
	bus := &endingProbeBus{mem: make([]byte, 2<<20)}
	copy(bus.mem, loaded.Mem)
	for bit, global := range []int{0x38c00, 0x38bf8, 0x38c04, 0x38bfc} {
		base := 0x100000 + bit*0x10000
		binary.LittleEndian.PutUint32(bus.mem[global:], uint32(base))
		for i, value := range original {
			bus.mem[base+i/8] |= ((value >> uint(bit)) & 1) << uint(7-i%8)
		}
	}
	states := []map[string]any{}
	total := 0
	for count := 0; count <= 240; count++ {
		if count > 0 {
			cpu := cpu386.New(bus)
			for _, seg := range []int{cpu386.SegDS, cpu386.SegES, cpu386.SegSS} {
				cpu.Seg[seg] = 16
			}
			cpu.SetDescriptor(16, cpu386.Descriptor{Limit: uint32(len(bus.mem) - 1), Writable: true})
			cpu.EIP = 0x184e6 // 明示跳過 __CHK 與暫存器序言的合成測試。
			steps := 0
			for cpu.EIP != 0x185a2 && steps < 400000 {
				if err = cpu.Step(); err != nil {
					panic(err)
				}
				steps++
			}
			if cpu.EIP != 0x185a2 {
				panic("instruction budget exhausted")
			}
			total += steps
		}
		current := make([]byte, 256000)
		packed := make([]byte, 128000)
		for bit := 0; bit < 4; bit++ {
			plane := bus.mem[0x100000+bit*0x10000 : 0x100000+bit*0x10000+32000]
			copy(packed[bit*32000:], plane)
			for i := range current {
				current[i] |= ((plane[i/8] >> uint(7-i%8)) & 1) << uint(bit)
			}
		}
		for i, value := range current {
			from := i
			if i < 96000 {
				from = min(i+count, 96000)
			}
			if value != original[from] {
				panic(fmt.Sprintf("independent transformation differs: %d/%d", count, i))
			}
		}
		crop := make([]byte, 400*150)
		for y := 0; y < 150; y++ {
			copy(crop[y*400:], current[y*640:y*640+400])
		}
		state := map[string]any{"count": count, "indices_sha256": endingDigest(current), "display_crop_sha256": endingDigest(crop), "planes_sha256": endingDigest(packed)}
		if count == 0 || count == 1 || count == 120 || count == 240 {
			name := fmt.Sprintf("ending-count-%03d.planes", count)
			endingWrite(name, packed)
			state["planes_path"] = name
		}
		states = append(states, state)
	}
	if bus.writes != 240*4*12000 {
		panic("write count differs")
	}
	report := map[string]any{"verified": true, "scope": "original instruction body in synthetic RAM; no normal ending or GUI claim", "input_sha256": endingDigest(exe), "source_indices_sha256": endingDigest(original), "tool": "dosgolem cpu386", "go_version": runtime.Version(), "dosgolem_head": "a9714ebdab2ad6b529f81225472680f2b11f2842", "cpu_source_sha256": endingDigest(endingMustRead(*cpuPath)), "probe_source_sha256": endingDigest(endingMustRead("/project/tools/ending-source-probe.go")), "IDA_exports": proofs, "instruction_body_linear_range": []uint32{0x184e6, 0x185a2}, "instructions_executed": total, "writes_verified": bus.writes, "states": states, "normal_ending_verified": false}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	endingWrite("ending-source-verification.json", append(data, '\n'))
	fmt.Printf("PASS: %d states, %d original instructions, %d bounded writes\n", len(states), total, bus.writes)
}
