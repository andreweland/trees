package trees

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/golang/geo/s2"
)

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
