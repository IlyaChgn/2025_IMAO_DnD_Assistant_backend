package usecases

import authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"

type authUsecases struct {
	repo authinterface.AuthRepository
}

func NewAuthUsecases(repo authinterface.AuthRepository) authinterface.AuthUsecases {
	return &authUsecases{
		repo: repo,
	}
}
