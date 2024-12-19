package ttd

import (
	"fmt"
	"reflect"
	"slices"
)

func pad(b []byte, l int) []byte {
	return append(b, slices.Repeat([]byte{0}, l-len(b))...)
}

func b(i uint8) []byte {
	return []byte{byte(i)}
}

func w(i uint16) []byte {
	return []byte{byte(i & 0xff), byte((i >> 8) & 0xff)}
}

func l(i uint32) []byte {
	return []byte{byte(i & 0xff), byte((i >> 8) & 0xff), byte((i >> 16) & 0xff), byte((i >> 24) & 0xff)}
}

func ll(i uint64) []byte {
	return []byte{byte(i & 0xff), byte((i >> 8) & 0xff), byte((i >> 16) & 0xff), byte((i >> 24) & 0xff), byte((i >> 32) & 0xff), byte((i >> 40) & 0xff), byte((i >> 48) & 0xff), byte((i >> 56) & 0xff)}
}

func boolb(in bool) []byte {
	i := uint8(0)
	if in {
		i = 1
	}
	return b(i)
}
func boolw(in bool) []byte {
	i := uint16(0)
	if in {
		i = 1
	}
	return w(i)
}

func structToBytes(v reflect.Value) ([]byte, error) {
	var b []byte
	for i := range v.NumField() {
		switch v.Field(i).Type().Name() {
		case "uint16":
			b = append(b, w(uint16(v.Field(i).Uint()))...)
		case "uint32":
			b = append(b, l(uint32(v.Field(i).Uint()))...)
		default:
			return nil, fmt.Errorf("unexpected field type %q", v.Field(i).Type().Name())
		}
	}
	return b, nil
}

func (s *Savegame) writeUncompressed(f OutFile, b []byte) error {
	n, err := f.Write(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		fmt.Errorf("writeUncompressed wrote %d bytes, expected %d", n, len(b))
	}
	s.checkBytes(b)
	return nil
}

func (s *Savegame) writeCompressed(f OutFile, data []byte) error {
	const maxc = 127 + 1
	for i := 0; i < len(data); i += maxc {
		c := len(data) - i
		if c > maxc {
			c = maxc
		}
		b := append([]byte{byte(c - 1)}, data[i:i+c]...)
		err := s.writeUncompressed(f, b)
		if err != nil {
			return err
		}
	}
	return nil
}

func bools(in []bool) []byte {
	b := byte(0)
	for i, v := range in {
		iv := 0
		if v {
			iv = 1
		}
		b += byte(iv << i)
	}
	return []byte{b}
}

func (s *Savegame) Validate() error {
	if len(s.Title) > maxTitleLength {
		return fmt.Errorf("Title too long (%d), max length %d", len(s.Title), maxTitleLength)
	}
	if len(s.TextEffects) > 30 {
		return fmt.Errorf("Too many text effects (%d)", len(s.TextEffects))
	}
	if len(s.Towns) > 70 {
		return fmt.Errorf("Too many towns (%d)", len(s.Towns))
	}
	if len(s.Depots) > 255 {
		return fmt.Errorf("Too many depots (%d)", len(s.Depots))
	}
	if len(s.Companies) > 8 {
		return fmt.Errorf("Too many companies (%d)", len(s.Companies))
	}
	if s.MaxInitialLoan == 0 {
		return fmt.Errorf("openttd will crash if MaxInitialLoan is 0")
	}
	if len(s.Tiles) != NumberOfTiles {
		return fmt.Errorf("Need exactly 0x10000 tiles (256x256), got %d\n", len(s.Tiles))
	}
	return nil
}

func get[T any](array []T, i int, def T) T {
	if i >= len(array) {
		return def
	}
	return array[i]
}

