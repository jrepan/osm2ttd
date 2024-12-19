package ttd

type TextEffect struct {
	ID                       uint16 // 0xFFFF if empty
	Left, Right, Top, Bottom uint16
	Expiration               uint16
	Data                     uint32
	Unused                   uint32
}

type Depot struct {
	XY   uint16
	Town uint32
}

type Town struct {
	X, Y       uint8 // 00 for empty slot
	Population uint16
	Name       string
}

type Company struct {
	Name             string
	NameParts        uint32
	Face             uint32
	ManagerName      string
	ManagerNameParts uint32
}

type Tile struct {
	Class  uint8
	Type   uint8
	Owner  uint8
	Height uint8
}

type Savegame struct {
	Checksum                                           uint32 // Do not set, this is calculated automatically
	Title                                              string
	Days, FractionalDays                               uint16
	TextEffects                                        []TextEffect
	Seed                                               uint64
	Towns                                              []Town
	Schedules                                          []uint16
	Animations                                         []uint16
	Depots                                             []Depot
	NextProcessedTown                                  uint32
	AnimationTicker                                    uint16
	LandscapeCode                                      uint16
	AgeTicker                                          uint16
	AnotherAnimationTicker                             uint16
	NextProcessedXY                                    uint16
	Companies                                          []Company
	NextVehicleArray                                   uint16
	AICompanyTicks                                     uint16
	MainViewX, MainViewY                               uint16
	Zoom                                               uint16 // 0 = normal, 1 = intermediate, 2 = most zoomed out
	MaximumLoan, MaximumLoanInternal                   uint32 // multiple of 50000
	RecessionCounter                                   uint16
	DaysUntilDisaster                                  uint16
	Player1Company, Player2Company                     uint8
	NextStationTick                                    uint8
	Currency                                           uint8
	MeasurementSystem                                  uint8
	NextCompanyTick                                    uint8
	Year, Month                                        uint8
	Inflation, CargoInflation                          uint8
	InterestRate                                       uint8
	SmallAirpots, LargeAirpots, Heliports              bool
	DriveOnTheRight, DriveOnTheRightFixed              bool
	TownNameStyle                                      uint8
	MaximumCompetitors                                 uint16
	CompetitorStartTime                                uint16
	NumberOfTowns, NumberOfIndustries                  uint16 // 0 = low, 1 = medium, 2 = high (relevant only at the game start)
	MaxInitialLoan                                     uint16
	InitialInterestRate                                uint16
	VehicleRunningCosts                                uint16
	AIConstructionSpeed, AIIntelligence                uint16
	Breakdowns                                         uint16
	SubsidyMultiplier                                  uint16
	CostsOfConstruction                                uint16
	TerrianType                                        uint16
	QuantityOfLakes                                    uint16
	FluctuatingEconomy                                 bool
	TrainReversingEndOfTheLineOnly                     bool
	Disasters                                          bool
	Difficulty                                         uint8
	LandscapeType                                      uint8
	TreeTicker                                         uint8
	CustomVehicleNames, CustomVehicleNamesCanBeChanged bool
	SnowLine                                           uint8
	Tiles                                              []Tile
}

type InFile interface {
	Read(b []byte) (int, error)
}

type OutFile interface {
	Write(b []byte) (int, error)
}
