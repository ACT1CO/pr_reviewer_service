package handlers

import (
	"encoding/json"
	"net/http"
	"reviewer_service/internal/domain"
	"reviewer_service/internal/service"
)

type AddTeamRequest struct {
	TeamName string          `json:"team_name"`
	Members  []TeamMemberDTO `json:"members"`
}

type TeamMemberDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

func AddTeamHandler(teamService *service.TeamService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AddTeamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		var members []domain.User
		for _, m := range req.Members {
			members = append(members, domain.User{
				ID:       m.UserID,
				Username: m.Username,
				IsActive: m.IsActive,
			})
		}

		team, err := teamService.AddTeam(req.TeamName, members)
		if err != nil {
			if _, ok := err.(service.TeamExistsError); ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "TEAM_EXISTS",
						"message": "team_name already exists",
					},
				})
				return
			}
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"team": map[string]interface{}{
				"team_name": team.Name,
				"members":   team.Members,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

type DeactivateUsersRequest struct {
	TeamName string   `json:"team_name"`
	UserIDs  []string `json:"user_ids"`
}

func DeactivateUsersHandler(teamService *service.TeamService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req DeactivateUsersRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.TeamName == "" || len(req.UserIDs) == 0 {
			http.Error(w, "team_name and user_ids are required", http.StatusBadRequest)
			return
		}

		if err := teamService.DeactivateUsersAndReassign(req.TeamName, req.UserIDs); err != nil {
			if _, ok := err.(service.TeamNotFoundError); ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "NOT_FOUND",
						"message": "team not found",
					},
				})
				return
			}
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status": "ok"}`)); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	}
}

func GetTeamHandler(teamService *service.TeamService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		teamName := r.URL.Query().Get("team_name")
		if teamName == "" {
			http.Error(w, "team_name is required", http.StatusBadRequest)
			return
		}

		team, err := teamService.GetTeam(teamName)
		if err != nil {
			switch err.(type) {
			case service.TeamNotFoundError:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "NOT_FOUND",
						"message": "team not found",
					},
				})
			default:
				http.Error(w, "Internal error", http.StatusInternalServerError)
			}
			return
		}

		var members []map[string]interface{}
		for _, m := range team.Members {
			members = append(members, map[string]interface{}{
				"user_id":   m.ID,
				"username":  m.Username,
				"is_active": m.IsActive,
			})
		}

		response := map[string]interface{}{
			"team_name": team.Name,
			"members":   members,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
