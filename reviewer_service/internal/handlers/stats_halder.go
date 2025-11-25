package handlers

import (
	"encoding/json"
	"net/http"
	"reviewer_service/internal/service"
)

func GetReviewStatsHandler(prService *service.PullRequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		stats, err := prService.GetReviewStats()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"review_assignments": stats,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
