package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"trees"

	"github.com/gorilla/handlers"
)

func main() {
	data := flag.String("data", "data/trees.json", "Filename containing JSON tree data")
	addr := flag.String("addr", "localhost:9000", "Host and port on which to serve HTTP")
	static := flag.String("static", "static", "Directory containing static content")
	flag.Parse()
	camdenTrees, err := trees.LoadCamdenTrees(*data)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Loaded %d trees", len(camdenTrees))
	sort.Sort(trees.ByLocation(camdenTrees))
	http.Handle("/", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(*static, "map.html"))
	})))
	http.Handle("/tile/", handlers.LoggingHandler(os.Stdout, &trees.TileHandler{Trees: camdenTrees}))
	server := http.Server{Addr: *addr}
	log.Printf("Listening on %s", *addr)
	log.Fatal(server.ListenAndServe())
}
