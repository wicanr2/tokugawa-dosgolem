package machine

import (
	"encoding/binary"
	"fmt"

	"github.com/wicanr2/dosgolem/internal/cpu386"
)

// 此檔由 Go overlay 掛入 internal/machine；契約見 RESEARCH-LOG 的 DOS MCB 節。
// 只提供 32 位 DPMI 客戶端的傳統記憶體池，不建立虛構的完整 DOS 系統鏈。
type dosMCBBlock struct {
	seg, size, owner uint16
}

type dosMCBArena struct {
	owner  uint16
	blocks []dosMCBBlock
}

// EnableDOSMCBMemory 在首次 DOS 配置前選用有標頭、可重用的配置模式。
// 未呼叫時保留 dosgolem 原有的單向診斷配置器。
func (h *DPMIHost) EnableDOSMCBMemory(ownerPSP uint16) error {
	if h.m == nil || ownerPSP == 0 || h.dosMCB != nil || len(h.dosBlocks) != 0 || h.realStack != 0 {
		return fmt.Errorf("MCB mode requires an attached, unused DOS allocator and nonzero owner")
	}
	if h.dosBrk >= dosMemTop || h.dosBrk != (uint32(len(h.m.Mem))+15)&^15 {
		return fmt.Errorf("MCB arena has no safe conventional-memory range")
	}
	if len(h.m.Mem) < dosMemTop {
		mem := make([]byte, dosMemTop)
		copy(mem, h.m.Mem)
		h.m.Mem = mem
	}
	seg := uint16(h.dosBrk >> 4)
	h.dosMCB = &dosMCBArena{owner: ownerPSP, blocks: []dosMCBBlock{{seg: seg, size: uint16(dosMemTop/16) - seg - 1}}}
	h.syncDOSMCB()
	return nil
}

func (h *DPMIHost) mcbHeader(i int) [16]byte {
	b := h.dosMCB.blocks[i]
	var out [16]byte
	out[0] = 'M'
	if i == len(h.dosMCB.blocks)-1 {
		out[0] = 'Z'
	}
	binary.LittleEndian.PutUint16(out[1:], b.owner)
	binary.LittleEndian.PutUint16(out[3:], b.size)
	copy(out[8:], "        ")
	return out
}

func (h *DPMIHost) validDOSMCB() bool {
	for i, b := range h.dosMCB.blocks {
		head := h.mcbHeader(i)
		base := int(b.seg) * 16
		if base+16 > len(h.m.Mem) {
			return false
		}
		for j, v := range head {
			if h.m.Mem[base+j] != v {
				return false
			}
		}
	}
	return true
}

func (h *DPMIHost) syncDOSMCB() {
	for i, b := range h.dosMCB.blocks {
		head := h.mcbHeader(i)
		copy(h.m.Mem[int(b.seg)*16:], head[:])
	}
}

func (h *DPMIHost) largestDOSMCB() uint16 {
	var largest uint16
	for _, b := range h.dosMCB.blocks {
		if b.owner == 0 && b.size > largest {
			largest = b.size
		}
	}
	return largest
}

// 分割會留下至少一個 MCB 段；餘下資料段數可以為零。
func (h *DPMIHost) splitDOSMCB(i int, size uint16) {
	a := h.dosMCB
	b := a.blocks[i]
	if b.size == size {
		return
	}
	rest := dosMCBBlock{seg: b.seg + 1 + size, size: b.size - size - 1}
	a.blocks[i].size = size
	a.blocks = append(a.blocks, dosMCBBlock{})
	copy(a.blocks[i+2:], a.blocks[i+1:])
	a.blocks[i+1] = rest
}

func (h *DPMIHost) joinDOSMCB(i int) {
	a := h.dosMCB
	for i+1 < len(a.blocks) && a.blocks[i+1].owner == 0 {
		a.blocks[i].size += 1 + a.blocks[i+1].size
		a.blocks = append(a.blocks[:i+1], a.blocks[i+2:]...)
	}
}

