package repository

import (
	"database/sql"
	"fmt"
	"log"
	"reviewer_service/internal/domain"
	"strconv"
	"strings"
	"time"
)

type PullRequestRepository interface {
	Create(pr *domain.PullRequest) error
	GetByID(id string) (*domain.PullRequest, error)
	AssignReviewers(prID string, reviewerIDs []string) error
	Merge(prID string, mergedAt time.Time) error
	GetReviewers(prID string) ([]string, error)
	ReplaceReviewer(prID, oldReviewerID, newReviewerID string) error
	GetPRsByReviewer(reviewerID string) ([]*domain.PullRequest, error)
	GetReviewStats() (map[string]int, error)
	GetOpenPRsWithReviewers(userIDs []string) ([]*domain.PullRequest, error)
}

type PostgresPullRequestRepository struct {
	db *sql.DB
}

func NewPullRequestRepository(db *sql.DB) *PostgresPullRequestRepository {
	return &PostgresPullRequestRepository{db: db}
}

func (r *PostgresPullRequestRepository) Create(pr *domain.PullRequest) error {
	_, err := r.db.Exec(`
		INSERT INTO pull_requests (id, title, author_id, status, created_at, merged_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, pr.ID, pr.Title, pr.AuthorID, pr.Status, pr.CreatedAt, pr.MergedAt)
	return err
}

func (r *PostgresPullRequestRepository) AssignReviewers(prID string, reviewerIDs []string) error {
	if len(reviewerIDs) == 0 {
		return nil
	}
	if len(reviewerIDs) > 2 {
		reviewerIDs = reviewerIDs[:2]
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO pr_reviewers (pr_id, reviewer_id) VALUES ($1, $2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range reviewerIDs {
		_, err := stmt.Exec(prID, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresPullRequestRepository) GetByID(id string) (*domain.PullRequest, error) {
	var pr domain.PullRequest
	var createdAt, mergedAt sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, title, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE id = $1
	`, id).Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)
	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		pr.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}

	rows, err := r.db.Query("SELECT reviewer_id FROM pr_reviewers WHERE pr_id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = append(pr.AssignedReviewers, reviewerID)
	}

	return &pr, nil
}

func (r *PostgresPullRequestRepository) Merge(prID string, mergedAt time.Time) error {
	_, err := r.db.Exec(`
		UPDATE pull_requests
		SET status = 'MERGED', merged_at = $1
		WHERE id = $2 AND status = 'OPEN'
	`, mergedAt, prID)
	return err
}

func (r *PostgresPullRequestRepository) GetReviewers(prID string) ([]string, error) {
	rows, err := r.db.Query("SELECT reviewer_id FROM pr_reviewers WHERE pr_id = $1", prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, id)
	}
	return reviewers, nil
}

func (r *PostgresPullRequestRepository) ReplaceReviewer(prID, oldReviewerID, newReviewerID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			log.Printf("Failed to rollback transaction: %v", err)
		}
	}()

	_, err = tx.Exec("DELETE FROM pr_reviewers WHERE pr_id = $1 AND reviewer_id = $2", prID, oldReviewerID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO pr_reviewers (pr_id, reviewer_id) VALUES ($1, $2)", prID, newReviewerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PostgresPullRequestRepository) GetPRsByReviewer(reviewerID string) ([]*domain.PullRequest, error) {
	query := `
		SELECT pr.id, pr.title, pr.author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		JOIN pr_reviewers prr ON pr.id = prr.pr_id
		WHERE prr.reviewer_id = $1
	`
	rows, err := r.db.Query(query, reviewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*domain.PullRequest
	for rows.Next() {
		var pr domain.PullRequest
		var createdAt, mergedAt sql.NullTime
		if err := rows.Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			pr.CreatedAt = &createdAt.Time
		}
		if mergedAt.Valid {
			pr.MergedAt = &mergedAt.Time
		}
		prs = append(prs, &pr)
	}
	return prs, nil
}

func (r *PostgresPullRequestRepository) GetReviewStats() (map[string]int, error) {
	query := `
		SELECT reviewer_id, COUNT(*) as count
		FROM pr_reviewers
		GROUP BY reviewer_id
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var userID string
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			return nil, err
		}
		stats[userID] = count
	}
	return stats, nil
}

func (r *PostgresPullRequestRepository) GetOpenPRsWithReviewers(userIDs []string) ([]*domain.PullRequest, error) {
	if len(userIDs) == 0 {
		return []*domain.PullRequest{}, nil
	}

	placeholders := make([]string, len(userIDs))
	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT pr.id, pr.title, pr.author_id, pr.status, pr.created_at, pr.merged_at
		FROM pull_requests pr
		JOIN pr_reviewers prr ON pr.id = prr.pr_id
		WHERE pr.status = 'OPEN' AND prr.reviewer_id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*domain.PullRequest
	for rows.Next() {
		var pr domain.PullRequest
		var createdAt, mergedAt sql.NullTime
		if err := rows.Scan(&pr.ID, &pr.Title, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			pr.CreatedAt = &createdAt.Time
		}
		if mergedAt.Valid {
			pr.MergedAt = &mergedAt.Time
		}
		prs = append(prs, &pr)
	}
	return prs, nil
}
