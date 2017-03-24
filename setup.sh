#!/bin/sh -x
export GOPATH=${PWD}:${GOPATH}
go get github.com/golang/protobuf/proto
go get github.com/golang/geo/s2
go get github.com/paulmach/go.geo
curl -o data/trees.json 'https://opendata.camden.gov.uk/resource/2ujt-4csu.json?$limit=20000'
protoc --go_out=src/trees proto/vector_tile.proto
