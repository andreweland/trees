package trees

import (
	"sort"

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
