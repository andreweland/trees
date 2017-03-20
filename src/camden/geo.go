package camden

import (
  "github.com/lukeroth/gdal"
)

type Transformer struct {
  from  gdal.SpatialReference
  to    gdal.SpatialReference
  point gdal.Geometry
}

func NewTransformer(fromEPSG int, toEPSG int) (*Transformer, error) {
  t := &Transformer{gdal.CreateSpatialReference(""), gdal.CreateSpatialReference(""), gdal.Create(gdal.GT_Point)}
  err := t.from.FromEPSG(fromEPSG)
  if err != nil {
    return nil, err
  }
  err = t.to.FromEPSG(toEPSG)
  if err != nil {
   return t, err
  }
  return t, nil
}

func (t *Transformer) Transform(x float64, y float64) (float64, float64) {
  t.point.SetSpatialReference(t.from)
  t.point.SetPoint2D(0, x, y)
  t.point.TransformTo(t.to)
  return t.point.X(0), t.point.Y(0)
}
