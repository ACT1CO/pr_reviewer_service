package handlers

import (
	"encoding/json"
	"net/http"
	"reviewer_service/internal/service"
)

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

func CreatePullRequestHandler(prService *service.PullRequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreatePRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		pr, err := prService.CreatePullRequest(req.PullRequestID, req.PullRequestName, req.AuthorID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			switch err.(type) {
			case service.PullRequestExistsError:
				w.WriteHeader(http.StatusConflict)
				if encodeErr := json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "PR_EXISTS",
						"message": "PR id already exists",
					},
				}); encodeErr != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				return
			case service.AuthorNotFoundError:
				w.WriteHeader(http.StatusNotFound)
				if encodeErr := json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "NOT_FOUND",
						"message": "author not found",
					},
				}); encodeErr != nil {
					http.Error(w, "Internal error", http.StatusInternalServerError)
					return
				}
				return
			default:
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
		}

		response := map[string]interface{}{
			"pr": map[string]interface{}{
				"pull_request_id":    pr.ID,
				"pull_request_name":  pr.Title,
				"author_id":          pr.AuthorID,
				"status":             pr.Status,
				"assigned_reviewers": pr.AssignedReviewers,
				"createdAt":          pr.CreatedAt,
				"mergedAt":           pr.MergedAt,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	}
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

func MergePullRequestHandler(prService *service.PullRequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MergePRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		pr, err := prService.MergePullRequest(req.PullRequestID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "NOT_FOUND",
					"message": "PR not found",
				},
			}); err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			return
		}

		response := map[string]interface{}{
			"pr": map[string]interface{}{
				"pull_request_id":    pr.ID,
				"pull_request_name":  pr.Title,
				"author_id":          pr.AuthorID,
				"status":             pr.Status,
				"assigned_reviewers": pr.AssignedReviewers,
				"createdAt":          pr.CreatedAt,
				"mergedAt":           pr.MergedAt,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	}
}

type ReassignRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldReviewerID string `json:"old_user_id"`
}

func ReassignReviewerHandler(prService *service.PullRequestService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ReassignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		newID, pr, err := prService.ReassignReviewer(req.PullRequestID, req.OldReviewerID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			var code int
			var errBody map[string]interface{}
			switch err.(type) {
			case service.PRMergedError:
				code = http.StatusConflict
				errBody = map[string]interface{}{
					"error": map[string]string{
						"code":    "PR_MERGED",
						"message": "cannot reassign on merged PR",
					},
				}
			case service.NotAssignedError:
				code = http.StatusConflict
				errBody = map[string]interface{}{
					"error": map[string]string{
						"code":    "NOT_ASSIGNED",
						"message": "reviewer is not assigned to this PR",
					},
				}
			case service.NoCandidateError:
				code = http.StatusConflict
				errBody = map[string]interface{}{
					"error": map[string]string{
						"code":    "NO_CANDIDATE",
						"message": "no active replacement candidate in team",
					},
				}
			case service.AuthorNotFoundError:
				code = http.StatusNotFound
				errBody = map[string]interface{}{
					"error": map[string]string{
						"code":    "NOT_FOUND",
						"message": "PR or user not found",
					},
				}
			default:
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(code)
			if err := json.NewEncoder(w).Encode(errBody); err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			return
		}

		response := map[string]interface{}{
			"pr": map[string]interface{}{
				"pull_request_id":    pr.ID,
				"pull_request_name":  pr.Title,
				"author_id":          pr.AuthorID,
				"status":             pr.Status,
				"assigned_reviewers": pr.AssignedReviewers,
				"createdAt":          pr.CreatedAt,
				"mergedAt":           pr.MergedAt,
			},
			"replaced_by": newID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
	}
}
