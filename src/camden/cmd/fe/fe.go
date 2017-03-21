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
}

type Painter struct {
  Tile *vector_tile.Tile
  X, Y, Z uint64
}

const TileExtent = 12 // Implies 1 << 2, ie 4096, units per tile

func (p *Painter) Init(x, y, z uint64) {
  p.X, p.Y, p.Z = x, y, z
  p.Tile = &vector_tile.Tile{
    Layers: []*vector_tile.Tile_Layer{
      &vector_tile.Tile_Layer{
        Name: proto.String("b"),
        Version: proto.Uint32(1),
        Extent: proto.Uint32(1 << TileExtent),
        Features: []*vector_tile.Tile_Feature{},
      },
    },
  }
}

func (p *Painter) project(ll s2.LatLng) (uint32, uint32) {
  x, y := geo.ScalarMercator.Project(ll.Lng.Degrees(), ll.Lat.Degrees(), p.Z + TileExtent)
  return uint32(x - (p.X << TileExtent)), uint32(y - (p.Y << TileExtent))
}

func (p *Painter) AddPoint(ll s2.LatLng) {
  x, y := p.project(ll)
  log.Printf("Tree: %s -> %d,%d", ll, x, y)
  feature := &vector_tile.Tile_Feature{
    Type: vector_tile.Tile_POINT.Enum(),
    Geometry: []uint32{
      (1 & 0x7) | (1 << 3), // MoveTo: id 1, count 1
      (x << 1) ^ (x >> 31), // zigzag encoded
      (y << 1) ^ (y >> 31),
    },
  }
  p.Tile.Layers[0].Features = append(p.Tile.Layers[0].Features, feature)
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
  trees := FindTrees(h.Trees, region)
  painter := &Painter{}
  painter.Init(x, y, z)
  for _, tree := range trees {
    painter.AddPoint(tree.CellID.LatLng())
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

type TreeByCellID []Tree

func (a TreeByCellID) Len() int           { return len(a) }
func (a TreeByCellID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TreeByCellID) Less(i, j int) bool { return a[i].CellID < a[j].CellID }

func main() {
  trees, err := LoadTrees()
  if err != nil {
    log.Fatal(err)
  }
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "html/map.html")
  })
  http.HandleFunc("/output.geojson", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "html/output.geojson")
  })
  tileHandler := &TileHandler{Trees: trees}
  http.Handle("/tile/", tileHandler)
  addr := "localhost:9000"
  server := http.Server{Addr: addr}
  log.Printf("Listening on %s", addr)
  log.Fatal(server.ListenAndServe())
}
