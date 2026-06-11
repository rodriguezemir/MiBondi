// Package gtfs provides domain types and a CSV loader for the General Transit
// Feed Specification (GTFS) data files used by the transport API.
//
// The Loader reads every required file from a directory and produces a single
// *Data value containing raw slices plus O(1) lookup indexes built post-load.
package gtfs

// Agency represents a transit agency from agency.txt.
type Agency struct {
	ID       string `json:"agency_id"`
	Name     string `json:"agency_name"`
	URL      string `json:"agency_url"`
	Timezone string `json:"agency_timezone"`
}

// Route represents a transit route (a "line") from routes.txt.
type Route struct {
	ID        string `json:"route_id"`
	AgencyID  string `json:"agency_id"`
	ShortName string `json:"route_short_name"`
	LongName  string `json:"route_long_name"`
	Type      int    `json:"route_type"`
	Color     string `json:"route_color"`
	TextColor string `json:"route_text_color"`
}

// Stop represents a physical stop/platform from stops.txt.
type Stop struct {
	ID   string  `json:"stop_id"`
	Name string  `json:"stop_name"`
	Lat  float64 `json:"stop_lat"`
	Lon  float64 `json:"stop_lon"`
}

// Trip represents a scheduled journey from trips.txt.
type Trip struct {
	ID        string `json:"trip_id"`
	RouteID   string `json:"route_id"`
	ServiceID string `json:"service_id"`
	ShapeID   string `json:"shape_id"`
	Headsign  string `json:"trip_headsign"`
	Direction int    `json:"direction_id"`
}

// StopTime represents one row of stop_times.txt — a trip arrival/departure
// at a particular stop.
type StopTime struct {
	TripID        string `json:"trip_id"`
	StopID        string `json:"stop_id"`
	ArrivalTime   string `json:"arrival_time"`
	DepartureTime string `json:"departure_time"`
	Sequence      int    `json:"stop_sequence"`
}

// ShapePoint represents a single point in shapes.txt. Shape points are
// stored per shape_id in ShapeByID, ordered by Sequence ascending.
type ShapePoint struct {
	ShapeID  string  `json:"shape_id"`
	Lat      float64 `json:"shape_pt_lat"`
	Lon      float64 `json:"shape_pt_lon"`
	Sequence int     `json:"shape_pt_sequence"`
}

// Calendar represents one row of calendar.txt — defines which days of the
// week a service runs.
type Calendar struct {
	ServiceID string `json:"service_id"`
	Monday    bool   `json:"monday"`
	Tuesday   bool   `json:"tuesday"`
	Wednesday bool   `json:"wednesday"`
	Thursday  bool   `json:"thursday"`
	Friday    bool   `json:"friday"`
	Saturday  bool   `json:"saturday"`
	Sunday    bool   `json:"sunday"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// CalendarDate represents an exception row from calendar_dates.txt.
type CalendarDate struct {
	ServiceID    string `json:"service_id"`
	Date         string `json:"date"`
	ExceptionType int   `json:"exception_type"`
}

// Data is the in-memory representation of the loaded GTFS feed. It exposes
// the raw slices (preserving order from disk) and the index maps that give
// handlers O(1) lookups.
type Data struct {
	Agencies      []Agency
	Routes        []Route
	Stops         []Stop
	Trips         []Trip
	StopTimes     []StopTime
	Shapes        []ShapePoint
	Calendar      []Calendar
	CalendarDates []CalendarDate

	// Indexes — built once after loading. Pointers avoid copying large
	// structs on every request.
	RouteByID    map[string]*Route
	StopByID     map[string]*Stop
	TripByID     map[string]*Trip
	TripsByRoute map[string][]*Trip
	TimesByStop  map[string][]*StopTime
	TimesByTrip  map[string][]*StopTime
	ShapeByID    map[string][]ShapePoint
	CalByService map[string]*Calendar
}
