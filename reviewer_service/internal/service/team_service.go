package service

import (
	"reviewer_service/internal/domain"
	"reviewer_service/internal/repository"
)

type TeamService struct {
	teamRepo  repository.TeamRepository
	userRepo  repository.UserRepository
	prRepo    repository.PullRequestRepository
	prService *PullRequestService
}

func NewTeamService(teamRepo repository.TeamRepository, userRepo repository.UserRepository, prRepo repository.PullRequestRepository, prService *PullRequestService) *TeamService {
	return &TeamService{teamRepo: teamRepo, userRepo: userRepo, prRepo: prRepo, prService: prService}
}

type TeamExistsError struct{}

func (e TeamExistsError) Error() string { return "team already exists" }

type TeamNotFoundError struct{}

func (e TeamNotFoundError) Error() string { return "team not found" }

func (s *TeamService) AddTeam(name string, members []domain.User) (*domain.Team, error) {
	exists, err := s.teamRepo.Exists(name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, TeamExistsError{}
	}

	teamID, err := s.teamRepo.Create(name)
	if err != nil {
		return nil, err
	}

	for i := range members {
		members[i].TeamID = teamID
	}

	err = s.userRepo.UpsertMany(members)
	if err != nil {
		return nil, err
	}

	return &domain.Team{
		ID:      teamID,
		Name:    name,
		Members: members,
	}, nil
}

func (s *TeamService) DeactivateUsersAndReassign(teamName string, userIDs []string) error {
	exists, err := s.teamRepo.Exists(teamName)
	if err != nil {
		return err
	}
	if !exists {
		return TeamNotFoundError{}
	}

	openPRs, err := s.prRepo.GetOpenPRsWithReviewers(userIDs)
	if err != nil {
		return err
	}

	for _, pr := range openPRs {
		reviewers, err := s.prRepo.GetReviewers(pr.ID)
		if err != nil {
			return err
		}

		for _, reviewerID := range reviewers {
			for _, id := range userIDs {
				if id == reviewerID {
					_, _, err := s.prService.ReassignReviewer(pr.ID, reviewerID)
					if err != nil {
						if err.Error() != "no active replacement candidate in team" {
							return err
						}
					}
					break
				}
			}
		}
	}

	return s.userRepo.DeactivateUsers(userIDs)
}

func (s *TeamService) GetTeam(teamName string) (*domain.Team, error) {
	team, err := s.teamRepo.GetByName(teamName)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, TeamNotFoundError{}
	}
	return team, nil
}
