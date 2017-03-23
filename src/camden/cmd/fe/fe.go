package main

import (
  "camden"
  // Fix the import
  vector_tile "camden/proto"
  "fmt"
  "regexp"
  "log"
  "net/http"
  "sort"
  "strconv"
  "github.com/golang/protobuf/proto"
  "github.com/golang/geo/s2"
  "github.com/paulmach/go.geo"
)

const (
  TreesFilename = "data/trees/Trees_In_Camden.csv"
  HousingStockFilename = "data/council-housing/Camden_residential_housing_stock_excluding_leasehold_properties.csv"
)

func TileRegion(x, y, z uint64) s2.Region {
  // Replace with all S2?
  // Unconvinced this is the right call
  bound := geo.NewBoundFromMapTile(x, y, z)
  nw := bound.NorthWest()
  se := bound.SouthEast()
  return s2.RectFromLatLng(s2.LatLngFromDegrees(nw[1], nw[0])).AddPoint(s2.LatLngFromDegrees(se[1], se[0]))
}

var tilePattern = regexp.MustCompile(`/tile/(\d+)/(\d+)/(\d+)\.mvt`)

type TileHandler struct {
  Trees []Tree
  Estates []Estate
}

type Painter struct {
  Tile *vector_tile.Tile
  Trees *vector_tile.Tile_Layer
  Estates *vector_tile.Tile_Layer
  X, Y, Z uint64
}

const TileExtent = 12 // Implies 1 << 2, ie 4096, units per tile

func (p *Painter) Init(x, y, z uint64) {
  p.X, p.Y, p.Z = x, y, z
  p.Trees = &vector_tile.Tile_Layer{
    Name: proto.String("trees"),
    Keys: []string{"spread"},
    Version: proto.Uint32(1),
    Extent: proto.Uint32(1 << TileExtent),
    Features: []*vector_tile.Tile_Feature{},
  }
  p.Estates = &vector_tile.Tile_Layer{
    Name: proto.String("estates"),
    Version: proto.Uint32(1),
    Extent: proto.Uint32(1 << TileExtent),
    Features: []*vector_tile.Tile_Feature{},
  }
  p.Tile = &vector_tile.Tile{
    Layers: []*vector_tile.Tile_Layer{
      p.Trees,
      p.Estates,
    },
  }
}

func (p *Painter) AddTree(t *Tree) {
  // Could optimise repeat values
  p.Trees.Values = append(p.Tile.Layers[0].Values, &vector_tile.Tile_Value{FloatValue: proto.Float32(t.Spread)})
  feature := p.point(t.CellID.LatLng())
  feature.Tags = []uint32{0, uint32(len(p.Trees.Values) - 1)}
  p.Trees.Features = append(p.Trees.Features, feature)
}

func (p *Painter) AddEstate(e *Estate) {
  feature := p.point(e.CellID.LatLng())
  p.Estates.Features = append(p.Estates.Features, feature)
}

func (p *Painter) point(ll s2.LatLng) *vector_tile.Tile_Feature {
  x, y := p.project(ll)
  return &vector_tile.Tile_Feature{
      Type: vector_tile.Tile_POINT.Enum(),
      Geometry: []uint32{
        (1 & 0x7) | (1 << 3), // MoveTo: id 1, count 1
        (x << 1) ^ (x >> 31), // zigzag encoded
        (y << 1) ^ (y >> 31),
      },
    }
}

func (p *Painter) project(ll s2.LatLng) (uint32, uint32) {
  x, y := geo.ScalarMercator.Project(ll.Lng.Degrees(), ll.Lat.Degrees(), p.Z + TileExtent)
  return uint32(x - (p.X << TileExtent)), uint32(y - (p.Y << TileExtent))
}

func (h *TileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  log.Printf("Tile: %s", r.URL.Path)
  var x, y, z uint64
  n, err := fmt.Sscanf(r.URL.Path, "/tile/%d/%d/%d.mvt", &z, &x, &y)
  if err != nil {
    log.Fatal(err)
  }
  if n != 3 {
    http.Error(w, "Bad tile path", http.StatusInternalServerError)
    return
  }
  region := TileRegion(x, y, z)
  log.Printf("Region: %s", region)
  painter := &Painter{}
  painter.Init(x, y, z)
  trees := FindTrees(h.Trees, region)
  for _, tree := range trees {
    painter.AddTree(&tree)
  }
  estates := FindEstates(h.Estates, region)
  for _, estate := range estates {
    painter.AddEstate(&estate)
  }
  data, err := proto.Marshal(painter.Tile)
  if err != nil {
    log.Printf("Error: %v", err)
    http.Error(w, "Bad tile encoding", http.StatusInternalServerError)
    return
  }
  w.Header().Set("Content-Type", "application/x-protobuf")
  w.Write(data)
}

type Tree struct {
  CellID s2.CellID
  Spread float32 // Meters
}

