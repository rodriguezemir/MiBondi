package controllers

import (
	"github.com/gin-gonic/gin"

	"mibondi.github.com/gtfs"
)

// shapePointDTO is the trimmed shape payload returned by /lines/:id/shape.
// Keeping it as a separate type (instead of leaking ShapePoint) lets us add
// more fields later (e.g. distance) without breaking the wire format.
type shapePointDTO struct {
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Sequence int     `json:"seq"`
}

type shapeResponse struct {
	ID     string         `json:"id"`
	Points []shapePointDTO `json:"points"`
}

func registerLines(group *gin.RouterGroup, data *gtfs.Data) {
	h := &linesHandler{data: data}
	group.GET("/lines", h.list)
	group.GET("/lines/:id", h.get)
	group.GET("/lines/:id/trips", h.trips)
	group.GET("/lines/:id/shape", h.shape)
}

type linesHandler struct {
	data *gtfs.Data
}

func (h *linesHandler) list(c *gin.Context) {
	ok(c, h.data.Routes)
}

func (h *linesHandler) get(c *gin.Context) {
	id := c.Param("id")
	r, ok2 := h.data.RouteByID[id]
	if !ok2 {
		respondNotFound(c, "line "+id+" not found")
		return
	}
	ok(c, r)
}

// trips returns every trip that belongs to the given route. The slice is
// built once at load time (TripsByRoute) so this is a single map lookup.
func (h *linesHandler) trips(c *gin.Context) {
	id := c.Param("id")
	if _, exists := h.data.RouteByID[id]; !exists {
		respondNotFound(c, "line "+id+" not found")
		return
	}
	trips := h.data.TripsByRoute[id]
	if trips == nil {
		// Route exists but has no trips — return an empty array, not 404.
		ok(c, []*gtfs.Trip{})
		return
	}
	ok(c, trips)
}

// shape returns the ordered point list for a line's geometry. The shape
// id is taken from the first trip of the line (GTFS guarantees that trips
// of the same route share a shape_id, but be defensive: 404 if no trip
// carries a shape). Points come pre-sorted by Sequence from the loader.
func (h *linesHandler) shape(c *gin.Context) {
	id := c.Param("id")
	trips := h.data.TripsByRoute[id]
	if len(trips) == 0 {
		// Distinguish "line does not exist" from "line exists but has no shape".
		if _, exists := h.data.RouteByID[id]; !exists {
			respondNotFound(c, "line "+id+" not found")
		} else {
			respondNotFound(c, "line "+id+" has no shape data")
		}
		return
	}

	shapeID := trips[0].ShapeID
	if shapeID == "" {
		respondNotFound(c, "line "+id+" has no shape data")
		return
	}
	pts := h.data.ShapeByID[shapeID]
	if len(pts) == 0 {
		respondNotFound(c, "shape for line "+id+" not found")
		return
	}

	dto := shapeResponse{ID: shapeID, Points: make([]shapePointDTO, len(pts))}
	for i, p := range pts {
		dto.Points[i] = shapePointDTO{Lat: p.Lat, Lon: p.Lon, Sequence: p.Sequence}
	}
	ok(c, dto)
}
