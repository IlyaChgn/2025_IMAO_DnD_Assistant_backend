package usecases

import (
	"strings"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestValidateMapRequest_ValidInput(t *testing.T) {
	data := &models.MapData{
		SchemaVersion: 1,
		WidthUnits:    78, // 13 tiles * 6 units
		HeightUnits:   60, // 10 tiles * 6 units
		Placements: []models.Placement{
			{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 0, Layer: 0},
			{ID: "cell:0:1", TileID: "stone", X: 6, Y: 0, Rot: 1, Layer: 1},
			{ID: "cell:1:0", TileID: "water", X: 0, Y: 6, Rot: 2, Layer: 0},
		},
	}

	errors := ValidateMapRequest("Test Map", data)
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(errors), errors)
	}
}

func TestValidateMapRequest_InvalidName(t *testing.T) {
	tests := []struct {
		name     string
		mapName  string
		expected string
	}{
		{
			name:     "empty name",
			mapName:  "",
			expected: "name",
		},
		{
			name:     "name too long",
			mapName:  strings.Repeat("a", 256),
			expected: "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    78, // 13 tiles * 6 units
				HeightUnits:   60, // 10 tiles * 6 units
				Placements:    []models.Placement{},
			}

			errors := ValidateMapRequest(tt.mapName, data)
			if len(errors) == 0 {
				t.Error("Expected validation error for invalid name")
				return
			}

			found := false
			for _, err := range errors {
				if err.Field == tt.expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected error for field '%s', got: %v", tt.expected, errors)
			}
		})
	}
}

func TestValidateMapRequest_InvalidSchemaVersion(t *testing.T) {
	tests := []struct {
		name    string
		version int
	}{
		{"version 0", 0},
		{"version 2", 2},
		{"negative version", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &models.MapData{
				SchemaVersion: tt.version,
				WidthUnits:    78, // 13 tiles * 6 units
				HeightUnits:   60, // 10 tiles * 6 units
				Placements:    []models.Placement{},
			}

			errors := ValidateMapRequest("Test Map", data)
			if len(errors) == 0 {
				t.Error("Expected validation error for invalid schema version")
				return
			}

			found := false
			for _, err := range errors {
				if err.Field == "data.schemaVersion" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected error for field 'data.schemaVersion', got: %v", errors)
			}
		})
	}
}

func TestValidateMapRequest_InvalidDimensions(t *testing.T) {
	tests := []struct {
		name         string
		width        int
		height       int
		expectWidth  bool
		expectHeight bool
	}{
		{"zero width", 0, 60, true, false},
		{"negative width", -10, 60, true, false},
		{"zero height", 78, 0, false, true},
		{"negative height", 78, -10, false, true},
		{"both invalid", 0, 0, true, true},
		{"width not multiple of 6", 77, 60, true, false},
		{"height not multiple of 6", 78, 61, false, true},
		{"both not multiple of 6", 77, 61, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    tt.width,
				HeightUnits:   tt.height,
				Placements:    []models.Placement{},
			}

			errors := ValidateMapRequest("Test Map", data)
			if len(errors) == 0 {
				t.Error("Expected validation error for invalid dimensions")
				return
			}

			widthFound := false
			heightFound := false
			for _, err := range errors {
				if err.Field == "data.widthUnits" {
					widthFound = true
				}
				if err.Field == "data.heightUnits" {
					heightFound = true
				}
			}

			if tt.expectWidth && !widthFound {
				t.Errorf("Expected error for field 'data.widthUnits', got: %v", errors)
			}
			if tt.expectHeight && !heightFound {
				t.Errorf("Expected error for field 'data.heightUnits', got: %v", errors)
			}
		})
	}
}

func TestValidateMapRequest_InvalidPlacements(t *testing.T) {
	tests := []struct {
		name      string
		placement models.Placement
		expected  string
	}{
		{
			name:      "empty id",
			placement: models.Placement{ID: "", TileID: "grass", X: 0, Y: 0, Rot: 0},
			expected:  "data.placements[0].id",
		},
		{
			name:      "empty tileId",
			placement: models.Placement{ID: "cell:0:0", TileID: "", X: 0, Y: 0, Rot: 0},
			expected:  "data.placements[0].tileId",
		},
		{
			name:      "negative x",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: -1, Y: 0, Rot: 0},
			expected:  "data.placements[0].x",
		},
		{
			name:      "negative y",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: 0, Y: -5, Rot: 0},
			expected:  "data.placements[0].y",
		},
		{
			name:      "x not multiple of 6",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: 7, Y: 0, Rot: 0},
			expected:  "data.placements[0].x",
		},
		{
			name:      "y not multiple of 6",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: 0, Y: 5, Rot: 0},
			expected:  "data.placements[0].y",
		},
		{
			name:      "invalid rotation negative",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: -1},
			expected:  "data.placements[0].rot",
		},
		{
			name:      "invalid rotation too high",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 4},
			expected:  "data.placements[0].rot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    78, // 13 tiles * 6 units
				HeightUnits:   60, // 10 tiles * 6 units
				Placements:    []models.Placement{tt.placement},
			}

			errors := ValidateMapRequest("Test Map", data)
			if len(errors) == 0 {
				t.Error("Expected validation error for invalid placement")
				return
			}

			found := false
			for _, err := range errors {
				if err.Field == tt.expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected error for field '%s', got: %v", tt.expected, errors)
			}
		})
	}
}

