// 原版 LE 入口的有界研究探針。遇到未支援中斷即停，不套用其他遊戲的 runtime。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"

	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
)

type instruction struct {
	Step      int       `json:"step"`
	LinearEIP string    `json:"linear_eip"`
	Bytes     string    `json:"bytes"`
	Registers [8]uint32 `json:"registers_eax_ecx_edx_ebx_esp_ebp_esi_edi"`
	Segments  [6]uint16 `json:"segments"`
}

func main() {
	data, err := os.ReadFile("/original/DX5.EXE")
	if err != nil {
		panic(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	if digest != "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a" {
		panic("DX5.EXE hash mismatch")
	}
	m, err := machine.LoadLE(data)
	if err != nil {
		panic(err)
	}
	entry := m.CPU.EIP
	interrupts := []map[string]any{}
	m.CPU.IntHook = func(c *cpu386.CPU, number uint8) bool {
		interrupts = append(interrupts, map[string]any{
			"number":               fmt.Sprintf("0x%02X", number),
			"linear_eip_after_int": fmt.Sprintf("0x%X", c.EIP),
			"registers_eax_ecx_edx_ebx_esp_ebp_esi_edi": c.R,
			"segments": c.Seg,
			"handled":  false,
		})
		return false
	}
	trace := []instruction{}
	steps := 0
	stop := "instruction_budget"
	var before uint32
	for steps < 10000 {
		before = m.CPU.EIP
		bytes := []byte{}
		for offset := uint32(0); offset < 16; offset++ {
			b, err := m.Read8(before + offset)
			if err != nil {
				break
			}
			bytes = append(bytes, b)
		}
		trace = append(trace, instruction{
			Step: steps, LinearEIP: fmt.Sprintf("0x%X", before), Bytes: hex.EncodeToString(bytes),
			Registers: m.CPU.R, Segments: m.CPU.Seg,
		})
		if len(trace) > 64 {
			trace = trace[1:]
		}
		if err := m.CPU.Step(); err != nil {
			stop = err.Error()
			break
		}
		steps++
	}
	script, err := os.ReadFile("/project/tools/le-entry-probe.go")
	if err != nil {
		panic(err)
	}
	receipt := map[string]any{
		"input_file": "DX5.EXE", "input_sha256": digest,
		"tool":         "dosgolem LoadLE + bounded raw-entry research probe",
		"probe_sha256": fmt.Sprintf("%x", sha256.Sum256(script)), "go_version": runtime.Version(),
		"address_space":    "dosgolem LE relocated flat linear memory; selectors are loader labels",
		"entry_linear_eip": fmt.Sprintf("0x%X", entry), "successful_steps": steps,
		"instruction_budget": 10000, "stop_instruction_linear_eip": fmt.Sprintf("0x%X", before),
		"current_linear_eip": fmt.Sprintf("0x%X", m.CPU.EIP), "stop": stop,
		"runtime_shims": []string{}, "interrupts": interrupts, "trace_tail": trace,
		"normal_player_path_verified": false, "game_screen_verified": false,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(receipt); err != nil {
		panic(err)
	}
}
