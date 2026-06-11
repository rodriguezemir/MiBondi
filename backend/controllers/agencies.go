package controllers

import (
	"github.com/gin-gonic/gin"

	"mibondi.github.com/gtfs"
)

func registerAgencies(group *gin.RouterGroup, data *gtfs.Data) {
	h := &agenciesHandler{data: data}
	group.GET("/agencies", h.list)
	group.GET("/agencies/:id", h.get)
}

type agenciesHandler struct {
	data *gtfs.Data
}

// list returns every loaded agency. Agencies are small (handful per city)
// so we serve the full slice with no pagination.
func (h *agenciesHandler) list(c *gin.Context) {
	ok(c, h.data.Agencies)
}

// get returns a single agency by id. A miss returns 404 with the spec
// error envelope.
func (h *agenciesHandler) get(c *gin.Context) {
	id := c.Param("id")
	for i := range h.data.Agencies {
		if h.data.Agencies[i].ID == id {
			ok(c, h.data.Agencies[i])
			return
		}
	}
	respondNotFound(c, "agency "+id+" not found")
}
