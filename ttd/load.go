package ttd

import (
	"fmt"
	"reflect"
	"slices"
)

func (s *Savegame) readUncompressed(f InFile, len int) ([]byte, error) {
	b := make([]byte, len)
	n, err := f.Read(b)
	if err != nil {
		return nil, err
	}
	if n != len {
		return nil, fmt.Errorf("readUncompressed: read %d bytes, expected %d", n, len)
	}
	s.checkBytes(b)
	return b, nil
}

func (s *Savegame) readB(f InFile) (byte, error) {
	b, err := s.readUncompressed(f, 1)
	if err != nil {
		return 0, err
	}
	if len(b) != 1 {
		return 0, fmt.Errorf("w: got input length %d, expected 2", len(b))
	}
	return b[0], nil
}

func (s *Savegame) readWBool(f InFile) (bool, error) {
	b, err := s.readW(f)
	if err != nil {
		return false, err
	}
	if b == 0 {
		return false, nil
	}
	return true, nil
}

func (s *Savegame) readW(f InFile) (uint16, error) {
	b, err := s.readUncompressed(f, 2)
	if err != nil {
		return 0, err
	}
	if len(b) != 2 {
		return 0, fmt.Errorf("w: got input length %d, expected 2", len(b))
	}
	return uint16(b[1])<<8 + uint16(b[0]), nil
}

func (s *Savegame) readL(f InFile) (uint32, error) {
	b, err := s.readUncompressed(f, 4)
	if err != nil {
		return 0, err
	}
	if len(b) != 4 {
		return 0, fmt.Errorf("l: got input length %d, expected 4", len(b))
	}
	return uint32(b[3])<<24 + uint32(b[2])<<16 + uint32(b[1])<<8 + uint32(b[0]), nil
}

func (s *Savegame) readLL(f InFile) (uint64, error) {
	b, err := s.readUncompressed(f, 8)
	if err != nil {
		return 0, err
	}
	if len(b) != 8 {
		return 0, fmt.Errorf("l: got input length %d, expected 8", len(b))
	}
	return uint64(b[7])<<56 + uint64(b[6])<<48 + uint64(b[5])<<40 + uint64(b[4])<<32 + uint64(b[3])<<24 + uint64(b[2])<<16 + uint64(b[1])<<8 + uint64(b[0]), nil
}

func (s *Savegame) readCompressed(f InFile, l int) ([]byte, error) {
	out := make([]byte, 0, l)
	for len(out) < l {
		cb, err := s.readB(f)
		c := int8(cb)
		if err != nil {
			return nil, err
		}
		if c >= 0 {
			r, err := s.readUncompressed(f, int(cb+1))
			if err != nil {
				return nil, err
			}
			out = append(out, r...)
		} else {
			b, err := s.readB(f)
			if err != nil {
				return nil, err
			}
			r := slices.Repeat([]byte{b}, int(-c+1))
			out = append(out, r...)
		}
	}
	return out, nil
}

type bytesFile struct {
	data  []byte
	index int
}

func (f *bytesFile) Read(b []byte) (int, error) {
	l := len(b)
	if l > len(f.data)-f.index {
		l = len(f.data) - f.index
	}
	for i := range l {
		b[i] = f.data[i+f.index]
	}
	f.index += l
	return l, nil
}

func (s *Savegame) readStruct(f InFile, v reflect.Value) error {
	for i := range v.NumField() {
		switch v.Field(i).Type().Name() {
		case "uint16":
			r, err := s.readW(f)
			if err != nil {
				return err
			}
			v.Field(i).SetUint(uint64(r))
		case "uint32":
			r, err := s.readL(f)
			if err != nil {
				return err
			}
			v.Field(i).SetUint(uint64(r))
		default:
			return fmt.Errorf("unexpected field type %q", v.Field(i).Type().Name())
		}
	}
	return nil
}

func boolFromByte(b byte, i int) bool {
	return (b>>i)&1 != 0
}

func Uncompress(f InFile) (*Savegame, []byte, uint32, error) {
	s := Savegame{
		Checksum: 0,
	}
	title, err := s.readUncompressed(f, maxTitleLength)
	if err != nil {
		return nil, nil, 0, err
	}
	s.Title = string(title)
	gotTitleChecksum, err := s.readW(f)
	if err != nil {
		return nil, nil, 0, err
	}
	if gotTitleChecksum != titleChecksum(title) {
		return nil, nil, 0, fmt.Errorf("Load: title checksum doesn't match, file had %v, calculated %v", gotTitleChecksum, titleChecksum(title))
	}

	uncompressed, err := s.readCompressed(f, uncompressedSize)
	if err != nil {
		return nil, nil, 0, err
	}

	calculatedChecksum := s.Checksum + fileChecksumAdd
	s.Checksum, err = s.readL(f)
	if err != nil {
		return nil, nil, 0, err
	}
	if s.Checksum != calculatedChecksum {
		fmt.Printf("Load: file checksum doesn't match, read %v, calculated %v\n", s.Checksum, calculatedChecksum)
	}

	return &s, uncompressed, calculatedChecksum, nil
}

