package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)

		//		fmt.Printf("Hits: %d\n", cfg.fileserverHits.Load())
		next.ServeHTTP(w, req)
	})
}

func healthz(w http.ResponseWriter, req *http.Request) {
	str := "OK"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(str))
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	resp := fmt.Sprintf("<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>", cfg.fileserverHits.Load())
	w.Write([]byte(resp))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
	resp := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
	w.Write([]byte(resp))
}

func assets(w http.ResponseWriter, req *http.Request) {
	str := `
<pre>
	<a href="logo.png">logo.png</a>
</pre>
	`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	w.Write([]byte(str))
}

func main() {
	a := new(apiConfig)
	mux := http.NewServeMux()
	mux.Handle("/app/", a.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("./")))))
	mux.HandleFunc("GET /api/healthz", healthz)
	mux.HandleFunc("GET /app/assets", assets)
	mux.HandleFunc("GET /admin/metrics", a.metrics)
	mux.HandleFunc("POST /admin/reset", a.reset)

	err := http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
