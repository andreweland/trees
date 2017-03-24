package main

import (
	"log"
	"net/http"
	"sort"
	"trees"
)

func main() {
	camdenTrees, err := trees.LoadCamdenTrees("data/trees.json")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Loaded %d trees", len(camdenTrees))
	sort.Sort(trees.ByLocation(camdenTrees))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "html/map.html")
	})
	http.Handle("/tile/", &trees.TileHandler{Trees: camdenTrees})
	addr := "localhost:9000"
	server := http.Server{Addr: addr}
	log.Printf("Listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