// doesn't support all text IDs
func Load(f InFile) (*Savegame, error) {
	s, uncompressed, checksum, err := Uncompress(f)
	if err != nil {
		return nil, err
	}
	// treat uncompressed data as a fake file, so we can reuse same functions
	bf := &bytesFile{data: uncompressed}

	s.Tiles = make([]Tile, NumberOfTiles)

	s.Days, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.FractionalDays, err = s.readW(bf)
	if err != nil {
		return nil, err
	}

	for range 0x1e {
		e := TextEffect{}
		v := reflect.ValueOf(&e).Elem()
		err = s.readStruct(bf, v)
		if err != nil {
			return nil, err
		}
		if e.ID != 0xFFFF {
			s.TextEffects = append(s.TextEffects, e)
		}
	}

	s.Seed, err = s.readLL(bf)
	if err != nil {
		return nil, err
	}

	var townNames []uint16
	for range 0x46 {
		t := Town{}
		t.X, err = s.readB(bf)
		if err != nil {
			return nil, err
		}
		t.Y, err = s.readB(bf)
		if err != nil {
			return nil, err
		}
		t.Population, err = s.readW(bf)
		if err != nil {
			return nil, err
		}
		name, err := s.readW(bf)
		if err != nil {
			return nil, err
		}
		townNames = append(townNames, name)
		_, err = s.readUncompressed(bf, townPlaceholder)
		if err != nil {
			return nil, err
		}
		if t.X != 0 || t.Y != 0 {
			s.Towns = append(s.Towns, t)
		}
	}

	for range 0x1388 {
		c, err := s.readW(bf)
		if err != nil {
			return nil, err
		}
		if c != 0 {
			s.Schedules = append(s.Schedules, c)
		}
	}

	for range 0x100 {
		c, err := s.readW(bf)
		if err != nil {
			return nil, err
		}
		if c != 0 {
			s.Animations = append(s.Animations, c)
		}
	}

	_, err = s.readL(bf)
	if err != nil {
		return nil, err
	}

	for range 0xff {
		d := Depot{}
		v := reflect.ValueOf(&d).Elem()
		err = s.readStruct(bf, v)
		if err != nil {
			return nil, err
		}
		if d.XY != 0 {
			s.Depots = append(s.Depots, d)
		}
	}

	s.NextProcessedTown, err = s.readL(bf)
	if err != nil {
		return nil, err
	}
	s.AnimationTicker, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.LandscapeCode, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.AgeTicker, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.AnotherAnimationTicker, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.NextProcessedXY, err = s.readW(bf)
	if err != nil {
		return nil, err
	}

	// placeholder for costs and cargo
	_, err = s.readUncompressed(bf, placeholder1)
	if err != nil {
		return nil, err
	}

	L1, err := s.readUncompressed(bf, NumberOfTiles)
	if err != nil {
		return nil, err
	}
	L2, err := s.readUncompressed(bf, NumberOfTiles)
	if err != nil {
		return nil, err
	}
	_, err = s.readUncompressed(bf, 2*NumberOfTiles) // L3
	if err != nil {
		return nil, err
	}
	_, err = s.readUncompressed(bf, 0x4000) // desert
	if err != nil {
		return nil, err
	}

	// placeholder for stations, industry
	_, err = s.readUncompressed(bf, placeholder2)
	if err != nil {
		return nil, err
	}

	companyNames := make([]uint16, 8)
	managerNames := make([]uint16, 8)
	for i := range 8 {
		c := Company{}
		companyNames[i], err = s.readW(bf)
		if err != nil {
			return nil, err
		}
		c.NameParts, err = s.readL(bf)
		if err != nil {
			return nil, err
		}
		c.Face, err = s.readL(bf)
		if err != nil {
			return nil, err
		}
		managerNames[i], err = s.readW(bf)
		if err != nil {
			return nil, err
		}
		c.ManagerNameParts, err = s.readL(bf)
		if err != nil {
			return nil, err
		}
		_, err = s.readUncompressed(bf, 0x3b2-16)
		if err != nil {
			return nil, err
		}
		if companyNames[i] != 0 {
			s.Companies = append(s.Companies, c)
		}
	}

	// placeholder vehicles
	_, err = s.readUncompressed(bf, placeholder3)
	if err != nil {
		return nil, err
	}

	for i := range 0x1f4 {
		c, err := s.readUncompressed(bf, 0x20)
		if err != nil {
			return nil, err
		}
		str := string(c)
		idx := uint16(i + firstCustomTextID)
		for j := range s.Towns {
			if townNames[j] == idx {
				s.Towns[i].Name = str
			}
		}
		for j := range s.Companies {
			if companyNames[j] == idx {
				s.Companies[j].Name = str
			}
			if managerNames[j] == idx {
				s.Companies[j].ManagerName = str
			}
		}
	}

	// vehicles in bounding blocks
	_, err = s.readUncompressed(bf, 0x1000*2)
	if err != nil {
		return nil, err
	}

	// placeholder for signs, vehicle types
	_, err = s.readUncompressed(bf, placeholder4)
	if err != nil {
		return nil, err
	}

	s.NextVehicleArray, err = s.readW(bf)
	if err != nil {
		return nil, err
	}

	// placeholder for subsidies
	_, err = s.readUncompressed(bf, 32)
	if err != nil {
		return nil, err
	}

	s.AICompanyTicks, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.MainViewX, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.MainViewY, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.Zoom, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.MaximumLoan, err = s.readL(bf)
	if err != nil {
		return nil, err
	}
	s.MaximumLoanInternal, err = s.readL(bf)
	if err != nil {
		return nil, err
	}
	s.RecessionCounter, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.DaysUntilDisaster, err = s.readW(bf)
	if err != nil {
		return nil, err
	}

	// placeholder for pointers to text IDs
	_, err = s.readUncompressed(bf, placeholder5)
	if err != nil {
		return nil, err
	}

	s.Player1Company, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.Player2Company, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.NextStationTick, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.Currency, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.MeasurementSystem, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.NextCompanyTick, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.Year, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.Month, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	_, err = s.readUncompressed(bf, 8)
	if err != nil {
		return nil, err
	}
	s.Inflation, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.CargoInflation, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.InterestRate, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	b, err := s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.SmallAirpots = boolFromByte(b, 0)
	s.LargeAirpots = boolFromByte(b, 1)
	s.Heliports = boolFromByte(b, 2)
	b, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.DriveOnTheRight = boolFromByte(b, 0)
	s.DriveOnTheRightFixed = boolFromByte(b, 1)
	s.TownNameStyle, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.MaximumCompetitors, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.CompetitorStartTime, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.NumberOfTowns, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.NumberOfIndustries, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.MaxInitialLoan, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.InitialInterestRate, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.VehicleRunningCosts, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.AIConstructionSpeed, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.AIIntelligence, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.Breakdowns, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.SubsidyMultiplier, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.CostsOfConstruction, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.TerrianType, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.QuantityOfLakes, err = s.readW(bf)
	if err != nil {
		return nil, err
	}
	s.FluctuatingEconomy, err = s.readWBool(bf)
	if err != nil {
		return nil, err
	}
	s.TrainReversingEndOfTheLineOnly, err = s.readWBool(bf)
	if err != nil {
		return nil, err
	}
	s.Disasters, err = s.readWBool(bf)
	if err != nil {
		return nil, err
	}
	s.Difficulty, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.LandscapeType, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.TreeTicker, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	b, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	s.CustomVehicleNames = boolFromByte(b, 0)
	s.CustomVehicleNamesCanBeChanged = boolFromByte(b, 1)
	s.SnowLine, err = s.readB(bf)
	if err != nil {
		return nil, err
	}
	_, err = s.readUncompressed(bf, placeholder6)
	if err != nil {
		return nil, err
	}
	L4, err := s.readUncompressed(bf, NumberOfTiles)
	if err != nil {
		return nil, err
	}
	L5, err := s.readUncompressed(bf, NumberOfTiles)
	if err != nil {
		return nil, err
	}

	for i := range NumberOfTiles {
		s.Tiles[i].Class = L4[i] >> 4
		s.Tiles[i].Height = L4[i] & 0x0f
		if s.Tiles[i].Class == 0 { // normal,
			s.Tiles[i].Owner = L1[i]
			s.Tiles[i].Type = L5[i] & 0x0f
		} else if s.Tiles[i].Class == 2 { // road
			s.Tiles[i].Owner = L1[i]
			s.Tiles[i].Type = L5[i] & 0x0f
		} else if s.Tiles[i].Class == 3 { // town building
			s.Tiles[i].Type = L2[i]
		} else if s.Tiles[i].Class == 6 { // water
			s.Tiles[i].Owner = L1[i]
			s.Tiles[i].Type = L5[i]
		} else {
			return nil, fmt.Errorf("Unsupported tile class %x\n", s.Tiles[i].Class)
		}
	}

	if bf.index != len(bf.data) {
		return nil, fmt.Errorf("not all uncompressed bytes used: %d used, %d total", bf.index, len(bf.data))
	}

	s.Checksum = checksum
	return s, nil
}
