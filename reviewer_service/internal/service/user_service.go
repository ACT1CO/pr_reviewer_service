package service

import (
	"reviewer_service/internal/domain"
	"reviewer_service/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
	teamRepo repository.TeamRepository
}

func NewUserService(userRepo repository.UserRepository, teamRepo repository.TeamRepository) *UserService {
	return &UserService{userRepo: userRepo, teamRepo: teamRepo}
}

func (s *UserService) SetIsActive(userID string, isActive bool) (*domain.User, error) {
	if err := s.userRepo.SetIsActive(userID, isActive); err != nil {
		return nil, err
	}
	return s.userRepo.GetUserByID(userID)
}
