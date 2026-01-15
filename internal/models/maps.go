package models

import "time"

// Placement represents a tile placement on the map
type Placement struct {
	ID     string `json:"id"`
	TileID string `json:"tileId"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Rot    int    `json:"rot"`
	Layer  int    `json:"layer"`
}

// MapData represents the map content stored as JSONB
type MapData struct {
	SchemaVersion int         `json:"schemaVersion"`
	WidthUnits    int         `json:"widthUnits"`
	HeightUnits   int         `json:"heightUnits"`
	Placements    []Placement `json:"placements"`
}

// MapFull represents the complete map with all data
type MapFull struct {
	ID        string    `json:"id"`
	UserID    int       `json:"userId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Data      MapData   `json:"data"`
}

// MapMetadata represents map metadata without full data (for list endpoint)
type MapMetadata struct {
	ID        string    `json:"id"`
	UserID    int       `json:"userId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MapsList represents paginated list of maps
type MapsList struct {
	Maps  []MapMetadata `json:"maps"`
	Total int           `json:"total"`
}

// CreateMapRequest represents the request to create a map
type CreateMapRequest struct {
	Name string  `json:"name"`
	Data MapData `json:"data"`
}

// UpdateMapRequest represents the request to update a map
type UpdateMapRequest struct {
	Name string  `json:"name"`
	Data MapData `json:"data"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// MapsErrorResponse represents an error response with details
type MapsErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details []ValidationError `json:"details,omitempty"`
}
