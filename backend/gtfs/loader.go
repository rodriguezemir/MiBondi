package gtfs

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// requiredFiles lists the GTFS files Load MUST find in the data directory.
// Missing required files cause Load to return a wrapped error.
var requiredFiles = []string{
	"agency.txt",
	"routes.txt",
	"stops.txt",
	"trips.txt",
	"stop_times.txt",
	"shapes.txt",
	"calendar.txt",
	"calendar_dates.txt",
}

// Load reads every GTFS CSV file from dir, parses the rows into typed
// structs, and returns a *Data with all index maps populated.
//
// Behavior:
//   - Missing required files return a wrapped error (caller should fatal).
//   - Malformed rows are logged with a warning and skipped; the rest of the
//     feed is loaded and partial data is served.
//   - Lookup indexes are built post-load. TimesByTrip and ShapeByID are
//     sorted by Sequence ascending so handlers can serve monotonic data.
func Load(dir string) (*Data, error) {
	if err := requireFiles(dir); err != nil {
		return nil, err
	}

	d := &Data{}

	// Parse in an order that mirrors file size ascending so early errors
	// surface quickly. The order itself does not affect correctness.
	if err := parseAgency(filepath.Join(dir, "agency.txt"), d); err != nil {
		return nil, fmt.Errorf("parse agency.txt: %w", err)
	}
	if err := parseCalendar(filepath.Join(dir, "calendar.txt"), d); err != nil {
		return nil, fmt.Errorf("parse calendar.txt: %w", err)
	}
	if err := parseCalendarDates(filepath.Join(dir, "calendar_dates.txt"), d); err != nil {
		return nil, fmt.Errorf("parse calendar_dates.txt: %w", err)
	}
	if err := parseRoutes(filepath.Join(dir, "routes.txt"), d); err != nil {
		return nil, fmt.Errorf("parse routes.txt: %w", err)
	}
	if err := parseStops(filepath.Join(dir, "stops.txt"), d); err != nil {
		return nil, fmt.Errorf("parse stops.txt: %w", err)
	}
	if err := parseTrips(filepath.Join(dir, "trips.txt"), d); err != nil {
		return nil, fmt.Errorf("parse trips.txt: %w", err)
	}
	if err := parseShapes(filepath.Join(dir, "shapes.txt"), d); err != nil {
		return nil, fmt.Errorf("parse shapes.txt: %w", err)
	}
	if err := parseStopTimes(filepath.Join(dir, "stop_times.txt"), d); err != nil {
		return nil, fmt.Errorf("parse stop_times.txt: %w", err)
	}

	buildIndexes(d)
	return d, nil
}

// requireFiles returns an error if any required GTFS file is missing.
func requireFiles(dir string) error {
	var missing []string
	for _, name := range requiredFiles {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("gtfs: missing required files in %q: %s", dir, strings.Join(missing, ", "))
	}
	return nil
}

// headerIndex returns the column index of name inside headers. The match is
// case-insensitive and trimmed, since some feeds include stray whitespace.
func headerIndex(headers []string, name string) (int, error) {
	target := strings.ToLower(strings.TrimSpace(name))
	for i, h := range headers {
		if strings.ToLower(strings.TrimSpace(h)) == target {
			return i, nil
		}
	}
	return -1, fmt.Errorf("column %q not found", name)
}

// openCSV opens path and returns a *csv.Reader positioned just past the
// header row. The caller is responsible for closing the underlying file.
func openCSV(path string) (*csv.Reader, *os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1 // tolerate rows of variable width during parse
	if _, err := r.Read(); err != nil { // discard header
		f.Close()
		return nil, nil, err
	}
	return r, f, nil
}

