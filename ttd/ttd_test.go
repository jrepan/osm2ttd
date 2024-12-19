package ttd

import (
	"github.com/google/go-cmp/cmp"
	"slices"
	"strings"
	"testing"
)

type fakeOutFile struct {
	written []byte
}

func (f *fakeOutFile) Write(b []byte) (int, error) {
	f.written = append(f.written, b...)
	return len(b), nil
}

func pads(s string, l int) string {
	return s + strings.Repeat(" ", l-len(s))
}

func TestSaveAndLoad(t *testing.T) {
	want := &Savegame{
		Title:          pads("test", maxTitleLength),
		Days:           42,
		FractionalDays: 3,
		TextEffects: []TextEffect{
			TextEffect{ID: 0x801, Left: 1, Right: 2, Top: 3, Bottom: 4, Expiration: 5, Data: 123},
			TextEffect{ID: 0x803, Left: 5, Right: 6, Top: 7, Bottom: 8, Expiration: 9, Data: 456},
		},
		Seed: 2 ^ 40,
		Towns: []Town{
			Town{X: 54, Y: 55, Population: 56, Name: pads("Town1", 0x20)},
			Town{X: 57, Y: 58, Population: 59, Name: pads("Town2", 0x20)},
		},
		Schedules:  []uint16{60, 61},
		Animations: []uint16{62, 63, 64},
		Depots: []Depot{
			Depot{XY: 10, Town: 11},
			Depot{XY: 12, Town: 13},
		},
		NextProcessedTown:              14,
		AnimationTicker:                15,
		LandscapeCode:                  16,
		AgeTicker:                      17,
		AnotherAnimationTicker:         18,
		NextProcessedXY:                19,
		Companies:                      []Company{Company{Name: pads("Company", 0x20), NameParts: 65, Face: 66, ManagerName: pads("Manager", 0x20), ManagerNameParts: 67}},
		NextVehicleArray:               20,
		AICompanyTicks:                 21,
		MainViewX:                      22,
		MainViewY:                      23,
		Zoom:                           1,
		MaximumLoan:                    50000,
		MaximumLoanInternal:            100000,
		RecessionCounter:               24,
		DaysUntilDisaster:              25,
		Player1Company:                 26,
		Player2Company:                 27,
		NextStationTick:                28,
		Currency:                       29,
		MeasurementSystem:              30,
		NextCompanyTick:                31,
		Year:                           32,
		Month:                          33,
		Inflation:                      34,
		CargoInflation:                 35,
		InterestRate:                   36,
		SmallAirpots:                   true,
		LargeAirpots:                   false,
		Heliports:                      true,
		DriveOnTheRight:                false,
		DriveOnTheRightFixed:           true,
		TownNameStyle:                  37,
		MaximumCompetitors:             38,
		CompetitorStartTime:            39,
		NumberOfTowns:                  1,
		NumberOfIndustries:             2,
		MaxInitialLoan:                 40,
		InitialInterestRate:            41,
		VehicleRunningCosts:            42,
		AIConstructionSpeed:            43,
		AIIntelligence:                 44,
		Breakdowns:                     45,
		SubsidyMultiplier:              46,
		CostsOfConstruction:            47,
		TerrianType:                    48,
		QuantityOfLakes:                49,
		FluctuatingEconomy:             false,
		TrainReversingEndOfTheLineOnly: true,
		Disasters:                      true,
		Difficulty:                     50,
		LandscapeType:                  51,
		TreeTicker:                     52,
		CustomVehicleNames:             true,
		CustomVehicleNamesCanBeChanged: false,
		SnowLine:                       53,
		Tiles:                          slices.Repeat([]Tile{Tile{Class: 0, Height: 1, Owner: 2, Type: 3}}, 0x10000),
	}

	out := &fakeOutFile{}
	err := want.Save(out)
	if err != nil {
		t.Fatal(err)
	}

	in := &bytesFile{data: out.written}
	got, err := Load(in)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Errorf("Diff: %v", cmp.Diff(want, got))
	}
}

func TestReadCompressed(t *testing.T) {
	in := &bytesFile{data: []byte{0xFD, 42, 1, 3, 4}}
	want := []byte{42, 42, 42, 42, 3, 4}
	s := Savegame{}
	got, err := s.readCompressed(in, len(want))
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Errorf("Got %v, wanted %v", got, want)
	}
}
