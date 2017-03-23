package camden

import (
  "encoding/csv"
  "io"
  "os"
)

const (
  PurchasePostcodeColumn = 3
  PurchasePropertyTypeColumn = 4
  PurchaseNewBuildColumn = 5
  PurchasePAONColumn = 7
  PurchaseSAONColumn = 8
  PurchaseStreetColumn = 9

  CodePointPostcodeColumn = 0
  CodePointEastingsColumn = 2
  CodePointNorthingsColumn = 3
  CodePointDistrictCodeColumn = 8

  PlanningAddressColumn = 2
  PlanningDescriptionColumn = 3

  TreesSpreadColumn = 4
  TreesLongditudeColumn = 14
  TreesLatitudeColumn = 15

  HousingStockPropertyReferenceColumn = 0
  HousingStockBedroomCountColumn = 3
  HousingStockEstateColumn = 7
  HousingStockLongditudeColumn = 11
  HousingStockLatitudeColumn = 12

  HousingBidPropertyReferenceColumn = 1
  HousingBidBedroomCountColumn = 5
  HousingBidMaxPointsColumn = 8

  CamdenDistrictCode = "E09000007"
)

type Address struct {
  PAON string
  SAON string
  Street string
}

func LoadCSVFromFile(filename string, skipHeader bool, f func(row []string) error) error {
  file, err := os.Open(filename)
  if err != nil {
    return err
  }
  return LoadCSV(file, skipHeader, f)
}

func LoadCSV(input io.Reader, skipHeader bool, f func(row []string) error) error {
  r := csv.NewReader(input)
  if skipHeader {
    r.Read()
  }
  for {
    row, err := r.Read()
    if err == io.EOF {
      return nil
    } else if err != nil {
      return err
    }
    err = f(row)
    if err != nil {
      return err
    }
  }
}
