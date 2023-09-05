package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// get version from ENV variable VERSION
var Version = "development"

type Position struct {
	Id    string `json:"id"`
	Value int    `json:"value"`
}

type Request struct {
	Positions []Position `json:"positions"`
}

func main() {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.info.With(prometheus.Labels{"version": Version}).Set(1)

	dMux := http.NewServeMux()
	dMux.HandleFunc("/positions", positionsHandler)
	dMux.HandleFunc("/version", versionHandler)
	dMux.HandleFunc("/env", envHandler)

	pMux := http.NewServeMux()
	pMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	go func() {
		log.Default().Println("Server running on port 8080")
		log.Fatal(http.ListenAndServe(":8080", dMux))
	}()

	go func() {
		log.Default().Println("Server running on port 8081")
		log.Fatal(http.ListenAndServe(":8081", pMux))
	}()

	select {}
}

// list all environment variables
func envHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(os.Environ())
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"version": Version})
}

func positionsHandler(w http.ResponseWriter, r *http.Request) {

	// only accept POST requests
	if r.Method != "POST" {
		http.Error(w, "Invalid request method.", http.StatusMethodNotAllowed)
		return
	}

	// decode the request JSON body into Positions struct and fail if any error occur
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// merge all positions with the same id
	positions := make(map[string]int)
	for _, position := range req.Positions {
		positions[position.Id] += position.Value
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// return the positions in JSON format
	json.NewEncoder(w).Encode(positions)

}
