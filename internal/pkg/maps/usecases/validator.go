package usecases

import (
	"fmt"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

const (
	MinNameLength      = 1
	MaxNameLength      = 255
	RequiredSchemaV    = 1
	MinRotation        = 0
	MaxRotation        = 3
	MapUnitsPerTile    = 6 // Each tile occupies 6x6 units
)

// ValidateMapRequest validates a map creation/update request and returns validation errors
func ValidateMapRequest(name string, data *models.MapData) []models.ValidationError {
	var errors []models.ValidationError

	// Validate name
	if len(name) < MinNameLength || len(name) > MaxNameLength {
		errors = append(errors, models.ValidationError{
			Field:   "name",
			Message: fmt.Sprintf("name must be between %d and %d characters", MinNameLength, MaxNameLength),
		})
	}

	// Validate schema version
	if data.SchemaVersion != RequiredSchemaV {
		errors = append(errors, models.ValidationError{
			Field:   "data.schemaVersion",
			Message: fmt.Sprintf("schemaVersion must be %d", RequiredSchemaV),
		})
	}

	// Validate dimensions
	if data.WidthUnits <= 0 {
		errors = append(errors, models.ValidationError{
			Field:   "data.widthUnits",
			Message: "widthUnits must be a positive integer",
		})
	} else if data.WidthUnits%MapUnitsPerTile != 0 {
		errors = append(errors, models.ValidationError{
			Field:   "data.widthUnits",
			Message: fmt.Sprintf("widthUnits must be a multiple of %d", MapUnitsPerTile),
		})
	}

	if data.HeightUnits <= 0 {
		errors = append(errors, models.ValidationError{
			Field:   "data.heightUnits",
			Message: "heightUnits must be a positive integer",
		})
	} else if data.HeightUnits%MapUnitsPerTile != 0 {
		errors = append(errors, models.ValidationError{
			Field:   "data.heightUnits",
			Message: fmt.Sprintf("heightUnits must be a multiple of %d", MapUnitsPerTile),
		})
	}

	// Validate placements
	for i, placement := range data.Placements {
		placementErrors := validatePlacement(i, &placement)
		errors = append(errors, placementErrors...)
	}

	return errors
}

func validatePlacement(index int, p *models.Placement) []models.ValidationError {
	var errors []models.ValidationError
	prefix := fmt.Sprintf("data.placements[%d]", index)

	if p.ID == "" {
		errors = append(errors, models.ValidationError{
			Field:   prefix + ".id",
			Message: "placement id cannot be empty",
		})
	}

	if p.TileID == "" {
		errors = append(errors, models.ValidationError{
			Field:   prefix + ".tileId",
			Message: "placement tileId cannot be empty",
		})
	}

	if p.X < 0 {
		errors = append(errors, models.ValidationError{
			Field:   prefix + ".x",
			Message: "placement x must be >= 0",
		})
	} else if p.X%MapUnitsPerTile != 0 {
		errors = append(errors, models.ValidationError{
			Field:   prefix + ".x",
			Message: fmt.Sprintf("placement x must be a multiple of %d", MapUnitsPerTile),
		})
	}

	if p.Y < 0 {
		errors = append(errors, models.ValidationError{
			Field:   prefix + ".y",
			Message: "placement y must be >= 0",
		})
	} else if p.Y%MapUnitsPerTile != 0 {
		errors = append(errors, models.ValidationError{
			Field:   prefix + ".y",
			Message: fmt.Sprintf("placement y must be a multiple of %d", MapUnitsPerTile),
		})
	}

	if p.Rot < MinRotation || p.Rot > MaxRotation {
		errors = append(errors, models.ValidationError{
			Field:   prefix + ".rot",
			Message: fmt.Sprintf("placement rot must be between %d and %d", MinRotation, MaxRotation),
		})
	}

	return errors
}

// CategorizeValidationErrors returns the primary error code based on validation errors
func CategorizeValidationErrors(errors []models.ValidationError) string {
	for _, err := range errors {
		switch err.Field {
		case "data.schemaVersion":
			return "INVALID_SCHEMA_VERSION"
		case "data.widthUnits", "data.heightUnits":
			return "INVALID_DIMENSIONS"
		}
		// Check for placement errors
		if len(err.Field) > 15 && err.Field[:15] == "data.placements" {
			return "INVALID_PLACEMENT"
		}
	}
	return "BAD_REQUEST"
}
