package repository

import (
	"context"
	"errors"
	"time"

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

func (s *authStorage) GetUserByID(ctx context.Context, userID int) (*models.User, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var user models.User

	_, err := dbcall.DBCall[*models.User](fnName, s.metrics, func() (*models.User, error) {
		line := s.pool.QueryRow(ctx, GetUserByIDQuery, userID)
		if err := line.Scan(&user.ID, &user.DisplayName, &user.AvatarURL, &user.Status); err != nil {
			return nil, err
		}

		return &user, nil
	})
	if err != nil {
		l.RepoWarn(err, map[string]any{"user_id": userID})
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

		line := tx.QueryRow(ctx, CreateUserQuery, user.DisplayName, user.AvatarURL)
		if err := line.Scan(&dbUser.ID, &dbUser.DisplayName, &dbUser.AvatarURL, &dbUser.Status); err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return &dbUser, nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"name": user.DisplayName, "avatar": user.AvatarURL})
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

		line := tx.QueryRow(ctx, UpdateUserQuery, user.ID, user.DisplayName, user.AvatarURL)
		if err := line.Scan(&dbUser.ID, &dbUser.DisplayName, &dbUser.AvatarURL, &dbUser.Status); err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return &dbUser, nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"user_id": user.ID, "name": user.DisplayName, "avatar": user.AvatarURL})
		return nil, apperrors.TxError
	}

	return &dbUser, nil
}

func (s *authStorage) CreateUserWithIdentity(ctx context.Context, user *models.User,
	identity *models.UserIdentity) (*models.User, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var dbUser models.User

	_, err := dbcall.DBCall[*models.User](fnName, s.metrics, func() (*models.User, error) {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		line := tx.QueryRow(ctx, CreateUserQuery, user.DisplayName, user.AvatarURL)
		if err := line.Scan(&dbUser.ID, &dbUser.DisplayName, &dbUser.AvatarURL, &dbUser.Status); err != nil {
			return nil, err
		}

		tag, err := tx.Exec(ctx, CreateIdentityOnConflictQuery,
			dbUser.ID, identity.Provider, identity.ProviderUserID, identity.Email)
		if err != nil {
			return nil, err
		}

		if tag.RowsAffected() == 0 {
			return nil, apperrors.IdentityAlreadyLinkedError
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return &dbUser, nil
	})
	if err != nil {
		if errors.Is(err, apperrors.IdentityAlreadyLinkedError) {
			return nil, apperrors.IdentityAlreadyLinkedError
		}

		l.RepoError(err, map[string]any{
			"name": user.DisplayName, "provider": identity.Provider,
		})
		return nil, apperrors.TxError
	}

	return &dbUser, nil
}

func (s *authStorage) UpdateLastLoginAt(ctx context.Context, userID int, t time.Time) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		_, execErr := s.pool.Exec(ctx, UpdateLastLoginAtQuery, t, userID)
		return execErr
	})
	if err != nil {
		l.RepoWarn(err, map[string]any{"user_id": userID})
		return err
	}

	return nil
}
