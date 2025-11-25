package service

import (
	"database/sql"
	"errors"
	"math/rand"
	"reviewer_service/internal/domain"
	"reviewer_service/internal/repository"
	"time"
)

type PullRequestService struct {
	prRepo   repository.PullRequestRepository
	userRepo repository.UserRepository
	teamRepo repository.TeamRepository
}

func NewPullRequestService(prRepo repository.PullRequestRepository, userRepo repository.UserRepository, teamRepo repository.TeamRepository) *PullRequestService {
	return &PullRequestService{prRepo: prRepo, userRepo: userRepo, teamRepo: teamRepo}
}

type PullRequestExistsError struct{}

func (e PullRequestExistsError) Error() string { return "PR already exists" }

type AuthorNotFoundError struct{}

func (e AuthorNotFoundError) Error() string { return "author not found" }

type PRMergedError struct{}

func (e PRMergedError) Error() string { return "cannot reassign on merged PR" }

type NotAssignedError struct{}

func (e NotAssignedError) Error() string { return "reviewer is not assigned to this PR" }

type NoCandidateError struct{}

func (e NoCandidateError) Error() string { return "no active replacement candidate in team" }

func (s *PullRequestService) CreatePullRequest(id, name, authorID string) (*domain.PullRequest, error) {
	existing, _ := s.prRepo.GetByID(id)
	if existing != nil {
		return nil, PullRequestExistsError{}
	}

	teamID, err := s.userRepo.GetTeamIDByUserID(authorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, AuthorNotFoundError{}
		}
		return nil, err
	}

	candidates, err := s.userRepo.GetActiveUsersInTeamExcluding(teamID, authorID)
	if err != nil {
		return nil, err
	}

	var reviewers []string
	for i, user := range candidates {
		if i >= 2 {
			break
		}
		reviewers = append(reviewers, user.ID)
	}

	now := time.Now()
	pr := &domain.PullRequest{
		ID:                id,
		Title:             name,
		AuthorID:          authorID,
		Status:            "OPEN",
		AssignedReviewers: reviewers,
		CreatedAt:         &now,
		MergedAt:          nil,
	}

	if err := s.prRepo.Create(pr); err != nil {
		return nil, err
	}
	if err := s.prRepo.AssignReviewers(pr.ID, reviewers); err != nil {
		return nil, err
	}

	return pr, nil
}

const StatusMerged = "MERGED"

func (s *PullRequestService) MergePullRequest(prID string) (*domain.PullRequest, error) {
	pr, err := s.prRepo.GetByID(prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, AuthorNotFoundError{}
		}
		return nil, err
	}
	if pr == nil {
		return nil, AuthorNotFoundError{}
	}

	if pr.Status == StatusMerged {
		return pr, nil
	}

	now := time.Now()
	if err := s.prRepo.Merge(prID, now); err != nil {
		return nil, err
	}

	pr.Status = "MERGED"
	pr.MergedAt = &now

	return pr, nil
}

func (s *PullRequestService) ReassignReviewer(prID, oldReviewerID string) (newReviewerID string, pr *domain.PullRequest, err error) {
	pr, err = s.prRepo.GetByID(prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, AuthorNotFoundError{}
		}
		return "", nil, err
	}
	if pr == nil {
		return "", nil, AuthorNotFoundError{}
	}

	if pr.Status == "MERGED" {
		return "", nil, PRMergedError{}
	}

	reviewers, err := s.prRepo.GetReviewers(prID)
	if err != nil {
		return "", nil, err
	}

	isAssigned := false
	for _, r := range reviewers {
		if r == oldReviewerID {
			isAssigned = true
			break
		}
	}
	if !isAssigned {
		return "", nil, NotAssignedError{}
	}

	team, err := s.userRepo.GetTeamByUserID(oldReviewerID)
	if err != nil {
		return "", nil, err
	}

	candidates, err := s.userRepo.GetActiveUsersInTeamExcluding(team.ID, oldReviewerID)
	if err != nil {
		return "", nil, err
	}

	if len(candidates) == 0 {
		return "", nil, NoCandidateError{}
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	newReviewer := candidates[rng.Intn(len(candidates))]
	newReviewerID = newReviewer.ID

	if err := s.prRepo.ReplaceReviewer(prID, oldReviewerID, newReviewerID); err != nil {
		return "", nil, err
	}

	for i, r := range pr.AssignedReviewers {
		if r == oldReviewerID {
			pr.AssignedReviewers[i] = newReviewerID
			break
		}
	}

	return newReviewerID, pr, nil
}

func (s *PullRequestService) GetReviewPRs(userID string) ([]*domain.PullRequest, error) {
	_, err := s.userRepo.GetTeamIDByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, AuthorNotFoundError{}
		}
		return nil, err
	}

	prs, err := s.prRepo.GetPRsByReviewer(userID)
	if err != nil {
		return nil, err
	}
	return prs, nil
}

func (s *PullRequestService) GetReviewStats() (map[string]int, error) {
	return s.prRepo.GetReviewStats()
}