func (h *DPMIHost) allocDOSMCB(paras uint16) (uint32, bool) {
	if paras == 0 || !h.validDOSMCB() {
		return 0, false
	}
	for i, b := range h.dosMCB.blocks {
		if b.owner == 0 && b.size >= paras {
			h.splitDOSMCB(i, paras)
			h.dosMCB.blocks[i].owner = h.dosMCB.owner
			h.syncDOSMCB()
			return uint32(b.seg+1) * 16, true
		}
	}
	return 0, false
}

// 描述子搜尋有界；耗盡時不修改配置池。
func (h *DPMIHost) freeDOSMCBSelector() (uint16, bool) {
	sel := h.nextSel
	for n := 0; n < 8192; n++ {
		if sel >= dpmiFirstSelector {
			if _, exists := h.m.CPU.Descriptors[sel]; !exists {
				return sel, true
			}
		}
		sel += dpmiSelectorStep
	}
	return 0, false
}

func (h *DPMIHost) handleDOSMCB(c *cpu386.CPU, fn uint16) bool {
	if !h.validDOSMCB() {
		return h.fail(c, 7)
	}
	setBX := func(v uint16) { c.R[cpu386.EBX] = c.R[cpu386.EBX]&0xffff0000 | uint32(v) }
	paras := uint16(c.R[cpu386.EBX])
	if fn == 0x0100 {
		if paras == 0 {
			return h.fail(c, 0x8021)
		}
		if paras > h.largestDOSMCB() {
			setBX(h.largestDOSMCB())
			return h.fail(c, 8)
		}
		sel, ok := h.freeDOSMCBSelector()
		if !ok {
			setBX(h.largestDOSMCB())
			return h.fail(c, 0x8011)
		}
		base, ok := h.allocDOSMCB(paras)
		if !ok {
			return h.fail(c, 7)
		}
		h.nextSel = sel + dpmiSelectorStep
		h.m.CPU.SetDescriptor(sel, cpu386.Descriptor{Base: base, Limit: uint32(paras)*16 - 1, Writable: true})
		b := &DOSBlock{Selector: sel, Segment: uint16(base >> 4), Base: base, Paras: paras}
		h.dosBlocks[sel] = b
		c.R[cpu386.EAX] = c.R[cpu386.EAX]&0xffff0000 | uint32(b.Segment)
		c.R[cpu386.EDX] = c.R[cpu386.EDX]&0xffff0000 | uint32(sel)
		return h.ok(c)
	}
	sel := uint16(c.R[cpu386.EDX])
	b, ok := h.dosBlocks[sel]
	if !ok {
		return h.fail(c, 0x8022)
	}
	i := -1
	for j, block := range h.dosMCB.blocks {
		if block.seg+1 == b.Segment && block.owner == h.dosMCB.owner && block.size == b.Paras {
			i = j
			break
		}
	}
	if i < 0 {
		return h.fail(c, 9)
	}
	if fn == 0x0101 {
		h.dosMCB.blocks[i].owner = 0
		h.joinDOSMCB(i)
		if i > 0 && h.dosMCB.blocks[i-1].owner == 0 {
			h.joinDOSMCB(i - 1)
		}
		h.syncDOSMCB()
		delete(h.dosBlocks, sel)
		delete(h.m.CPU.Descriptors, sel)
		for j := range c.Seg {
			if c.Seg[j] == sel {
				c.Seg[j] = 0
			}
		}
		return h.ok(c)
	}
	if paras == 0 {
		return h.fail(c, 0x8021)
	}
	if paras > b.Paras {
		h.joinDOSMCB(i)
	}
	maximum := h.dosMCB.blocks[i].size
	actual := paras
	if actual > maximum {
		actual = maximum
	}
	h.splitDOSMCB(i, actual)
	if i+1 < len(h.dosMCB.blocks) && h.dosMCB.blocks[i+1].owner == 0 {
		h.joinDOSMCB(i + 1)
	}
	b.Paras = actual
	h.m.CPU.SetDescriptor(sel, cpu386.Descriptor{Base: b.Base, Limit: uint32(actual)*16 - 1, Writable: true})
	h.syncDOSMCB()
	if paras > maximum {
		setBX(maximum)
		return h.fail(c, 8)
	}
	return h.ok(c)
}
