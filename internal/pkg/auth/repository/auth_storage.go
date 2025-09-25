package repository

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	authinterface "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/auth"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	mymetrics "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/metrics"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	serverrepo "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbinit"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
)

type authStorage struct {
	pool    serverrepo.PostgresPool
	metrics mymetrics.DBMetrics
}

func NewAuthStorage(pool serverrepo.PostgresPool, metrics mymetrics.DBMetrics) authinterface.AuthRepository {
	return &authStorage{
		pool:    pool,
		metrics: metrics,
	}
}

func (s *authStorage) CheckUser(ctx context.Context, vkid string) (*models.User, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var user models.User

	_, err := dbcall.DBCall[*models.User](fnName, s.metrics, func() (*models.User, error) {
		line := s.pool.QueryRow(ctx, CheckUserQuery, vkid)
		if err := line.Scan(&user.ID, &user.VKID, &user.Name, &user.Avatar); err != nil {
			return nil, err
		}

		return &user, nil
	})
	if err != nil {
		l.RepoWarn(err, map[string]any{"vk_id": vkid})
		return nil, apperrors.UserDoesNotExistError
	}

	return &user, err
}

func (s *authStorage) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var dbUser models.User

	_, err := dbcall.DBCall[*models.User](fnName, s.metrics, func() (*models.User, error) {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		line := tx.QueryRow(ctx, CreateUserQuery, user.VKID, user.Name, user.Avatar)
		if err := line.Scan(&dbUser.ID, &dbUser.VKID, &dbUser.Name, &dbUser.Avatar); err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return &dbUser, nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"vk_id": user.VKID, "name": user.Name, "avatar": user.Avatar})
		return nil, apperrors.TxError
	}

	return &dbUser, err
}

func (s *authStorage) UpdateUser(ctx context.Context, user *models.User) (*models.User, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var dbUser models.User

	_, err := dbcall.DBCall[*models.User](fnName, s.metrics, func() (*models.User, error) {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		line := tx.QueryRow(ctx, UpdateUserQuery, user.VKID, user.Name, user.Avatar)
		if err := line.Scan(&dbUser.ID, &dbUser.VKID, &dbUser.Name, &dbUser.Avatar); err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return &dbUser, nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"vk_id": user.VKID, "name": user.Name, "avatar": user.Avatar})
		return nil, apperrors.TxError
	}

	return &dbUser, nil
}
