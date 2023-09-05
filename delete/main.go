package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	dMux := http.NewServeMux()
	dMux.HandleFunc("/version", versionHandler)
	dMux.HandleFunc("/env", envHandler)

	go func() {
		log.Default().Println("Server running on port 8080")
		log.Fatal(http.ListenAndServe(":8080", dMux))
	}()

	select {}
}

// list all environment variables
func envHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(os.Environ())
}
