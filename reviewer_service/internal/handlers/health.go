package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"service":   "reviewer_service",
		"version":   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
