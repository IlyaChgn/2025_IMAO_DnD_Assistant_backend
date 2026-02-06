package usecases

import (
	"strings"
	"testing"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

func TestValidateMapRequest_ValidInput(t *testing.T) {
	data := &models.MapData{
		SchemaVersion: 1,
		WidthUnits:    13,
		HeightUnits:   7,
		Placements: []models.Placement{
			{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 0, Layer: 0},
			{ID: "cell:0:1", TileID: "stone", X: 5, Y: 0, Rot: 1, Layer: 1},
			{ID: "cell:1:0", TileID: "water", X: 0, Y: 3, Rot: 2, Layer: 0},
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
				WidthUnits:    13,
				HeightUnits:   7,
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
				WidthUnits:    13,
				HeightUnits:   7,
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
		{"zero width", 0, 7, true, false},
		{"negative width", -10, 7, true, false},
		{"zero height", 13, 0, false, true},
		{"negative height", 13, -10, false, true},
		{"both invalid", 0, 0, true, true},
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
			name:      "invalid rotation negative",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: -1},
			expected:  "data.placements[0].rot",
		},
		{
			name:      "invalid rotation too high",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 4},
			expected:  "data.placements[0].rot",
		},
		{
			name:      "negative layer",
			placement: models.Placement{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 0, Layer: -1},
			expected:  "data.placements[0].layer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    13,
				HeightUnits:   7,
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
		WidthUnits:    13,
		HeightUnits:   7,
		Placements: []models.Placement{
			{ID: "cell:0:0", TileID: "grass", X: 0, Y: 0, Rot: 0}, // valid
			{ID: "", TileID: "stone", X: 5, Y: 0, Rot: 1},         // invalid id
			{ID: "cell:0:2", TileID: "", X: 9, Y: 0, Rot: 2},      // invalid tileId
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
				WidthUnits:    1,
				HeightUnits:   1,
				Placements:    []models.Placement{},
			},
			expectErr: false,
		},
		{
			name:    "maximum valid name length",
			mapName: strings.Repeat("a", 255),
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    1,
				HeightUnits:   1,
				Placements:    []models.Placement{},
			},
			expectErr: false,
		},
		{
			name:    "valid rotation at boundary 0",
			mapName: "Test",
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    10,
				HeightUnits:   10,
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
				WidthUnits:    10,
				HeightUnits:   10,
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
				WidthUnits:    10,
				HeightUnits:   10,
				Placements:    []models.Placement{},
			},
			expectErr: false,
		},
		{
			name:    "placement at non-aligned position",
			mapName: "Test",
			data: &models.MapData{
				SchemaVersion: 1,
				WidthUnits:    100,
				HeightUnits:   100,
				Placements: []models.Placement{
					{ID: "cell:1:1", TileID: "grass", X: 5, Y: 7, Rot: 0},
					{ID: "cell:2:2", TileID: "stone", X: 13, Y: 11, Rot: 1},
					{ID: "cell:3:0", TileID: "water", X: 17, Y: 0, Rot: 2},
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

// F.1 #1: No divisibility check — already covered by TestValidateMapRequest_ValidInput (widthUnits=13, heightUnits=7)

// F.1 #2: Layer negative — already covered by TestValidateMapRequest_InvalidPlacements "negative layer"

// F.1 #4
func TestValidateUpdateMapRequest_NameNil_OK(t *testing.T) {
	validData := &models.MapData{
		SchemaVersion: 1,
		WidthUnits:    13,
		HeightUnits:   7,
		Placements:    []models.Placement{},
	}

	errors := ValidateUpdateMapRequest(nil, validData)
	if len(errors) != 0 {
		t.Errorf("Expected no errors when name is nil, got %d: %v", len(errors), errors)
	}
}

// F.1 #5
func TestValidateUpdateMapRequest_NameEmpty_Error(t *testing.T) {
	validData := &models.MapData{
		SchemaVersion: 1,
		WidthUnits:    13,
		HeightUnits:   7,
		Placements:    []models.Placement{},
	}

	empty := ""
	errors := ValidateUpdateMapRequest(&empty, validData)
	if len(errors) == 0 {
		t.Error("Expected validation error for empty name")
		return
	}

	found := false
	for _, err := range errors {
		if err.Field == "name" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected error for field 'name', got: %v", errors)
	}
}

// F.1 #3: CategorizeValidationErrors name → INVALID_NAME — already covered below in TestCategorizeValidationErrors

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
			name: "name error returns INVALID_NAME",
			errors: []models.ValidationError{
				{Field: "name", Message: "too short"},
			},
			expected: "INVALID_NAME",
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
