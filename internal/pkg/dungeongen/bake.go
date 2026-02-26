package dungeongen

import (
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
)

// BakedTerrain is the flattened terrain data produced by stamping all tile
// placements into global walkability/occlusion grids and merging edges.
type BakedTerrain struct {
	Rows        int                      `json:"rows"`
	Cols        int                      `json:"cols"`
	Walkability [][]int                  `json:"walkability"`
	Occlusion   [][]int                  `json:"occlusion"`
	Edges       []models.SerializedEdge  `json:"edges"`
}

// BakeTerrain stamps all tile placements from a MapComposition into flat
// walkability/occlusion grids and collects edges with global coordinates.
func BakeTerrain(comp *MapComposition, tileData map[string]*models.TileWalkability) *BakedTerrain {
	// Step 1: Create flat grids (uncovered = blocked/opaque)
	walkability := makeFilledGrid(comp.Rows, comp.Cols, 0)
	occlusion := makeFilledGrid(comp.Rows, comp.Cols, 1)

	// Edge accumulator: global key → merged properties
	edgeMap := make(map[string]models.SerializedEdge)

	// Step 2: Stamp each placement
	for _, p := range comp.Placements {
		tile, ok := tileData[p.TileID]
		if !ok {
			log.Printf("BakeTerrain: tile %s not found, skipping placement %s", p.TileID, p.NodeID)
			continue
		}

		// Rotate tile grids
		rotWalk := RotateGrid(tile.Walkability, p.Rotation)
		rotOccl := RotateGrid(tile.Occlusion, p.Rotation)

		// Stamp into flat grids
		stampTile(walkability, rotWalk, p.OriginRow, p.OriginCol)
		stampTile(occlusion, rotOccl, p.OriginRow, p.OriginCol)

		// Step 3: Collect and translate edges
		if len(tile.Edges) > 0 {
			rotEdges := RotateEdges(tile.Edges, p.Rotation, TileSize)
			for _, e := range rotEdges {
				globalKey := translateEdgeKey(e.Key, p.OriginRow, p.OriginCol)
				if existing, exists := edgeMap[globalKey]; exists {
					// Block-wins merge
					edgeMap[globalKey] = models.SerializedEdge{
						Key:       globalKey,
						MoveBlock: existing.MoveBlock || e.MoveBlock,
						LosBlock:  existing.LosBlock || e.LosBlock,
					}
				} else {
					edgeMap[globalKey] = models.SerializedEdge{
						Key:       globalKey,
						MoveBlock: e.MoveBlock,
						LosBlock:  e.LosBlock,
					}
				}
			}
		}
	}

	// Step 4: Convert edge map to sorted slice
	edges := make([]models.SerializedEdge, 0, len(edgeMap))
	for _, e := range edgeMap {
		edges = append(edges, e)
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Key < edges[j].Key
	})

	return &BakedTerrain{
		Rows:        comp.Rows,
		Cols:        comp.Cols,
		Walkability: walkability,
		Occlusion:   occlusion,
		Edges:       edges,
	}
}

// makeFilledGrid creates a rows×cols grid filled with the given value.
func makeFilledGrid(rows, cols, fill int) [][]int {
	grid := make([][]int, rows)
	for r := range grid {
		grid[r] = make([]int, cols)
		if fill != 0 {
			for c := range grid[r] {
				grid[r][c] = fill
			}
		}
	}
	return grid
}

// stampTile overwrites cells in dst with values from src at the given origin.
func stampTile(dst [][]int, src [][]int, originRow, originCol int) {
	for r := 0; r < len(src); r++ {
		globalRow := originRow + r
		if globalRow < 0 || globalRow >= len(dst) {
			continue
		}
		for c := 0; c < len(src[r]); c++ {
			globalCol := originCol + c
			if globalCol < 0 || globalCol >= len(dst[globalRow]) {
				continue
			}
			dst[globalRow][globalCol] = src[r][c]
		}
	}
}

// translateEdgeKey shifts an edge key by (originRow, originCol).
// Parses "r1,c1-r2,c2", adds offsets, returns normalized result.
func translateEdgeKey(key string, originRow, originCol int) string {
	parts := strings.Split(key, "-")
	if len(parts) != 2 {
		return key
	}

	coords0 := strings.Split(parts[0], ",")
	coords1 := strings.Split(parts[1], ",")
	if len(coords0) != 2 || len(coords1) != 2 {
		return key
	}

	r1, err1 := strconv.Atoi(coords0[0])
	c1, err2 := strconv.Atoi(coords0[1])
	r2, err3 := strconv.Atoi(coords1[0])
	c2, err4 := strconv.Atoi(coords1[1])
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return key
	}

	r1 += originRow
	c1 += originCol
	r2 += originRow
	c2 += originCol

	return normalizedEdgeKey(r1, c1, r2, c2)
}