// at returns the field at idx or "" if idx is out of range. GTFS feeds
// frequently have empty trailing fields, so this is the safe accessor.
func at(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// parseInt parses a non-empty string as int, returning an error on failure.
// Empty strings parse as 0 with no error so the caller can still emit a
// warning for missing optional integers.
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

// parseFloat parses a non-empty string as float64.
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

func parseAgency(path string, d *Data) error {
	r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("gtfs: agency: skip malformed row: %v", err)
			continue
		}
		// Header is discarded by openCSV; remaining rows align by column.
		d.Agencies = append(d.Agencies, Agency{
			ID:       at(row, 0),
			Name:     at(row, 1),
			URL:      at(row, 2),
			Timezone: at(row, 3),
		})
	}
	return nil
}

func parseRoutes(path string, d *Data) error {
	r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("gtfs: routes: skip malformed row: %v", err)
			continue
		}
		typ, perr := parseInt(at(row, 4))
		if perr != nil {
			log.Printf("gtfs: routes: skip row %q: bad route_type: %v", strings.Join(row, ","), perr)
			continue
		}
		d.Routes = append(d.Routes, Route{
			ID:        at(row, 0),
			AgencyID:  at(row, 1),
			ShortName: at(row, 2),
			LongName:  at(row, 3),
			Type:      typ,
			Color:     at(row, 5),
			TextColor: at(row, 6),
		})
	}
	return nil
}

func parseStops(path string, d *Data) error {
	r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("gtfs: stops: skip malformed row: %v", err)
			continue
		}
		lat, err1 := parseFloat(at(row, 2))
		lon, err2 := parseFloat(at(row, 3))
		if err1 != nil || err2 != nil {
			log.Printf("gtfs: stops: skip row %q: bad lat/lon: %v / %v", strings.Join(row, ","), err1, err2)
			continue
		}
		d.Stops = append(d.Stops, Stop{
			ID:   at(row, 0),
			Name: at(row, 1),
			Lat:  lat,
			Lon:  lon,
		})
	}
	return nil
}

func parseTrips(path string, d *Data) error {
	r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("gtfs: trips: skip malformed row: %v", err)
			continue
		}
		dir, perr := parseInt(at(row, 4))
		if perr != nil {
			log.Printf("gtfs: trips: skip row %q: bad direction_id: %v", strings.Join(row, ","), perr)
			continue
		}
		d.Trips = append(d.Trips, Trip{
			ID:        at(row, 2),
			RouteID:   at(row, 0),
			ServiceID: at(row, 1),
			Headsign:  at(row, 3),
			Direction: dir,
			ShapeID:   at(row, 5),
		})
	}
	return nil
}

func parseStopTimes(path string, d *Data) error {
	r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// stop_times.txt is the largest file (~1.1M rows). We pre-grow the
	// slice to amortize reallocations.
	d.StopTimes = make([]StopTime, 0, 1_200_000)

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("gtfs: stop_times: skip malformed row: %v", err)
			continue
		}
		seq, perr := parseInt(at(row, 4))
		if perr != nil {
			log.Printf("gtfs: stop_times: skip row %q: bad stop_sequence: %v", strings.Join(row, ","), perr)
			continue
		}
		d.StopTimes = append(d.StopTimes, StopTime{
			TripID:        at(row, 0),
			ArrivalTime:   at(row, 1),
			DepartureTime: at(row, 2),
			StopID:        at(row, 3),
			Sequence:      seq,
		})
	}
	return nil
}

func parseShapes(path string, d *Data) error {
	r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// shapes.txt is large (~40K rows) but still small. Plain append is fine.
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("gtfs: shapes: skip malformed row: %v", err)
			continue
		}
		lat, err1 := parseFloat(at(row, 1))
		lon, err2 := parseFloat(at(row, 2))
		seq, err3 := parseInt(at(row, 3))
		if err1 != nil || err2 != nil || err3 != nil {
			log.Printf("gtfs: shapes: skip row %q: %v / %v / %v", strings.Join(row, ","), err1, err2, err3)
			continue
		}
		d.Shapes = append(d.Shapes, ShapePoint{
			ShapeID:  at(row, 0),
			Lat:      lat,
			Lon:      lon,
			Sequence: seq,
		})
	}
	return nil
}

