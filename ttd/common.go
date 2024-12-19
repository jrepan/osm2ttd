package ttd

const (
	fileChecksumAdd   = 201100  // for TTD, 105128 for TTO
	maxTitleLength    = 47      // for TTD, 39 for TTO
	NumberOfTiles     = 0x10000 // 256x256
	townPlaceholder   = 0x5e - 6
	firstCustomTextID = 0x7c00 // it seems values outside 0x7c00 - 0x7df4 are special values, such as random names for towns
	placeholder1      = 49*6 + 0xc*8
	placeholder2      = 0x8e*0xfa + 0x36*0x5a
	placeholder3      = 0x80 * 0x352
	placeholder4      = 0xe*0x28 + 0x1c*0x100
	placeholder5      = 6*2*0xc + 2*0x100 + 0x90
	placeholder6      = 0x20 + 3*0xc
	uncompressedSize  = 4 + // days
		0x14*0x1e + // effects
		8 + // seed
		0x5e*0x46 + // towns
		2*0x1388 + // schedules
		2*0x100 + // animations
		4 + // end of schedules
		0x6*0xff + // depots
		14 +
		placeholder1 + // costs, cargo
		6*NumberOfTiles + 0x4000 +
		placeholder2 + // stations, industry
		8*0x3b2 + // companies
		placeholder3 + // vehicles
		0x20*0x1f4 + // custom strings
		0x1000*2 + // vehicles in bounding blocks
		placeholder4 + // signs, vehicle types
		2 + // NextVehicleArray
		32 + // subsidies
		20 +
		placeholder5 + // text IDs, cargo type icons, vehicles for cargo types
		8 + 8 + 6 + 17*2 + 5 +
		placeholder6 // random industry types, cargo types
)

func titleChecksum(title []byte) uint16 {
	// Title checksum
	// This is calculated by adding up all bytes of the title field, rotating the (16-bit) value 1 bit to the left after each addition, then EXORing the resulting value with 0xAAAA.
	var sum uint16 = 0
	for _, b := range title {
		sum += uint16(b)
		sum = (sum << 1) | (sum >> 15) // rotate 1 left
	}
	sum ^= 0xAAAA
	return sum
}

func (s *Savegame) checkBytes(bs []byte) {
	for _, b := range bs {
		s.Checksum += uint32(b)
		s.Checksum = (s.Checksum << 3) | (s.Checksum >> 29)
	}
}
