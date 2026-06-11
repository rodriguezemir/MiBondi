// Package controllers wires the GTFS-backed HTTP handlers for the transport
// API. Every handler takes a *gtfs.Data (no globals) so the package stays
// unit-testable and the data layer can be swapped without touching Gin.
package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mibondi.github.com/gtfs"
)

// successEnvelope is the JSON shape for every 2xx response. Keeping a typed
// alias lets the spec's `{ "data": ... }` contract stay clear at the call
// site without leaking gin.H strings through the code.
type successEnvelope struct {
	Data any `json:"data"`
}

// errorEnvelope matches the spec error contract: { "error": { "code", "message" } }.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RegisterRoutes mounts every v1 endpoint on r, sharing the supplied *gtfs.Data.
// /ping is intentionally not registered here so liveness probes can stay
// independent of the transport data load.
func RegisterRoutes(r *gin.Engine, data *gtfs.Data) {
	v1 := r.Group("/api/v1")

	registerAgencies(v1, data)
	registerLines(v1, data)
	registerStops(v1, data)
	registerTrips(v1, data)
	registerCalendar(v1, data)
}

// respondNotFound aborts the request with a 404 + the spec's error envelope.
// Used by every handler when an :id lookup misses, so the format stays
// consistent across the whole API.
func respondNotFound(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusNotFound, errorEnvelope{
		Error: errorBody{Code: "NOT_FOUND", Message: message},
	})
}

// ok wraps the supplied payload in the success envelope and writes a 200.
func ok(c *gin.Context, payload any) {
	c.JSON(http.StatusOK, successEnvelope{Data: payload})
}
