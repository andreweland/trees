package main

import (
  "encoding/csv"
  "flag"
  "io"
  "log"
  "os"
)

const (
  DistictColumn = 12
)

func main() {
  in := flag.String("in", "data/purchases/pp-complete.csv", "Input file")
  district := flag.String("district", "CAMDEN", "District to keep")
  flag.Parse()

  file, err := os.Open(*in)
  if err != nil {
    log.Fatal(err)
  }
  r := csv.NewReader(file)
  w := csv.NewWriter(os.Stdout)
  for {
    row, err := r.Read()
    if err == io.EOF {
      break
    } else if err != nil {
      log.Fatal(err)
    }
    if row[DistictColumn] == *district {
      err = w.Write(row)
      if err != nil {
        log.Fatal(err)
      }
    }
  }
  w.Flush()
}
