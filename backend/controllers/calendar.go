package controllers

import (
	"github.com/gin-gonic/gin"

	"mibondi.github.com/gtfs"
)

func registerCalendar(group *gin.RouterGroup, data *gtfs.Data) {
	h := &calendarHandler{data: data}
	group.GET("/calendar", h.list)
}

type calendarHandler struct {
	data *gtfs.Data
}

// list returns the raw calendar slice. The proposal did not commit to
// filtering to "currently active" services, so the loader's whole-file
// output is served verbatim. CalendarDates is omitted from the response on
// purpose — it is reference data, not part of the v1 contract.
func (h *calendarHandler) list(c *gin.Context) {
	ok(c, h.data.Calendar)
}
