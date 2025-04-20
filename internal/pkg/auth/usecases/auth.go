package usecases

import authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"

type authUsecases struct {
	repo           authinterface.AuthRepository
	sessionManager authinterface.SessionManager
}

func NewAuthUsecases(repo authinterface.AuthRepository,
	sessionManager authinterface.SessionManager) authinterface.AuthUsecases {
	return &authUsecases{
		repo:           repo,
		sessionManager: sessionManager,
	}
}
