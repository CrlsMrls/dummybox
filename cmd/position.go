package cmd

import (
	"encoding/json"
	"net/http"
)

type Position struct {
	Id    string `json:"id"`
	Value int    `json:"value"`
}

type Request struct {
	Positions []Position `json:"positions"`
}

func PositionsHandler(w http.ResponseWriter, r *http.Request) {

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
