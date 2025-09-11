package repository

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
)

func (s *encounterStorage) SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, id string,
	userID int) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, SaveEncounterQuery, userID, encounter.Name, encounter.Data, id)
		if err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id, "encounter_name": encounter.Name})
		return apperrors.TxError
	}

	return nil
}

func (s *encounterStorage) UpdateEncounter(ctx context.Context, data []byte, id string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, UpdateEncounterQuery, id, data)
		if err != nil {
			return apperrors.TxError
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return apperrors.TxError
	}

	return nil
}

func (s *encounterStorage) RemoveEncounter(ctx context.Context, id string) error {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	err := dbcall.ErrOnlyDBCall(fnName, s.metrics, func() error {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, DeleteEncounterQuery, id)
		if err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return apperrors.TxError
	}

	return nil
}
