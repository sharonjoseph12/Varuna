package engine

import "math"

// Grid implements a fixed-size lat/lon spatial index for O(1) zone lookup.
type Grid struct {
	cellSize float64                // degrees per cell
	cells    map[CellID][]*Zone     // cell → zones overlapping that cell
}

// NewGrid builds the spatial index from zone polygons.
func NewGrid(cellSize float64, zones []Zone) *Grid {
	g := &Grid{
		cellSize: cellSize,
		cells:    make(map[CellID][]*Zone),
	}
	for i := range zones {
		zones[i].GridCells = g.computeOverlappingCells(&zones[i])
		for _, c := range zones[i].GridCells {
			g.cells[c] = append(g.cells[c], &zones[i])
		}
	}
	return g
}

// CellFor returns the grid cell containing the given lat/lon.
func (g *Grid) CellFor(lat, lon float64) CellID {
	return CellID{
		LatCell: int(math.Floor(lat / g.cellSize)),
		LonCell: int(math.Floor(lon / g.cellSize)),
	}
}

// ZonesAt returns all zones whose precomputed cells overlap the cell containing (lat, lon).
func (g *Grid) ZonesAt(lat, lon float64) []*Zone {
	return g.cells[g.CellFor(lat, lon)]
}

// computeOverlappingCells returns all grid cells that a zone's bounding box overlaps.
func (g *Grid) computeOverlappingCells(z *Zone) []CellID {
	minLat, maxLat := z.Polygon[0][0], z.Polygon[0][0]
	minLon, maxLon := z.Polygon[0][1], z.Polygon[0][1]
	for _, v := range z.Polygon[1:] {
		if v[0] < minLat { minLat = v[0] }
		if v[0] > maxLat { maxLat = v[0] }
		if v[1] < minLon { minLon = v[1] }
		if v[1] > maxLon { maxLon = v[1] }
	}
	// Expand by hysteresis margin for boundary cells
	minLat -= z.HysteresisMarginDeg
	maxLat += z.HysteresisMarginDeg
	minLon -= z.HysteresisMarginDeg
	maxLon += z.HysteresisMarginDeg

	latLo := int(math.Floor(minLat / g.cellSize))
	latHi := int(math.Floor(maxLat / g.cellSize))
	lonLo := int(math.Floor(minLon / g.cellSize))
	lonHi := int(math.Floor(maxLon / g.cellSize))

	var cells []CellID
	for la := latLo; la <= latHi; la++ {
		for lo := lonLo; lo <= lonHi; lo++ {
			cells = append(cells, CellID{la, lo})
		}
	}
	return cells
}

// PointInPolygon uses the ray-cast algorithm to test if (lat, lon) is inside a polygon.
// ponytail: ~20 lines, stdlib only, correct on edge cases
func PointInPolygon(lat, lon float64, polygon [][2]float64) bool {
	n := len(polygon)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		yi, xi := polygon[i][0], polygon[i][1]
		yj, xj := polygon[j][0], polygon[j][1]
		if ((yi > lat) != (yj > lat)) &&
			(lon < (xj-xi)*(lat-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}

// DistToPolygonEdgeDeg returns the approximate minimum distance in degrees
// from a point to the nearest edge of a polygon.
// ponytail: approximate using point-to-segment, good enough for hysteresis
func DistToPolygonEdgeDeg(lat, lon float64, polygon [][2]float64) float64 {
	minDist := math.MaxFloat64
	n := len(polygon)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		d := pointToSegmentDist(lat, lon, polygon[i][0], polygon[i][1], polygon[j][0], polygon[j][1])
		if d < minDist {
			minDist = d
		}
	}
	return minDist
}

func pointToSegmentDist(px, py, ax, ay, bx, by float64) float64 {
	dx := bx - ax
	dy := by - ay
	if dx == 0 && dy == 0 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy)
	if t < 0 { t = 0 }
	if t > 1 { t = 1 }
	nx := ax + t*dx
	ny := ay + t*dy
	return math.Hypot(px-nx, py-ny)
}
