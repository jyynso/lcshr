package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	dir := flag.String("dir", ".", "folder to serve")
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	fs := http.FileServer(http.Dir(*dir))
	http.Handle("/", fs)

	addr := "0.0.0.0:" + *port
	log.Printf("running at http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
