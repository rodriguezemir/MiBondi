package controllers

import (
	"github.com/gin-gonic/gin"

	"mibondi.github.com/gtfs"
)

// stopTimeDTO is the trimmed schedule payload. Returning only the fields
// the frontend needs keeps the wire format small and stable.
type stopTimeDTO struct {
	TripID        string `json:"trip_id"`
	ArrivalTime   string `json:"arrival"`
	DepartureTime string `json:"departure"`
}

type stopScheduleResponse struct {
	Stop  *gtfs.Stop   `json:"stop"`
	Times []stopTimeDTO `json:"times"`
}

func registerStops(group *gin.RouterGroup, data *gtfs.Data) {
	h := &stopsHandler{data: data}
	group.GET("/stops", h.list)
	group.GET("/stops/:id", h.get)
	group.GET("/stops/:id/schedule", h.schedule)
}

type stopsHandler struct {
	data *gtfs.Data
}

func (h *stopsHandler) list(c *gin.Context) {
	ok(c, h.data.Stops)
}

func (h *stopsHandler) get(c *gin.Context) {
	id := c.Param("id")
	s, exists := h.data.StopByID[id]
	if !exists {
		respondNotFound(c, "stop "+id+" not found")
		return
	}
	ok(c, s)
}

// schedule returns every stop_time scheduled at this stop. The loader does
// not pre-sort TimesByStop — most consumers want a chronological arrival
// order, but stable order is not specified, so we preserve the on-disk
// order (which is grouped by trip, not by clock time) for now.
func (h *stopsHandler) schedule(c *gin.Context) {
	id := c.Param("id")
	stop, exists := h.data.StopByID[id]
	if !exists {
		respondNotFound(c, "stop "+id+" not found")
		return
	}
	raw := h.data.TimesByStop[id]
	times := make([]stopTimeDTO, len(raw))
	for i, st := range raw {
		times[i] = stopTimeDTO{
			TripID:        st.TripID,
			ArrivalTime:   st.ArrivalTime,
			DepartureTime: st.DepartureTime,
		}
	}
	ok(c, stopScheduleResponse{Stop: stop, Times: times})
}
