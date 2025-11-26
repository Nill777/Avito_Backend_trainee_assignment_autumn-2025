package service

import (
	"math/rand/v2"

	"reviewer/internal/domain"
	"reviewer/internal/repository"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

// Вспомогательная функция (общая)
func (s *Service) pickRandomReviewers(users []domain.User, n int) []domain.User {
	if len(users) == 0 || n <= 0 {
		return []domain.User{}
	}
	shuffled := make([]domain.User, len(users))
	copy(shuffled, users)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	if n > len(shuffled) {
		n = len(shuffled)
	}
	return shuffled[:n]
}
