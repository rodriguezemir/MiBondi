package controllers

import (
	"github.com/gin-gonic/gin"

	"mibondi.github.com/gtfs"
)

// tripStopDTO joins a stop with its stop_time entry for a single trip. The
// sequence is preserved from TimesByTrip, which the loader sorts ascending.
type tripStopDTO struct {
	Stop     *gtfs.Stop   `json:"stop"`
	StopTime *gtfs.StopTime `json:"stop_time"`
}

func registerTrips(group *gin.RouterGroup, data *gtfs.Data) {
	h := &tripsHandler{data: data}
	group.GET("/trips", h.list)
	group.GET("/trips/:id", h.get)
	group.GET("/trips/:id/stops", h.stops)
}

type tripsHandler struct {
	data *gtfs.Data
}

func (h *tripsHandler) list(c *gin.Context) {
	ok(c, h.data.Trips)
}

func (h *tripsHandler) get(c *gin.Context) {
	id := c.Param("id")
	t, exists := h.data.TripByID[id]
	if !exists {
		respondNotFound(c, "trip "+id+" not found")
		return
	}
	ok(c, t)
}

// stops returns the ordered list of stops a trip visits, each joined with
// its stop_time. The lookup misses in StopByID (a referenced stop that was
// not loaded) are skipped with a log so the trip sequence still renders.
func (h *tripsHandler) stops(c *gin.Context) {
	id := c.Param("id")
	if _, exists := h.data.TripByID[id]; !exists {
		respondNotFound(c, "trip "+id+" not found")
		return
	}

	times := h.data.TimesByTrip[id]
	out := make([]tripStopDTO, 0, len(times))
	for _, st := range times {
		stop, ok2 := h.data.StopByID[st.StopID]
		if !ok2 {
			// The stop_times row references a stop_id we did not load. Skip
			// rather than 404 the whole trip — partial sequence is more
			// useful than nothing for a frontend.
			continue
		}
		out = append(out, tripStopDTO{Stop: stop, StopTime: st})
	}
	ok(c, out)
}
