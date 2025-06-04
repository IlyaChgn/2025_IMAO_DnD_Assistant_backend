package repository

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	serverrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbinit"
)

type authStorage struct {
	pool    serverrepo.PostgresPool
	metrics *mymetrics.DBMetrics
}

func NewAuthStorage(pool serverrepo.PostgresPool, metrics *mymetrics.DBMetrics) authinterface.AuthRepository {
	return &authStorage{
		pool:    pool,
		metrics: metrics,
	}
}

func (s *authStorage) CheckUser(ctx context.Context, vkid string) (*models.User, error) {
	var user models.User

	line := s.pool.QueryRow(ctx, CheckUserQuery, vkid)
	if err := line.Scan(&user.ID, &user.VKID, &user.Name, &user.Avatar); err != nil {
		return nil, apperrors.UserDoesNotExistError
	}

	return &user, nil
}

func (s *authStorage) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, apperrors.TxStartError
	}
	defer tx.Rollback(ctx)

	var dbUser models.User

	line := tx.QueryRow(ctx, CreateUserQuery, user.VKID, user.Name, user.Avatar)
	if err := line.Scan(&dbUser.ID, &dbUser.VKID, &dbUser.Name, &dbUser.Avatar); err != nil {
		return nil, apperrors.TxError
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.TxCommitError
	}

	return &dbUser, nil
}

func (s *authStorage) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, apperrors.TxStartError
	}
	defer tx.Rollback(ctx)

	var dbUser models.User

	line := tx.QueryRow(ctx, UpdateUserQuery, user.VKID, user.Name, user.Avatar)
	if err := line.Scan(&dbUser.ID, &dbUser.VKID, &dbUser.Name, &dbUser.Avatar); err != nil {
		return nil, apperrors.TxError
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.TxCommitError
	}

	return &dbUser, nil
}
