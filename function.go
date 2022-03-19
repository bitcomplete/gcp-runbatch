package runbatch

import (
	"encoding/json"
	"net/http"
)

// Function is the entrypoint for running runbatch as a Cloud Function.
func Function(w http.ResponseWriter, r *http.Request) {
	var input Input
	err := json.NewDecoder(r.Body).Decode(&input)
	r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	output, err := Start(r.Context(), &input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
