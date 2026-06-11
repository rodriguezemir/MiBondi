// Command mibondi serves the Buenos Aires public transport API. GTFS data
// is loaded once at startup from GTFS_DATA_DIR (default: ../../resources
// relative to the backend/ working directory) and exposed read-only
// through /api/v1. /ping is kept as a liveness probe.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"mibondi.github.com/controllers"
	"mibondi.github.com/gtfs"
)

const defaultDataDir = "../../resources"

func main() {
	dir := os.Getenv("GTFS_DATA_DIR")
	if dir == "" {
		dir = defaultDataDir
	}

	log.Printf("loading GTFS feed from %q", dir)
	data, err := gtfs.Load(dir)
	if err != nil {
		log.Fatalf("failed to load GTFS data: %v", err)
	}
	log.Printf("loaded %d agencies, %d routes, %d stops, %d trips, %d stop_times, %d shape points, %d calendar entries",
		len(data.Agencies), len(data.Routes), len(data.Stops), len(data.Trips),
		len(data.StopTimes), len(data.Shapes), len(data.Calendar))

	r := gin.Default()

	// Liveness probe — independent of GTFS data so health checks stay green
	// even if a handler is being debugged.
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	controllers.RegisterRoutes(r, data)

	if err := r.Run(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
