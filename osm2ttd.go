package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"osm2ttd/ttd"
	"slices"
	"strconv"
	"strings"

	"github.com/paulmach/osm"
	"github.com/paulmach/osm/osmpbf"
)

var (
	minLat   float64
	maxLat   float64
	minLon   float64
	maxLon   float64
	size     = flag.Float64("size", 0.1, "Size of the map in degrees")
	townTags = flag.String("towns", "village,city", "OpenStreetMaps tags to count as towns")
	roadTags = flag.String("roads", "roads,motorway,trunk,primary,secondary,tertiary,unclassified,residential", "OpenStreetMaps tags to count as roads")
)

func coordToXY(c float64, lat bool) int {
	if lat {
		return int(255 - (c-minLat)/(*size)*256)
	} else {
		return int(255 - (c-minLon)/(*size)*256)
	}
}

func xyToTile(X, Y int) int {
	return Y*256 + X
}

func coordToTile(lat float64, lon float64) int {
	return xyToTile(coordToXY(lat, true), coordToXY(lon, false))
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func road(s *ttd.Savegame, x1, y1, x2, y2 int) {
	d := 0
	xm := 1.0
	ym := 1.0
	pieces := 15
	if x1 != x2 || y1 != y2 {
		if abs(x2-x1) >= abs(y2-y1) {
			d = abs(x2 - x1)
			pieces = 10
			if x1 > x2 {
				xm = -1.0
			}
			ym = float64(y2-y1) / float64(abs(x2-x1))
		} else {
			d = abs(y2 - y1)
			pieces = 5
			if y1 > y2 {
				ym = -1.0
			}
			xm = float64(x2-x1) / float64(abs(y2-y1))
		}
	}
	for i := 0; i <= d; i++ {
		x := x1 + int(float64(i)*xm)
		y := y1 + int(float64(i)*ym)
		s.Tiles[xyToTile(x, y)].Class = 2
		s.Tiles[xyToTile(x, y)].Type = uint8(pieces)
	}
}

func main() {
	flag.Parse()
	if flag.NArg() != 4 {
		panic("Usage: osm2ttd [--size=0.1] INFILE OUTFILE LATITUDE LONGITUDE")
	}
	inFilename := flag.Arg(0)
	outFile := flag.Arg(1)
	lat, err := strconv.ParseFloat(flag.Arg(2), 64)
	if err != nil {
		panic(err)
	}
	lon, err := strconv.ParseFloat(flag.Arg(3), 64)
	if err != nil {
		panic(err)
	}
	minLat = lat - *size/2
	maxLat = lat + *size/2
	minLon = lon - *size/2
	maxLon = lon + *size/2

	s := ttd.Savegame{
		Title:          inFilename,
		MaxInitialLoan: 50000,
		Tiles: slices.Repeat([]ttd.Tile{ttd.Tile{
			Height: 1,
			Owner:  0x10, // no owner
			Type:   0x03, // full grass
		}}, ttd.NumberOfTiles),
	}

	in, err := os.Open(inFilename)
	if err != nil {
		panic(err)
	}
	defer in.Close()

	scanner := osmpbf.New(context.Background(), in, 3)
	scanner.SkipRelations = true
	defer scanner.Close()

	nodes := make(map[osm.NodeID]*osm.Node)
	for scanner.Scan() {
		o := scanner.Object()
		switch o.(type) {
		case *osm.Node:
			n := o.(*osm.Node)
			nodes[n.ID] = n
			if minLat < n.Lat && n.Lat < maxLat && minLon < n.Lon && n.Lon < maxLon {
				isTown := false
				town := ttd.Town{
					X: uint8(coordToXY(n.Lon, false)),
					Y: uint8(coordToXY(n.Lat, true)),
				}
				for _, t := range n.Tags {
					if t.Key == "building" && t.Value == "isolated_dwelling" {
						s.Tiles[coordToTile(n.Lat, n.Lon)].Class = 3
						s.Tiles[coordToTile(n.Lat, n.Lon)].Type = 0x18
					}
					if t.Key == "place" && slices.Contains(strings.Split(*townTags, ","), t.Value) {
						isTown = true
					}
					if t.Key == "name" {
						town.Name = t.Value
					}
				}
				if isTown && len(town.Name) != 0 {
					s.Towns = append(s.Towns, town)
					fmt.Printf("Added town %v\n", town)
				}
			}
		case *osm.Way:
			w := o.(*osm.Way)
			if w.Visible {
				for _, t := range w.Tags {
					if t.Key == "building" {
						for _, wn := range w.Nodes {
							n := nodes[wn.ID]
							if minLat < n.Lat && n.Lat < maxLat && minLon < n.Lon && n.Lon < maxLon {
								s.Tiles[coordToTile(n.Lat, n.Lon)].Class = 3
								s.Tiles[coordToTile(n.Lat, n.Lon)].Type = 0x06
								break
							}
						}
					}
					if t.Key == "highway" && slices.Contains(strings.Split(*roadTags, ","), t.Value) {
						prevValid := false
						var prevX, prevY int
						for _, wn := range w.Nodes {
							n := nodes[wn.ID]
							if minLat < n.Lat && n.Lat < maxLat && minLon < n.Lon && n.Lon < maxLon {
								curX := coordToXY(n.Lon, false)
								curY := coordToXY(n.Lat, true)
								if prevValid {
									road(&s, prevX, prevY, curX, curY)
								}
								prevValid = true
								prevX = curX
								prevY = curY
							} else {
								prevValid = false
							}
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	f, err := os.Create(outFile)
	if err != nil {
		panic(err)
	}
	err = s.Save(f)
	if err != nil {
		panic(err)
	}
}
