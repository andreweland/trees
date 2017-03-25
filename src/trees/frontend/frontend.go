package main

import (
	"flag"
	"log"
	"net/http"
	"sort"
	"trees"
)

func main() {
	data := flag.String("data", "data/trees.json", "Filename containing JSON tree data")
	addr := flag.String("addr", "localhost:9000", "Host and port on which to serve HTTP")
	flag.Parse()
	camdenTrees, err := trees.LoadCamdenTrees(*data)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Loaded %d trees", len(camdenTrees))
	sort.Sort(trees.ByLocation(camdenTrees))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "html/map.html")
	})
	http.Handle("/tile/", &trees.TileHandler{Trees: camdenTrees})
	server := http.Server{Addr: *addr}
	log.Printf("Listening on %s", *addr)
	log.Fatal(server.ListenAndServe())
}
