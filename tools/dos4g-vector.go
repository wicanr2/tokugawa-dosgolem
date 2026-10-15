package machine

// ProtectedInterruptVector exposes only the current DOS AH=25h registration.
// It does not install, invent or modify a guest handler.
func (s *FD2StartupDOS) ProtectedInterruptVector(number uint8) (uint16, uint32, bool) {
	vector := s.dosVectors[number]
	return uint16(vector >> 32), uint32(vector), vector != 0
}
