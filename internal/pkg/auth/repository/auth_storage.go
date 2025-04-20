package repository

import authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"

type authStorage struct {
}

func NewAuthStorage() authinterface.AuthRepository {
	return &authStorage{}
}