func parseCalendar(path string, d *Data) error {
	r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("gtfs: calendar: skip malformed row: %v", err)
			continue
		}
		// day-of-week fields are 0/1; empty is treated as 0 (no service).
		mon, _ := parseInt(at(row, 1))
		tue, _ := parseInt(at(row, 2))
		wed, _ := parseInt(at(row, 3))
		thu, _ := parseInt(at(row, 4))
		fri, _ := parseInt(at(row, 5))
		sat, _ := parseInt(at(row, 6))
		sun, _ := parseInt(at(row, 7))

		d.Calendar = append(d.Calendar, Calendar{
			ServiceID: at(row, 0),
			Monday:    mon == 1,
			Tuesday:   tue == 1,
			Wednesday: wed == 1,
			Thursday:  thu == 1,
			Friday:    fri == 1,
			Saturday:  sat == 1,
			Sunday:    sun == 1,
			StartDate: at(row, 8),
			EndDate:   at(row, 9),
		})
	}
	return nil
}

func parseCalendarDates(path string, d *Data) error {
	r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("gtfs: calendar_dates: skip malformed row: %v", err)
			continue
		}
		et, perr := parseInt(at(row, 2))
		if perr != nil {
			log.Printf("gtfs: calendar_dates: skip row %q: bad exception_type: %v", strings.Join(row, ","), perr)
			continue
		}
		d.CalendarDates = append(d.CalendarDates, CalendarDate{
			ServiceID:    at(row, 0),
			Date:         at(row, 1),
			ExceptionType: et,
		})
	}
	return nil
}

// buildIndexes populates all lookup maps in d. Pointers are stored so the
// request path does not copy the underlying struct on every lookup.
//
// TimesByTrip and ShapeByID are sorted by Sequence ascending after grouping
// so handlers can return monotonic order without a per-request sort.
func buildIndexes(d *Data) {
	d.RouteByID = make(map[string]*Route, len(d.Routes))
	d.StopByID = make(map[string]*Stop, len(d.Stops))
	d.TripByID = make(map[string]*Trip, len(d.Trips))
	d.TripsByRoute = make(map[string][]*Trip, len(d.Routes))
	d.TimesByStop = make(map[string][]*StopTime)
	d.TimesByTrip = make(map[string][]*StopTime)
	d.ShapeByID = make(map[string][]ShapePoint)
	d.CalByService = make(map[string]*Calendar, len(d.Calendar))

	for i := range d.Routes {
		r := &d.Routes[i]
		d.RouteByID[r.ID] = r
	}
	for i := range d.Stops {
		s := &d.Stops[i]
		d.StopByID[s.ID] = s
	}
	for i := range d.Trips {
		t := &d.Trips[i]
		d.TripByID[t.ID] = t
		d.TripsByRoute[t.RouteID] = append(d.TripsByRoute[t.RouteID], t)
	}
	for i := range d.StopTimes {
		st := &d.StopTimes[i]
		d.TimesByStop[st.StopID] = append(d.TimesByStop[st.StopID], st)
		d.TimesByTrip[st.TripID] = append(d.TimesByTrip[st.TripID], st)
	}
	for i := range d.Shapes {
		sp := d.Shapes[i]
		d.ShapeByID[sp.ShapeID] = append(d.ShapeByID[sp.ShapeID], sp)
	}
	for i := range d.Calendar {
		c := &d.Calendar[i]
		d.CalByService[c.ServiceID] = c
	}

	// Sort the two sequence-sensitive indexes so handlers serve monotonic
	// data directly. Sort copies slice headers, not the underlying structs.
	for k := range d.TimesByTrip {
		ts := d.TimesByTrip[k]
		sort.Slice(ts, func(i, j int) bool { return ts[i].Sequence < ts[j].Sequence })
		d.TimesByTrip[k] = ts
	}
	for k := range d.ShapeByID {
		pts := d.ShapeByID[k]
		sort.Slice(pts, func(i, j int) bool { return pts[i].Sequence < pts[j].Sequence })
		d.ShapeByID[k] = pts
	}
}
