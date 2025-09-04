package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./")))

	err := http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
