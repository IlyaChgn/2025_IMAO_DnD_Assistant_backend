package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server/repository/dbcall"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/utils"
	"github.com/jackc/pgx/v5"
)

func (s *encounterStorage) GetEncountersList(ctx context.Context,
	size, start, userID int) (*models.EncountersList, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	rows, err := dbcall.DBCall[pgx.Rows](fnName, s.metrics, func() (pgx.Rows, error) {
		return s.pool.Query(ctx, GetEncountersListQuery, userID, size, start)
	})
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		l.RepoWarn(err, map[string]any{"size": size, "start": start, "userID": userID})
		return nil, nil
	} else if err != nil {
		l.RepoError(err, map[string]any{"size": size, "start": start, "userID": userID})
		return nil, apperrors.QueryError
	}
	defer rows.Close()

	var list models.EncountersList

	for rows.Next() {
		var encounter models.EncounterInList

		if err := rows.Scan(&encounter.UserID, &encounter.Name, &encounter.UUID); err != nil {
			l.RepoError(err, map[string]any{"size": size, "start": start, "userID": userID})
			return nil, apperrors.ScanError
		}

		list = append(list, &encounter)
	}

	return &list, nil
}

func (s *encounterStorage) GetEncountersListWithSearch(ctx context.Context, size, start, userID int,
	search *models.SearchParams) (*models.EncountersList, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	searchValue := fmt.Sprintf("%s:*", search.Value)

	rows, err := dbcall.DBCall[pgx.Rows](fnName, s.metrics, func() (pgx.Rows, error) {
		return s.pool.Query(ctx, GetEncountersListWithSearchQuery, userID, searchValue, size, start)
	})
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		l.RepoWarn(err, map[string]any{"size": size, "start": start, "userID": userID, "search": searchValue})
		return nil, nil
	} else if err != nil {
		l.RepoError(err, map[string]any{"size": size, "start": start, "userID": userID, "search": searchValue})
		return nil, apperrors.QueryError
	}
	defer rows.Close()

	var list models.EncountersList

	for rows.Next() {
		var encounter models.EncounterInList

		if err := rows.Scan(&encounter.UserID, &encounter.Name, &encounter.UUID); err != nil {
			l.RepoError(err, map[string]any{"size": size, "start": start, "userID": userID, "search": searchValue})
			return nil, apperrors.ScanError
		}

		list = append(list, &encounter)
	}

	return &list, nil
}

func (s *encounterStorage) GetEncounterByID(ctx context.Context, id string) (*models.Encounter, error) {
	l := logger.FromContext(ctx)
	fnName := utils.GetFunctionName()

	var encounter models.Encounter

	_, err := dbcall.DBCall[*models.Encounter](fnName, s.metrics, func() (*models.Encounter, error) {
		line := s.pool.QueryRow(ctx, GetEncounterByIDQuery, id)
		if err := line.Scan(&encounter.UserID, &encounter.Name, &encounter.Data,
			&encounter.UUID); err != nil {
			return nil, err
		}

		return &encounter, nil
	})
	if err != nil {
		l.RepoError(err, map[string]any{"id": id})
		return nil, apperrors.ScanError
	}

	return &encounter, nil
}
