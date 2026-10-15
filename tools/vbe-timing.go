package machine

// SetVBEStandardTiming supplies the BIOS timing setup for supported banked modes.
// It reuses the existing VGA timing approximation and leaves pixels/palette intact.
func (v *VGA) SetVBEStandardTiming(mode uint16) bool {
	switch mode {
	case 0x101:
		setModeTiming(&v.crtc, 0x12)
	case 0x100:
		setModeTiming(&v.crtc, 0x0e)
	default:
		return false
	}
	v.seq[1] |= 1
	v.misc &^= 0x0c
	return true
}
