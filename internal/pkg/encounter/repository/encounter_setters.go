package repository

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
)

func (s *encounterStorage) SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, id string,
	userID int) error {
	fnName := utils.GetFunctionName()

	return dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return apperrors.TxStartError
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, SaveEncounterQuery, userID, encounter.Name, encounter.Data, id)
		if err != nil {
			return apperrors.TxError
		}

		if err := tx.Commit(ctx); err != nil {
			return apperrors.TxCommitError
		}

		return nil
	})
}

func (s *encounterStorage) UpdateEncounter(ctx context.Context, data []byte, id string) error {
	fnName := utils.GetFunctionName()

	return dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return apperrors.TxStartError
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, UpdateEncounterQuery, id, data)
		if err != nil {
			return apperrors.TxError
		}

		if err := tx.Commit(ctx); err != nil {
			return apperrors.TxCommitError
		}

		return nil
	})
}

func (s *encounterStorage) RemoveEncounter(ctx context.Context, id string) error {
	fnName := utils.GetFunctionName()

	return dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return apperrors.TxStartError
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, DeleteEncounterQuery, id)
		if err != nil {
			return apperrors.TxError
		}

		if err := tx.Commit(ctx); err != nil {
			return apperrors.TxCommitError
		}

		return nil
	})
}