func LoadTrees() ([]Tree, error) {
  trees := []Tree{}
  err := camden.LoadCSVFromFile(TreesFilename, true, func (row []string) error {
    lat, err := strconv.ParseFloat(row[camden.TreesLatitudeColumn], 64)
    if err != nil {
      return nil
    }
    lng, err := strconv.ParseFloat(row[camden.TreesLongditudeColumn], 64)
    if err != nil {
      return nil
    }
    spread, err := strconv.ParseFloat(row[camden.TreesSpreadColumn], 32)
    if err != nil {
      return nil
    }
    trees = append(trees, Tree{s2.CellIDFromLatLng(s2.LatLngFromDegrees(lat, lng)), float32(spread)})
    return nil
  })
  if err != nil {
    return nil, err
  }
  sort.Sort(TreeByCellID(trees))
  return trees, nil
}

type TreeByCellID []Tree

func (a TreeByCellID) Len() int           { return len(a) }
func (a TreeByCellID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TreeByCellID) Less(i, j int) bool { return a[i].CellID < a[j].CellID }

func FindTrees(trees []Tree, region s2.Region) []Tree {
  found := []Tree{}
  coverer := &s2.RegionCoverer{MaxLevel: 30, MaxCells: 5}
  covering := coverer.Covering(region)
  for _, id := range covering {
    begin := sort.Search(len(trees), func(i int) bool {return trees[i].CellID >= id.ChildBegin()})
    for i := begin; i < len(trees) && trees[i].CellID < id.ChildEnd(); i++ {
      if region.ContainsCell(s2.CellFromCellID(trees[i].CellID)) {
        found = append(found, trees[i])
      }
    }
  }
  return found
}

type Estate struct {
  Name string
  CellID s2.CellID
  BedroomCounts []int
}

const MaxBedrooms = 10

func LoadHousing() ([]Estate, error) {
  estatesByName := map[string]Estate{}
  err := camden.LoadCSVFromFile(HousingStockFilename, true, func (row []string) error {
    estateName := row[camden.HousingStockEstateColumn]
    if estateName == "" {
      return nil
    }
    lat, err := strconv.ParseFloat(row[camden.HousingStockLatitudeColumn], 64)
    if err != nil {
      return nil
    }
    lng, err := strconv.ParseFloat(row[camden.HousingStockLongditudeColumn], 64)
    if err != nil {
      return nil
    }
    bedrooms, err := strconv.Atoi(row[camden.HousingStockBedroomCountColumn])
    if err != nil {
      return nil
    }
    estate, ok := estatesByName[estateName]
    if !ok {
      estate = Estate{
        Name: estateName,
        CellID: s2.CellIDFromLatLng(s2.LatLngFromDegrees(lat, lng)),
        BedroomCounts: make([]int, MaxBedrooms, MaxBedrooms),
      }
      estatesByName[estateName] = estate
    }
    if bedrooms < len(estate.BedroomCounts) {
      estate.BedroomCounts[bedrooms]++
    }
    return nil
  })
  if err != nil {
    return nil, err
  }
  estates := make([]Estate, 0, len(estatesByName))
  for _, estate := range estatesByName {
    estates = append(estates, estate)
  }
  sort.Sort(EstateByCellID(estates))
  return estates, nil
}

// TODO:
// To factor this out we'd have to make the Load... methods return an interface
// Or can we implement an operation interface like Sort()/Find()
// By making a searchable interaface on top of []Estate and []Tree?
type EstateByCellID []Estate

func (a EstateByCellID) Len() int           { return len(a) }
func (a EstateByCellID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EstateByCellID) Less(i, j int) bool { return a[i].CellID < a[j].CellID }

func FindEstates(estates []Estate, region s2.Region) []Estate {
  found := []Estate{}
  coverer := &s2.RegionCoverer{MaxLevel: 30, MaxCells: 5}
  covering := coverer.Covering(region)
  for _, id := range covering {
    begin := sort.Search(len(estates), func(i int) bool {return estates[i].CellID >= id.ChildBegin()})
    for i := begin; i < len(estates) && estates[i].CellID < id.ChildEnd(); i++ {
      if region.ContainsCell(s2.CellFromCellID(estates[i].CellID)) {
        found = append(found, estates[i])
      }
    }
  }
  return found
}

func main() {
  trees, err := LoadTrees()
  if err != nil {
    log.Fatal(err)
  }
  estates, err := LoadHousing()
  if err != nil {
    log.Fatal(err)
  }
  log.Printf("%d trees, %d estates", len(trees), len(estates))
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "html/map.html")
  })
  http.HandleFunc("/output.geojson", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "html/output.geojson")
  })
  tileHandler := &TileHandler{Trees: trees, Estates: estates}
  http.Handle("/tile/", tileHandler)
  addr := "localhost:9000"
  server := http.Server{Addr: addr}
  log.Printf("Listening on %s", addr)
  log.Fatal(server.ListenAndServe())
}
