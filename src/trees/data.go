package trees

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"

	"github.com/golang/geo/s2"
)

type Tree struct {
	Location s2.CellID
	Name     string
	Spread   float32 // Meters
	Height   float32 // Meters
}

type ByLocation []Tree

func (a ByLocation) Len() int           { return len(a) }
func (a ByLocation) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLocation) Less(i, j int) bool { return a[i].Location < a[j].Location }

func FindTrees(trees []Tree, region s2.Region) []Tree {
	found := []Tree{}
	coverer := &s2.RegionCoverer{MaxLevel: 30, MaxCells: 5}
	covering := coverer.Covering(region)
	for _, id := range covering {
		begin := sort.Search(len(trees), func(i int) bool { return trees[i].Location >= id.ChildBegin() })
		for i := begin; i < len(trees) && trees[i].Location < id.ChildEnd(); i++ {
			if region.ContainsCell(s2.CellFromCellID(trees[i].Location)) {
				found = append(found, trees[i])
			}
		}
	}
	return found
}

func LoadCamdenTrees(filename string) ([]Tree, error) {
	trees := []Tree{}
	var jsonTrees []struct {
		Lat    string `json:"latitude"`
		Lng    string `json:"longitude"`
		Name   string `json:"common_name"`
		Height string `json:"height_in_metres"`
		Spread string `json:"spread_in_metres"`
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(file).Decode(&jsonTrees)
	if err != nil {
		return nil, err
	}
	for _, t := range jsonTrees {
		lat, err := strconv.ParseFloat(t.Lat, 64)
		if err != nil {
			continue
		}
		lng, err := strconv.ParseFloat(t.Lng, 64)
		if err != nil {
			continue
		}
		height, err := strconv.ParseFloat(t.Height, 32)
		if err != nil {
			continue
		}
		spread, err := strconv.ParseFloat(t.Spread, 32)
		if err != nil {
			continue
		}
		trees = append(trees, Tree{
			Location: s2.CellIDFromLatLng(s2.LatLngFromDegrees(lat, lng)),
			Name:     t.Name,
			Height:   float32(height),
			Spread:   float32(spread),
		})
	}
	return trees, nil
}
