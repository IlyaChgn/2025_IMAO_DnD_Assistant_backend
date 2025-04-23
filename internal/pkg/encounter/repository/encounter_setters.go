package repository

import (
	"context"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
)

func (s *encounterStorage) SaveEncounter(ctx context.Context, encounter *models.SaveEncounterReq, userID int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return apperrors.TxStartError
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, SaveEncounterQuery, userID, encounter.Name, encounter.Data)
	if err != nil {
		return apperrors.TxError
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.TxCommitError
	}

	return nil
}

func (s *encounterStorage) UpdateEncounter(ctx context.Context, data []byte, id int) error {
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
}

func (s *encounterStorage) RemoveEncounter(ctx context.Context, id int) error {
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
}
