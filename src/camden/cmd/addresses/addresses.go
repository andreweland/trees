package main

import (
  "camden"
  "fmt"
  "os"
  "log"
  "regexp"
  "strings"
  "sort"
)

const (
  PurchaseFilename = "data/purchases/pp-complete-camden.csv"
  PlanningFilename = "data/planning/Planning_Applications.csv"
)

type ByLength []camden.Address

func (a ByLength) Len() int           { return len(a) }
func (a ByLength) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLength) Less(i, j int) bool { return len(a[i].Street) > len(a[j].Street) }

var (
  singleHouseNumber = regexp.MustCompile("\\d+[A-G]?")
  houseNumberRange = regexp.MustCompile("\\d+[A-G]?-\\d+[A-G]")
)

func ParseAddress(address string, streets map[string]bool) (camden.Address, bool) {
  matches := []camden.Address{}
  // TODO: remove empty strings
  parts := strings.Split(address, " ")
  for start := range parts {
    if !singleHouseNumber.MatchString(parts[start]) && !houseNumberRange.MatchString(parts[start]) {
      continue
    }
    for end := start + 2; end < len(parts) + 1; end++ {
      search := strings.ToUpper(strings.Join(parts[start + 1:end], " "))
      if _, ok := streets[search]; ok {
        matches = append(matches, camden.Address{PAON: parts[start], Street: search})
      }
    }
  }
  sort.Sort(ByLength(matches))
  if len(matches) > 0 {
    return matches[0], true
  }
  return camden.Address{}, false
}

func main() {
  file, err := os.Open(PurchaseFilename)
  if err != nil {
    log.Fatal(err)
  }
  streets := map[string]bool{}
  err = camden.LoadCSV(file, func(row []string) error {
    streets[row[camden.PurchaseStreetColumn]] = true
    return nil
  })
  if err != nil {
    log.Fatal(err)
  }
  log.Printf("Streets: %d", len(streets))

  file, err = os.Open(PlanningFilename)
  if err != nil {
    log.Fatal(err)
  }
  err = camden.LoadCSV(file, func(row []string) error {
    address := row[camden.PlanningAddressColumn]
    parsed, ok := ParseAddress(address, streets)
    if false && ok {
      fmt.Printf("%s: %q %q\n", address, parsed.PAON, parsed.Street)
    }
    if !ok {
      fmt.Printf("Failed: %s\n", address)
    }
    return nil
  })
  if err != nil {
    log.Fatal(err)
  }
}
