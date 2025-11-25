package handlers

import (
	"encoding/json"
	"net/http"
	"reviewer_service/internal/service"
)

type SetIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

func SetIsActiveHandler(userService *service.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SetIsActiveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		user, err := userService.SetIsActive(req.UserID, req.IsActive)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "NOT_FOUND",
					"message": "user not found",
				},
			})
			return
		}

		response := map[string]interface{}{
			"user": map[string]interface{}{
				"user_id":   user.ID,
				"username":  user.Username,
				"team_name": user.TeamName,
				"is_active": user.IsActive,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

func GetReviewPRsHandler(prService *service.PullRequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		prs, err := prService.GetReviewPRs(userID)
		if err != nil {
			switch err.(type) {
			case service.AuthorNotFoundError:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "NOT_FOUND",
						"message": "user not found",
					},
				})
			default:
				http.Error(w, "Internal error", http.StatusInternalServerError)
			}
			return
		}

		response := map[string]interface{}{
			"user_id":       userID,
			"pull_requests": prs,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
