# Camden Trees

A simple [map of council owned trees in Camden](http://trees.andreweland.org), with data from [Open Data Camden](https://opendata.camden.gov.uk/Environment/Trees-In-Camden/csqp-kdss), using
vector tiles created on the fly by a [Go](http://www.golang.org/) server with an [S2](https://docs.google.com/presentation/d/1Hl4KapfAENAOf4gv-pSngKwvS_jwNVHRPZTTDzXXn6Q/view#slide=id.i0)
spatial index, over a [MapBox](http://www.mapbox.com) basemap. Vector tiles are
rendered in the browser with [Leaflet](http://leafletjs.com/) and [Leaflet.VectorGrid](https://github.com/Leaflet/Leaflet.VectorGrid).

## Building & running

Setup your environment (OSX, using [Homebrew](https://brew.sh/))

    brew install go
    brew install protobuf    
    ./setup.sh # Download data and Go dependencies
    ./build.sh
    bin/frontend --addr=localhost:9000
