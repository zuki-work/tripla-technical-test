package services

import (
	"tripla-technical-test/internal/models"
	"tripla-technical-test/internal/repositories"
)

type UserService struct {
	userRepository *repositories.UserRepository
}

func NewUserService(userRepository *repositories.UserRepository) *UserService {
	return &UserService{userRepository: userRepository}
}

func (s *UserService) CreateUser(user *models.User) (*models.User, error) {
	if err := s.userRepository.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUsers() ([]models.User, error) {
	return s.userRepository.FindAll()
}
