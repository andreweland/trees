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

func ServeTile(w http.ResponseWriter, r *http.Request) {
  log.Printf("Tile: %s", r.URL.Path)
  m := tilePattern.FindStringSubmatch(r.URL.Path)
  if m == nil {
    http.Error(w, "Bad tile path", http.StatusInternalServerError)
    return
  }
  coordinates := [...]int{0, 0, 0}
  for i := 0; i < 3; i++ {
    var err error
    coordinates[i], err = strconv.Atoi(m[i+1])
    if err != nil {
      http.Error(w, "Bad tile coordinates", http.StatusInternalServerError)
      return
    }
  }
  // Move cast earlier. Scanf not pattern?
  x, y, z := uint64(coordinates[1]), uint64(coordinates[2]), uint64(coordinates[0])
  region := TileRegion(z, x, y)
  log.Printf("Region: %s", region)
  tile := vector_tile.Tile{
    Layers: []*vector_tile.Tile_Layer{
      &vector_tile.Tile_Layer{
        Name: proto.String("b"),
        Version: proto.Uint32(1),
        Extent: proto.Uint32(4096),
        Features: []*vector_tile.Tile_Feature{
          &vector_tile.Tile_Feature{
            Type: vector_tile.Tile_POINT.Enum(),
            Geometry: []uint32{
              (1 & 0x7) | (1 << 3), // MoveTo: id 1, count 1
              (2048 << 1) ^ (2048 >> 31), // 2048, zigzag encoded
              (2048 << 1) ^ (2048 >> 31),
            },
          },
        },
      },
    },
  }
  data, err := proto.Marshal(&tile)
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

type TreeByCellID []Tree

func (a TreeByCellID) Len() int           { return len(a) }
func (a TreeByCellID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TreeByCellID) Less(i, j int) bool { return a[i].CellID < a[j].CellID }

func LoadTrees(trees []Tree) error {
  err := camden.LoadCSVFromFile(TreesFilename, true, func (row []string) error {
    fmt.Printf("Tree: %s,%s\n", row[camden.TreesLatitudeColumn], row[camden.TreesLongditudeColumn])
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
    return err
  }
  sort.Sort(TreeByCellID(trees))
  return nil
}

func main() {
  trees := []Tree{}
  err := LoadTrees(trees)
  if err != nil {
    log.Fatal(err)
  }
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "html/map.html")
  })
  http.HandleFunc("/output.geojson", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "html/output.geojson")
  })
  http.HandleFunc("/tile/", ServeTile)
  addr := "localhost:9000"
  server := http.Server{Addr: addr}
  log.Printf("Listening on %s", addr)
  log.Fatal(server.ListenAndServe())
}
