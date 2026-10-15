package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/wicanr2/dosgolem/internal/cpu386"
	"github.com/wicanr2/dosgolem/internal/machine"
)

// Synthetic one-instruction capability screening, never a game execution receipt.
type instructionProbeBus []byte

func (b instructionProbeBus) Read8(addr uint32) (byte, error) {
	if uint64(addr) >= uint64(len(b)) {
		return 0, fmt.Errorf("synthetic read outside bus: %X", addr)
	}
	return b[addr], nil
}
func (b instructionProbeBus) Write8(addr uint32, value byte) error {
	if uint64(addr) >= uint64(len(b)) {
		return fmt.Errorf("synthetic write outside bus: %X", addr)
	}
	b[addr] = value
	return nil
}

func main() {
	cpuSource := flag.String("cpu-source", "", "exact CPU source used by the overlay")
	rowsPath := flag.String("rows", "/output/range-1f400-1f800.json", "bounded IDA instruction rows for synthetic screening")
	flag.Parse()
	if *cpuSource == "" {
		panic("cpu-source is required")
	}
	input, err := os.ReadFile("/original/DX5.EXE")
	if err != nil {
		panic(err)
	}
	inputSHA := fmt.Sprintf("%x", sha256.Sum256(input))
	if inputSHA != "2af4ad61eaf6126bc2b09b7df6024b8e710dafc5085ea0558daea744d794079a" {
		panic("DX5 source mismatch")
	}
	loaded, err := machine.LoadLE(input)
	if err != nil {
		panic(err)
	}
	data, err := os.ReadFile(*rowsPath)
	if err != nil {
		panic(err)
	}
	var inventory struct {
		Rows []struct {
			EA       string `json:"ida_linear_ea"`
			Bytes    string `json:"bytes"`
			Listing  string `json:"listing"`
			Function string `json:"original_ida_function"`
		} `json:"rows"`
	}
	if err = json.Unmarshal(data, &inventory); err != nil {
		panic(err)
	}
	if len(inventory.Rows) == 0 || len(inventory.Rows) > 8192 {
		panic("IDA row count outside bounded screening limit")
	}
	var failures []map[string]any
	bus := make(instructionProbeBus, 2<<20)
	checked := 0
	for _, row := range inventory.Rows {
		code, err := hex.DecodeString(row.Bytes)
		if err != nil {
			panic(err)
		}
		ea, err := strconv.ParseUint(row.EA, 0, 64)
		if err != nil || ea+uint64(len(code)) > uint64(len(loaded.Mem)) || string(loaded.Mem[ea:ea+uint64(len(code))]) != string(code) {
			panic("IDA and dosgolem loaded bytes mismatch: " + row.EA)
		}
		if row.Function == "" || len(code) == 0 || len(code) > 15 {
			continue
		}
		clear(bus[:64])
		copy(bus, code)
		c := cpu386.New(bus)
		for _, segment := range []int{cpu386.SegDS, cpu386.SegES, cpu386.SegSS} {
			c.Seg[segment] = 16
		}
		c.SetDescriptor(16, cpu386.Descriptor{Limit: uint32(len(bus) - 1), Writable: true})
		c.R[cpu386.EAX], c.R[cpu386.EBX], c.R[cpu386.ECX] = 1, 1024, 3
		for _, reg := range []int{cpu386.ESP, cpu386.EBP, cpu386.ESI, cpu386.EDI} {
			c.R[reg] = 1 << 20
		}
		if err = c.Step(); err != nil {
			failures = append(failures, map[string]any{"ida_linear_ea": row.EA, "bytes": row.Bytes, "listing": row.Listing, "synthetic_error": err.Error()})
		}
		checked++
	}
	report := map[string]any{
		"scope":        "synthetic instruction-form screening; not a normal game path or semantic proof",
		"input_sha256": inputSHA, "go_version": runtime.Version(), "IDA_rows_sha256": fmt.Sprintf("%x", sha256.Sum256(data)),
		"IDA_rows_path":            *rowsPath,
		"IDA_loaded_rows_verified": len(inventory.Rows), "synthetic_instructions_checked": checked, "failures": failures,
		"dosgolem_head": "a9714ebdab2ad6b529f81225472680f2b11f2842",
	}
	for label, path := range map[string]string{"cpu_source": *cpuSource, "probe_source": "/project/tools/battle-instruction-probe.go"} {
		bytes, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		report[label+"_path"], report[label+"_sha256"] = path, fmt.Sprintf("%x", sha256.Sum256(bytes))
	}
	result, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile("/output/battle-instruction-capabilities.json", append(result, '\n'), 0600); err != nil {
		panic(err)
	}
	fmt.Printf("%d loaded rows verified; %d synthetic instructions; %d errors for review\n", len(inventory.Rows), checked, len(failures))
}
