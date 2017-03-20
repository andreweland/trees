export GOPATH := ${PWD}:${GOPATH}
export GOBIN = ${PWD}/bin

all: pp map fe addresses

pp:
	go install src/camden/cmd/pp/pp.go

map:
	go install src/camden/cmd/map/map.go

fe: src/camden/proto/vector_tile.pb.go
	go install src/camden/cmd/fe/fe.go

addresses:
	go install src/camden/cmd/addresses/addresses.go

src/camden/proto/vector_tile.pb.go: proto/vector_tile.proto
	protoc --go_out=src/camden $<
