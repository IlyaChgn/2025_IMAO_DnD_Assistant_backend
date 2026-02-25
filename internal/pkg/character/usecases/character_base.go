package usecases

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/apperrors"
	characterinterfaces "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/compute"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/converter"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/logger"
)

const maxAvatarSize = 500 * 1024 // 500 KB

type characterBaseUsecases struct {
	repo      characterinterfaces.CharacterBaseRepository
	s3Manager characterinterfaces.AvatarS3Manager
}

func NewCharacterBaseUsecases(repo characterinterfaces.CharacterBaseRepository, s3Manager characterinterfaces.AvatarS3Manager) characterinterfaces.CharacterBaseUsecases {
	return &characterBaseUsecases{repo: repo, s3Manager: s3Manager}
}

func (uc *characterBaseUsecases) Create(ctx context.Context, char *models.CharacterBase) error {
	return uc.repo.Create(ctx, char)
}

func (uc *characterBaseUsecases) GetByID(ctx context.Context, id string, userID int) (*models.CharacterBase, error) {
	l := logger.FromContext(ctx)

	if id == "" {
		l.UsecasesWarn(apperrors.InvalidInputError, userID, nil)
		return nil, apperrors.InvalidInputError
	}

	char, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return nil, err
	}

	if char == nil {
		return nil, nil
	}

	// Check ownership
	if char.UserID != strconv.Itoa(userID) {
		l.UsecasesWarn(apperrors.PermissionDeniedError, userID, map[string]any{"id": id})
		return nil, apperrors.PermissionDeniedError
	}

	return char, nil
}

func (uc *characterBaseUsecases) GetComputed(ctx context.Context, id string, userID int) (*models.CharacterBase, *models.DerivedStats, error) {
	char, err := uc.GetByID(ctx, id, userID)
	if err != nil {
		return nil, nil, err
	}
	if char == nil {
		return nil, nil, nil
	}
	derived := compute.ComputeDerived(char)
	return char, derived, nil
}

func (uc *characterBaseUsecases) Update(ctx context.Context, char *models.CharacterBase, expectedVersion int, userID int) error {
	// Ownership is enforced atomically: the handler sets char.UserID from the
	// authenticated user, and the repository includes userId in the MongoDB filter.
	// No separate GetByID check needed — eliminates TOCTOU race.
	return uc.repo.Update(ctx, char, expectedVersion)
}

func (uc *characterBaseUsecases) Delete(ctx context.Context, id string, userID int) error {
	// Ownership is enforced atomically via userId in the MongoDB delete filter.
	return uc.repo.Delete(ctx, id, strconv.Itoa(userID))
}

func (uc *characterBaseUsecases) List(ctx context.Context, userID int, page, size int,
	search string) ([]*models.CharacterBase, int64, error) {
	l := logger.FromContext(ctx)

	if page < 0 || size <= 0 {
		l.UsecasesWarn(apperrors.StartPosSizeError, userID, map[string]any{"page": page, "size": size})
		return nil, 0, apperrors.StartPosSizeError
	}

	return uc.repo.List(ctx, strconv.Itoa(userID), page, size, search)
}

func (uc *characterBaseUsecases) ImportLSS(ctx context.Context, fileData []byte,
	userID int) (*models.CharacterBase, *models.ConversionReport, error) {
	l := logger.FromContext(ctx)

	char, report, err := converter.ConvertLSS(fileData, strconv.Itoa(userID))
	if err != nil {
		l.UsecasesError(err, userID, nil)
		return nil, nil, apperrors.ConversionFailedError
	}

	if err := uc.repo.Create(ctx, char); err != nil {
		l.UsecasesError(err, userID, map[string]any{"charName": char.Name})
		return nil, nil, err
	}

	return char, report, nil
}

