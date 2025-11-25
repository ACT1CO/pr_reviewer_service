package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

func TestE2E(t *testing.T) {
	time.Sleep(10 * time.Second)

	t.Run("CreateTeam", func(t *testing.T) {
		body := map[string]interface{}{
			"team_name": "backend",
			"members": []map[string]interface{}{
				{"user_id": "u1", "username": "Alice", "is_active": true},
				{"user_id": "u2", "username": "Bob", "is_active": true},
			},
		}
		resp, err := http.Post(baseURL+"/team/add", "application/json", jsonBody(body))
		if err != nil {
			t.Fatalf("Failed to create team: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 201, got %d", resp.StatusCode)
		}
	})

	t.Run("CreatePR", func(t *testing.T) {
		body := map[string]interface{}{
			"pull_request_id":   "pr-1",
			"pull_request_name": "Add feature",
			"author_id":         "u1",
		}
		resp, err := http.Post(baseURL+"/pullRequest/create", "application/json", jsonBody(body))
		if err != nil {
			t.Fatalf("Failed to create PR: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected 201, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		pr := result["pr"].(map[string]interface{})
		reviewers := pr["assigned_reviewers"].([]interface{})
		if len(reviewers) != 1 {
			t.Errorf("Expected 1 reviewer, got %d", len(reviewers))
		}
		if reviewers[0] != "u2" {
			t.Errorf("Expected reviewer u2, got %v", reviewers[0])
		}
	})

	t.Run("MergePR", func(t *testing.T) {
		body := map[string]interface{}{
			"pull_request_id": "pr-1",
		}
		resp, err := http.Post(baseURL+"/pullRequest/merge", "application/json", jsonBody(body))
		if err != nil {
			t.Fatalf("Failed to merge PR: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})
}

func jsonBody(v interface{}) *bytes.Buffer {
	data, _ := json.Marshal(v)
	return bytes.NewBuffer(data)
}