func TestValidateMapRequest_MultiplePlacements(t *testing.T) {
	data := &models.MapData{
		SchemaVersion: 1,
		WidthUnits:    78, // 13 tiles * 6 units
		HeightUnits:   60, // 10 tiles * 6 units
		Placements: []models.Placement{
			{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 0},  // valid
			{ID: "", TileID: "stone", X: 6, Y: 0, Rot: 1},           // invalid id
			{ID: "cell:0:2", TileID: "", X: 12, Y: 0, Rot: 2},       // invalid tileId
		},
	}

	errors := ValidateMapRequest("Test Map", data)
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d: %v", len(errors), errors)
	}

	expectedFields := map[string]bool{
		"data.placements[1].id":     false,
		"data.placements[2].tileId": false,
	}

	for _, err := range errors {
		if _, ok := expectedFields[err.Field]; ok {
			expectedFields[err.Field] = true
		}
	}

	for field, found := range expectedFields {
		if !found {
			t.Errorf("Expected error for field '%s' not found", field)
		}
	}
}

func TestValidateMapRequest_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		mapName   string
		data      *models.MapData
		expectErr bool
	}{
		{
			name:    "minimum valid name length",
			mapName: "A",
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    6, // 1 tile * 6 units
				HeightUnits:   6, // 1 tile * 6 units
				Placements:    []models.Placement{},
			},
			expectErr: false,
		},
		{
			name:    "maximum valid name length",
			mapName: strings.Repeat("a", 255),
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    6, // 1 tile * 6 units
				HeightUnits:   6, // 1 tile * 6 units
				Placements:    []models.Placement{},
			},
			expectErr: false,
		},
		{
			name:    "valid rotation at boundary 0",
			mapName: "Test",
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    12, // 2 tiles * 6 units
				HeightUnits:   12, // 2 tiles * 6 units
				Placements: []models.Placement{
					{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 0},
				},
			},
			expectErr: false,
		},
		{
			name:    "valid rotation at boundary 3",
			mapName: "Test",
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    12, // 2 tiles * 6 units
				HeightUnits:   12, // 2 tiles * 6 units
				Placements: []models.Placement{
					{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 3},
				},
			},
			expectErr: false,
		},
		{
			name:    "empty placements array",
			mapName: "Test",
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    12, // 2 tiles * 6 units
				HeightUnits:   12, // 2 tiles * 6 units
				Placements:    []models.Placement{},
			},
			expectErr: false,
		},
		{
			name:    "placement at non-zero aligned position",
			mapName: "Test",
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    24, // 4 tiles * 6 units
				HeightUnits:   24, // 4 tiles * 6 units
				Placements: []models.Placement{
					{ID: "cell:1:1", TileID: "grass", X: 6, Y: 6, Rot: 0},
					{ID: "cell:2:2", TileID: "stone", X: 12, Y: 12, Rot: 1},
					{ID: "cell:3:0", TileID: "water", X: 18, Y: 0, Rot: 2},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateMapRequest(tt.mapName, tt.data)
			if tt.expectErr && len(errors) == 0 {
				t.Error("Expected validation errors but got none")
			}
			if !tt.expectErr && len(errors) > 0 {
				t.Errorf("Expected no errors but got: %v", errors)
			}
		})
	}
}

func TestCategorizeValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []models.ValidationError
		expected string
	}{
		{
			name: "schema version error",
			errors: []models.ValidationError{
				{Field: "data.schemaVersion", Message: "invalid"},
			},
			expected: "INVALID_SCHEMA_VERSION",
		},
		{
			name: "width dimension error",
			errors: []models.ValidationError{
				{Field: "data.widthUnits", Message: "invalid"},
			},
			expected: "INVALID_DIMENSIONS",
		},
		{
			name: "height dimension error",
			errors: []models.ValidationError{
				{Field: "data.heightUnits", Message: "invalid"},
			},
			expected: "INVALID_DIMENSIONS",
		},
		{
			name: "placement error",
			errors: []models.ValidationError{
				{Field: "data.placements[0].id", Message: "invalid"},
			},
			expected: "INVALID_PLACEMENT",
		},
		{
			name: "name error (default)",
			errors: []models.ValidationError{
				{Field: "name", Message: "invalid"},
			},
			expected: "BAD_REQUEST",
		},
		{
			name:     "empty errors",
			errors:   []models.ValidationError{},
			expected: "BAD_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategorizeValidationErrors(tt.errors)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
