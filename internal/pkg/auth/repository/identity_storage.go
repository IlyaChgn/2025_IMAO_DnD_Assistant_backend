package repository

import (
	"context"
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

type identityStorage struct {
	pool    serverrepo.PostgresPool
	metrics mymetrics.DBMetrics
}

func NewIdentityStorage(pool serverrepo.PostgresPool, metrics mymetrics.DBMetrics) authinterface.IdentityRepository {
	return &identityStorage{
		pool:    pool,
		metrics: metrics,
	}
}

func (s *identityStorage) FindByProvider(ctx context.Context, provider, providerUserID string) (*models.UserIdentity, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var identity models.UserIdentity

	_, err := dbcall.DBCall[*models.UserIdentity](fnName, s.metrics, func() (*models.UserIdentity, error) {
		line := s.pool.QueryRow(ctx, FindIdentityByProviderQuery, provider, providerUserID)
		if err := line.Scan(&identity.ID, &identity.UserID, &identity.Provider,
			&identity.ProviderUserID, &identity.Email); err != nil {
			return nil, err
		}

		return &identity, nil
	})
	if err != nil {
		l.RepoWarn(err, map[string]any{"provider": provider, "provider_user_id": providerUserID})
		return nil, apperrors.IdentityNotFoundError
	}

	return &identity, nil
}

func (s *identityStorage) CreateIdentity(ctx context.Context, identity *models.UserIdentity) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	return dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		_, err := s.pool.Exec(ctx, CreateIdentityQuery,
			identity.UserID, identity.Provider, identity.ProviderUserID, identity.Email)
		if err != nil {
			l.RepoError(err, map[string]any{
				"user_id": identity.UserID, "provider": identity.Provider,
				"provider_user_id": identity.ProviderUserID,
			})
			return apperrors.TxError
		}

		return nil
	})
}

func (s *identityStorage) UpdateLastUsed(ctx context.Context, identityID int, t time.Time) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	return dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		_, err := s.pool.Exec(ctx, UpdateIdentityLastUsedQuery, t, identityID)
		if err != nil {
			l.RepoWarn(err, map[string]any{"identity_id": identityID})
			return err
		}

		return nil
	})
}

func (s *identityStorage) ListByUserID(ctx context.Context, userID int) ([]models.UserIdentity, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	identities, err := dbcall.DBCall[[]models.UserIdentity](fnName, s.metrics, func() ([]models.UserIdentity, error) {
		rows, err := s.pool.Query(ctx, ListIdentitiesByUserIDQuery, userID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var result []models.UserIdentity
		for rows.Next() {
			var id models.UserIdentity
			if err := rows.Scan(&id.ID, &id.UserID, &id.Provider, &id.ProviderUserID,
				&id.Email, &id.CreatedAt); err != nil {
				return nil, err
			}

			result = append(result, id)
		}

		return result, rows.Err()
	})
	if err != nil {
		l.RepoError(err, map[string]any{"user_id": userID})
		return nil, err
	}

	return identities, nil
}

func (s *identityStorage) DeleteByUserAndProvider(ctx context.Context, userID int, provider string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	return dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		tag, err := s.pool.Exec(ctx, DeleteIdentityByUserAndProviderQuery, userID, provider)
		if err != nil {
			l.RepoError(err, map[string]any{"user_id": userID, "provider": provider})
			return apperrors.TxError
		}

		if tag.RowsAffected() == 0 {
			return apperrors.IdentityNotFoundError
		}

		return nil
	})
}