func (s *Savegame) Save(f OutFile) error {
	if err := s.Validate(); err != nil {
		return err
	}

	L1 := make([]byte, NumberOfTiles)
	L2 := slices.Repeat([]byte{0}, NumberOfTiles)
	L3 := slices.Repeat([]byte{0, 0}, NumberOfTiles) // no hedge
	desert := slices.Repeat([]byte{0}, 0x4000)       // desert: normal
	L4 := make([]byte, NumberOfTiles)
	L5 := make([]byte, NumberOfTiles)
	for i, tile := range s.Tiles {
		L4[i] = (tile.Height & 0x0f) | (tile.Class << 4)
		if tile.Class == 0 { // normal
			L1[i] = tile.Owner
			L5[i] = tile.Type & 0x0f
		} else if tile.Class == 2 { // road
			L1[i] = tile.Owner
			L5[i] = tile.Type & 0x0f
		} else if tile.Class == 3 { // building
			L2[i] = tile.Type
		} else if tile.Class == 6 { // water
			L1[i] = 0x11 // owner
			L5[i] = tile.Type
		} else {
			return fmt.Errorf("Unsupported tile class %x\n", tile.Class)
		}
	}

	s.Checksum = 0
	title := pad([]byte(s.Title), maxTitleLength)
	err := s.writeUncompressed(f, slices.Concat(title, w(titleChecksum(title))))
	if err != nil {
		return err
	}

	err = s.writeCompressed(f, slices.Concat(w(s.Days), w(s.FractionalDays)))
	if err != nil {
		return err
	}

	for i := range 0x1e {
		e := get[TextEffect](s.TextEffects, i, TextEffect{ID: 0xFFFF}) // 0xFFFF denotes empty
		v := reflect.ValueOf(&e).Elem()
		b, err := structToBytes(v)
		if err != nil {
			return err
		}
		err = s.writeCompressed(f, b)
		if err != nil {
			return err
		}
	}

	err = s.writeCompressed(f, ll(s.Seed))
	if err != nil {
		return err
	}

	var customStrings []string
	for i := range 0x46 {
		t := get[Town](s.Towns, i, Town{})
		name := uint16(len(customStrings)) + firstCustomTextID
		err = s.writeCompressed(f, slices.Concat(b(t.X), b(t.Y), w(t.Population), w(name), slices.Repeat([]byte{0}, townPlaceholder)))
		if err != nil {
			return err
		}
		customStrings = append(customStrings, t.Name)
	}

	for i := range 0x1388 {
		c := get[uint16](s.Schedules, i, 0)
		s.writeCompressed(f, w(c))
	}
	for i := range 0x100 {
		c := get[uint16](s.Animations, i, 0)
		s.writeCompressed(f, w(c))
	}
	err = s.writeCompressed(f, l(uint32(len(s.Schedules))))
	if err != nil {
		return err
	}

	for i := range 0xff {
		d := get[Depot](s.Depots, i, Depot{})
		v := reflect.ValueOf(&d).Elem()
		b, err := structToBytes(v)
		if err != nil {
			return err
		}
		err = s.writeCompressed(f, b)
		if err != nil {
			return err
		}
	}

	err = s.writeCompressed(f, slices.Concat(
		l(s.NextProcessedTown),
		w(s.AnimationTicker),
		w(s.LandscapeCode),
		w(s.AgeTicker),
		w(s.AnotherAnimationTicker),
		w(s.NextProcessedXY),
		slices.Repeat([]byte{0}, placeholder1),
		L1,
		L2,
		L3,
		desert,
		slices.Repeat([]byte{0}, placeholder2))) // placeholder for stations, industry
	if err != nil {
		return err
	}

	for i := range 8 {
		c := get[Company](s.Companies, i, Company{})
		name := uint16(0)
		manager := uint16(0)
		if i < len(s.Companies) {
			name = uint16(len(customStrings)) + firstCustomTextID
			customStrings = append(customStrings, c.Name)
			manager = uint16(len(customStrings)) + firstCustomTextID
			customStrings = append(customStrings, c.ManagerName)
		}
		err = s.writeCompressed(f, slices.Concat(w(name), l(c.NameParts), l(c.Face), w(manager), l(c.ManagerNameParts), slices.Repeat([]byte{0}, 0x3b2-16)))
	}

	customStringsBytes := make([]byte, 0, 0x20*0x1f4)
	for _, str := range customStrings {
		c := []byte(str)
		if len(c) > 0x20 {
			return fmt.Errorf("Custom string %q exceeds maximum length %d\n", c, 0x20)
		}
		customStringsBytes = append(customStringsBytes, pad(c, 0x20)...)
	}
	customStringsBytes = pad(customStringsBytes, 0x20*0x1f4)

	err = s.writeCompressed(f, slices.Concat(
		slices.Repeat([]byte{0}, placeholder3), // vehicles
		customStringsBytes,
		slices.Repeat([]byte{0xff, 0xff}, 0x1000), // vehicles in bounding blocks
		slices.Repeat([]byte{0x00}, 0xe*0x28),     // signs
		slices.Repeat([]byte{0x00}, 0x1c*0x100),   // vehicle types
		w(s.NextVehicleArray),
		slices.Repeat([]byte{0xFF, 0, 0, 0}, 8), // subsidies
		w(s.AICompanyTicks),
		w(s.MainViewX),
		w(s.MainViewY),
		w(s.Zoom),
		l(s.MaximumLoan),
		l(s.MaximumLoanInternal),
		w(s.RecessionCounter),
		w(s.DaysUntilDisaster),
		slices.Repeat([]byte{0}, placeholder5), // placeholder for pointers to text IDs
		b(s.Player1Company),
		b(s.Player2Company),
		b(s.NextStationTick),
		b(s.Currency),
		b(s.MeasurementSystem),
		b(s.NextCompanyTick),
		b(s.Year),
		b(s.Month),
		slices.Repeat([]byte{0}, 8),
		b(s.Inflation),
		b(s.CargoInflation),
		b(s.InterestRate),
		bools([]bool{s.SmallAirpots, s.LargeAirpots, s.Heliports}),
		bools([]bool{s.DriveOnTheRight, s.DriveOnTheRightFixed}),
		b(s.TownNameStyle),
		w(s.MaximumCompetitors),
		w(s.CompetitorStartTime),
		w(s.NumberOfTowns),
		w(s.NumberOfIndustries),
		w(s.MaxInitialLoan),
		w(s.InitialInterestRate),
		w(s.VehicleRunningCosts),
		w(s.AIConstructionSpeed),
		w(s.AIIntelligence),
		w(s.Breakdowns),
		w(s.SubsidyMultiplier),
		w(s.CostsOfConstruction),
		w(s.TerrianType),
		w(s.QuantityOfLakes),
		boolw(s.FluctuatingEconomy),
		boolw(s.TrainReversingEndOfTheLineOnly),
		boolw(s.Disasters),
		b(s.Difficulty),
		b(s.LandscapeType),
		b(s.TreeTicker),
		bools([]bool{s.CustomVehicleNames, s.CustomVehicleNamesCanBeChanged}),
		b(s.SnowLine),
		slices.Repeat([]byte{0}, placeholder6),
		L4,
		L5,
	))
	if err != nil {
		return err
	}

	s.Checksum += fileChecksumAdd
	n, err := f.Write(l(s.Checksum))
	if err != nil {
		return err
	}
	if n != 4 {
		return fmt.Errorf("wrote %d bytes for the file checksum, expected 4", n)
	}

	return nil
}