func (uc *characterBaseUsecases) UploadAvatar(ctx context.Context, id string, userID int, fileData []byte) (string, error) {
	l := logger.FromContext(ctx)

	if len(fileData) > maxAvatarSize {
		l.UsecasesWarn(apperrors.AvatarTooLargeErr, userID, map[string]any{"size": len(fileData)})
		return "", apperrors.AvatarTooLargeErr
	}

	// Fetch character to validate existence/ownership and get old avatar URL for cleanup.
	char, err := uc.GetByID(ctx, id, userID)
	if err != nil {
		return "", err
	}
	if char == nil {
		return "", apperrors.CharacterNotFoundErr
	}

	objectName := fmt.Sprintf("%s-%d.webp", id, time.Now().UnixMilli())

	avatarURL, err := uc.s3Manager.UploadAvatar(ctx, fileData, objectName)
	if err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		return "", apperrors.AvatarUploadErr
	}

	// Ownership is enforced atomically via userId in the MongoDB update filter.
	if err := uc.repo.UpdateAvatarURL(ctx, id, strconv.Itoa(userID), avatarURL); err != nil {
		l.UsecasesError(err, userID, map[string]any{"id": id})
		// Best-effort cleanup: delete the just-uploaded S3 object to avoid orphan.
		if delErr := uc.s3Manager.DeleteAvatar(ctx, objectName); delErr != nil {
			l.UsecasesError(delErr, userID, map[string]any{"id": id, "orphanedObject": objectName})
		}
		return "", err
	}

	// Best-effort cleanup: delete old S3 avatar object if it existed.
	if char.Avatar != nil && char.Avatar.Url != "" {
		parts := strings.Split(char.Avatar.Url, "/")
		if len(parts) > 0 {
			oldObjectName := parts[len(parts)-1]
			if delErr := uc.s3Manager.DeleteAvatar(ctx, oldObjectName); delErr != nil {
				l.UsecasesError(delErr, userID, map[string]any{"id": id, "oldObject": oldObjectName})
				// Non-fatal: old object leaked but new avatar is live.
			}
		}
	}

	return avatarURL, nil
}

func (uc *characterBaseUsecases) DeleteAvatar(ctx context.Context, id string, userID int) error {
	l := logger.FromContext(ctx)

	// Need to fetch character to get the current avatar URL for S3 deletion.
	char, err := uc.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}
	if char == nil {
		return apperrors.CharacterNotFoundErr
	}

	// Delete from S3 if avatar URL exists
	if char.Avatar != nil && char.Avatar.Url != "" {
		// Extract object name from URL: https://encounterium.ru/character-avatars/{objectName}
		parts := strings.Split(char.Avatar.Url, "/")
		if len(parts) > 0 {
			objectName := parts[len(parts)-1]
			if err := uc.s3Manager.DeleteAvatar(ctx, objectName); err != nil {
				l.UsecasesError(err, userID, map[string]any{"id": id, "objectName": objectName})
				// Continue to clear DB even if S3 delete fails — avoid orphaned DB state
			}
		}
	}

	// Ownership is enforced atomically via userId in the MongoDB update filter.
	return uc.repo.ClearAvatar(ctx, id, strconv.Itoa(userID))
}

func (uc *characterBaseUsecases) GetHotbarLayout(ctx context.Context, id string, userID int) (*models.HotbarLayout, error) {
	l := logger.FromContext(ctx)

	if id == "" {
		l.UsecasesWarn(apperrors.InvalidInputError, userID, nil)
		return nil, apperrors.InvalidInputError
	}

	// Ownership is enforced atomically via userId in the MongoDB query filter.
	return uc.repo.GetHotbarLayout(ctx, id, strconv.Itoa(userID))
}

func (uc *characterBaseUsecases) SetHotbarLayout(ctx context.Context, id string, userID int, layout *models.HotbarLayout) error {
	l := logger.FromContext(ctx)

	if id == "" {
		l.UsecasesWarn(apperrors.InvalidInputError, userID, nil)
		return apperrors.InvalidInputError
	}

	if layout.RowCount < 2 || layout.RowCount > 5 {
		l.UsecasesWarn(apperrors.InvalidInputError, userID, map[string]any{"rowCount": layout.RowCount})
		return apperrors.InvalidInputError
	}

	// Ownership is enforced atomically via userId in the MongoDB update filter.
	return uc.repo.SetHotbarLayout(ctx, id, strconv.Itoa(userID), layout)
}
