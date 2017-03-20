package main

import (
  "camden"
  "encoding/csv"
  "fmt"
  "io"
  "path/filepath"
  "log"
  "os"
  "strconv"
  "strings"
  "github.com/paulmach/go.geo"
  "github.com/paulmach/go.geojson"
)

const (
  PurchaseFilename = "data/purchases/pp-complete-camden.csv"
  CodePointPattern = "data/codepoint-open/Data/CSV/*.csv"
  PlanningFilename = "data/planning/Planning_Applications.csv"
)

type Postcodes map[string]geo.Point

func LoadPostcodes(district string) (Postcodes, error) {
  postcodes := Postcodes{}
  matches, err := filepath.Glob(CodePointPattern)
  if err != nil {
    return nil, err
  }
  t, err := camden.NewTransformer(27700, 4326)
  if err != nil {
    return nil, err
  }
  for _, filename := range matches {
    file, err := os.Open(filename)
    if err != nil {
      return nil, err
    }
    r := csv.NewReader(file)
    for {
      row, err := r.Read()
      if err == io.EOF {
        break
      } else if err != nil {
        return nil, err
      }
      if row[camden.CodePointDistrictCodeColumn] == district {
        eastings, err := strconv.Atoi(row[camden.CodePointEastingsColumn])
        if err != nil {
          return nil, err
        }
        northings, err := strconv.Atoi(row[camden.CodePointNorthingsColumn])
        if err != nil {
          return nil, err
        }
        lng, lat := t.Transform(float64(eastings), float64(northings))
        point := geo.Point{lng, lat}
        postcodes[strings.Replace(row[camden.CodePointPostcodeColumn], " ", "", -1)] = point
      }
    }
  }
  return postcodes, nil
}

func FindPostcode(address string, postcodes Postcodes) (string, bool) {
  parts := strings.Split(address, " ")
  for i := range parts {
    if i + 1 < len(parts) {
      _, ok := postcodes[parts[i] + parts[i+1]]
      if ok {
        return parts[i] + parts[i+1], true
      }
    }
    _, ok := postcodes[parts[i]]
    if ok {
      return parts[i], true
    }
  }
  return "", false
}

func LoadPlanning(postcodes Postcodes) (map[string][]string, error) {
  file, err := os.Open(PlanningFilename)
  if err != nil {
    return nil, err
  }
  planning := map[string][]string{}
  r := csv.NewReader(file)
  for {
    row, err := r.Read()
    if err == io.EOF {
      break
    } else if err != nil {
      return nil, err
    }
    postcode, ok := FindPostcode(row[camden.PlanningAddressColumn], postcodes)
    if !ok {
      continue
    }
    descriptions, ok := planning[postcode]
    if !ok {
      descriptions = make([]string, 0, 1)
    }
    planning[postcode] = append(descriptions, row[camden.PlanningDescriptionColumn])
  }
  return planning, nil
}

type Address struct {
  PAON string
  SAON string
  Street string
}

type State struct {
  PropertyType string
}

type Counts struct {
  Total int
  NewBuild int
}

func LoadPurchases(planning map[string][]string) (map[string]Counts, error) {
  file, err := os.Open(PurchaseFilename)
  if err != nil {
    return nil, err
  }
  properties := map[Address]State{}
  r := csv.NewReader(file)
  purchases := 0
  propertiesByPostcode := map[string]Counts{}
  for {
    row, err := r.Read()
    if err == io.EOF {
      break
    } else if err != nil {
      return nil, err
    }
    purchases++
    postcode := strings.Replace(row[camden.PurchasePostcodeColumn], " ", "", -1)
    address := Address{row[camden.PurchasePAONColumn], row[camden.PurchaseSAONColumn], row[camden.PurchaseStreetColumn]}
    newState := State{row[camden.PurchasePropertyTypeColumn]}
    oldState, ok := properties[address]
    changed := false
    if ok {
      fmt.Printf("%s: Old: %v New: %v\n", postcode, oldState, newState)
      changed = true
    } else {
      current := propertiesByPostcode[postcode]
      current.Total++
      propertiesByPostcode[postcode] = current
      oldState, ok = properties[Address{PAON: address.PAON, Street: address.Street}]
      if ok {
        fmt.Printf("%s: Old: %v New: %v (converted?)\n", postcode, oldState, newState)
        changed = true
      }
    }
    if row[camden.PurchaseNewBuildColumn] == "Y" {
      current := propertiesByPostcode[postcode]
      current.NewBuild++
      propertiesByPostcode[postcode] = current
      fmt.Printf("%s: New build %v\n", postcode, newState)
      changed = true
    }
    if changed {
      descriptions, ok := planning[postcode]
      if ok {
        fmt.Printf("- %d applications\n", len(descriptions))
      }
    }
    properties[address] = newState
  }
  fmt.Printf("%d properties, %d purchases\n", len(properties), purchases)
  return propertiesByPostcode, nil
}

func main() {
  log.Print("Loading Camden postcodes")
  postcodes, err := LoadPostcodes(camden.CamdenDistrictCode)
  if err != nil {
    log.Fatal(err)
  }
  log.Print("Loading planning")
  planning, err := LoadPlanning(postcodes)
  if err != nil {
    log.Fatal(err)
  }
  log.Print("Loading purchases")
  propertiesByPostcode, err := LoadPurchases(planning)
  if err != nil {
    log.Fatal(err)
  }
  collection := geojson.NewFeatureCollection()
  for postcode, counts := range propertiesByPostcode {
    if !strings.HasPrefix(postcode, "NW5") {
      continue
    }
    point, ok := postcodes[postcode]
    if ok {
      feature := point.ToGeoJSON()
      feature.SetProperty("total", counts.Total)
      feature.SetProperty("newBuild", counts.NewBuild)
      collection.AddFeature(feature)
    }
  }
  json, err := collection.MarshalJSON()
  if err != nil {
    log.Fatal(err)
  }
  out, err := os.OpenFile("html/output.geojson", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0777)
  if err != nil {
    log.Fatal(err)
  }
  out.Write(json)
  out.Close()
}
