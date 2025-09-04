package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	/*	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path != "/" {
				http.NotFound(w, req)
				return
			}

			//		fmt.Fprintf(w, "Hello world!")
		})
	*/
	err := http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
