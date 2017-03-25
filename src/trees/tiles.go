package trees

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"trees/proto" // Imports as vector_tile

	"github.com/golang/geo/s2"
	"github.com/golang/protobuf/proto"
	"github.com/paulmach/go.geo"
)

const tileExtent = 12 // Implies 1 << 12, ie 4096, units per tile
const tileNameTag = 0
const tileSpreadTag = 1
const tileHeightTag = 2
const tileRadiusTag = 3

var tileKeys []string

func init() {
	tileKeys = make([]string, 4, 4)
	tileKeys[tileNameTag] = "name"
	tileKeys[tileSpreadTag] = "spread"
	tileKeys[tileHeightTag] = "height"
	tileKeys[tileRadiusTag] = "radius"
}

type Painter struct {
	Tile           *vector_tile.Tile
	Layer          *vector_tile.Tile_Layer
	X, Y, Z        uint64
	MetersPerPixel float64

	Names map[string]uint32
}

func (p *Painter) Init(x, y, z uint64) {
	p.X, p.Y, p.Z = x, y, z
	p.Names = map[string]uint32{}
	p.Layer = &vector_tile.Tile_Layer{
		Name:     proto.String("trees"),
		Keys:     tileKeys,
		Version:  proto.Uint32(1),
		Extent:   proto.Uint32(1 << tileExtent),
		Features: []*vector_tile.Tile_Feature{},
	}
	p.Tile = &vector_tile.Tile{
		Layers: []*vector_tile.Tile_Layer{
			p.Layer,
		},
	}
	bound := geo.NewBoundFromMapTile(x, y, z)
	p.MetersPerPixel = bound.GeoHeight() / 256.0
}

func (p *Painter) Paint(t *Tree) {
	feature := p.point(t.Location.LatLng())
	p.addProperty(feature, tileSpreadTag, t.Spread)
	p.addProperty(feature, tileHeightTag, t.Height)
	p.addProperty(feature, tileRadiusTag, float32(math.Ceil(float64(t.Spread)/p.MetersPerPixel)))
	nameID, ok := p.Names[t.Name]
	if !ok {
		p.Layer.Values = append(p.Layer.Values, &vector_tile.Tile_Value{StringValue: proto.String(t.Name)})
		nameID = uint32(len(p.Layer.Values) - 1)
		p.Names[t.Name] = nameID
	}
	feature.Tags = append(feature.Tags, tileNameTag, nameID)
	p.Layer.Features = append(p.Layer.Features, feature)
}

func (p *Painter) addProperty(feature *vector_tile.Tile_Feature, tagID uint32, value interface{}) {
	p.Layer.Values = append(p.Layer.Values, tileValue(value))
	feature.Tags = append(feature.Tags, tagID, uint32(len(p.Layer.Values)-1))
}

func tileValue(value interface{}) *vector_tile.Tile_Value {
	switch value.(type) {
	case float32:
		return &vector_tile.Tile_Value{FloatValue: proto.Float32(value.(float32))}
	case string:
		return &vector_tile.Tile_Value{StringValue: proto.String(value.(string))}
	default:
		panic("Can't encode tile value")
	}
}

func (p *Painter) point(ll s2.LatLng) *vector_tile.Tile_Feature {
	x, y := p.project(ll)
	return &vector_tile.Tile_Feature{
		Type: vector_tile.Tile_POINT.Enum(),
		Geometry: []uint32{
			(1 & 0x7) | (1 << 3),         // MoveTo: id 1, count 1
			uint32(x<<1) ^ uint32(x>>31), // zigzag encoded
			uint32(y<<1) ^ uint32(y>>31),
		},
	}
}

func (p *Painter) project(ll s2.LatLng) (int32, int32) {
	x, y := geo.ScalarMercator.Project(ll.Lng.Degrees(), ll.Lat.Degrees(), p.Z+tileExtent)
	return int32(x - (p.X << tileExtent)), int32(y - (p.Y << tileExtent))
}

func TileRegion(x, y, z uint64) s2.Region {
	// Add some padding to prevent features clipping at tile boundaries
	bound := geo.NewBoundFromMapTile(x, y, z).GeoPad(100.0)
	nw := bound.NorthWest()
	se := bound.SouthEast()
	return s2.RectFromLatLng(s2.LatLngFromDegrees(nw[1], nw[0])).AddPoint(s2.LatLngFromDegrees(se[1], se[0]))
}

type TileHandler struct {
	Trees []Tree
}

func (h *TileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	painter := &Painter{}
	painter.Init(x, y, z)
	trees := FindTrees(h.Trees, region)
	for _, tree := range trees {
		painter.Paint(&tree)
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
