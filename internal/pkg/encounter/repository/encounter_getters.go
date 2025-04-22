package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	"github.com/jackc/pgx/v5"
)

func (s *encounterStorage) GetEncountersList(ctx context.Context,
	size, start, userID int) (*models.EncountersList, error) {
	rows, err := s.pool.Query(ctx, GetEncountersListQuery, userID, size, start)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, apperrors.QueryError
	}

	var list models.EncountersList

	for rows.Next() {
		var encounter models.EncounterInList

		if err := rows.Scan(&encounter.ID, &encounter.UserID, &encounter.Name); err != nil {
			return nil, apperrors.ScanError
		}

		list = append(list, &encounter)
	}

	return &list, nil
}

func (s *encounterStorage) GetEncountersListWithSearch(ctx context.Context, size, start, userID int,
	search *models.SearchParams) (*models.EncountersList, error) {
	searchValue := fmt.Sprintf("%s:*", search.Value)

	rows, err := s.pool.Query(ctx, GetEncountersListWithSearchQuery, userID, searchValue, size, start)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, apperrors.QueryError
	}

	var list models.EncountersList

	for rows.Next() {
		var encounter models.EncounterInList

		if err := rows.Scan(&encounter.ID, &encounter.UserID, &encounter.Name); err != nil {
			return nil, apperrors.ScanError
		}

		list = append(list, &encounter)
	}

	return &list, nil
}

func (s *encounterStorage) GetEncounterByID(ctx context.Context, id int) (*models.Encounter, error) {
	line := s.pool.QueryRow(ctx, GetEncounterByIDQuery, id)
	var encounter models.Encounter

	if err := line.Scan(&encounter.ID, &encounter.UserID, &encounter.Name, &encounter.Data); err != nil {
		return nil, apperrors.ScanError
	}

	return &encounter, nil
}
