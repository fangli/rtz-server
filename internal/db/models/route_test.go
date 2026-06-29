package models

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRouteSegmentArraysScanPostgresArrayText(t *testing.T) {
	db := newRouteTestDB(t)
	start := time.Date(2026, 6, 29, 22, 11, 32, 0, time.UTC)
	end := start.Add(time.Minute)

	err := db.Exec(`
		INSERT INTO routes (
			device_id, route_id, start_time, end_time,
			segment_start_times, segment_end_times, segment_numbers
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, 4, "2c33d381dc", start, end, "{1782771092838954186}", "{1782771153797129217}", "{1}").Error
	if err != nil {
		t.Fatalf("insert route: %v", err)
	}

	route := findOnlyRoute(t, db, start, end)
	assertInt64Slice(t, "segment_start_times", route.SegmentStartTimes, []int64{1782771092838954186})
	assertInt64Slice(t, "segment_end_times", route.SegmentEndTimes, []int64{1782771153797129217})
	assertInt64Slice(t, "segment_numbers", route.SegmentNumbers, []int64{1})
}

func TestRouteSegmentArraysScanLegacySQLiteScalar(t *testing.T) {
	db := newRouteTestDB(t)
	start := time.Date(2026, 6, 29, 22, 11, 32, 0, time.UTC)
	end := start.Add(time.Minute)

	err := db.Exec(`
		INSERT INTO routes (
			device_id, route_id, start_time, end_time,
			segment_start_times, segment_end_times, segment_numbers
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, 4, "2c33d381dc", start, end, 1782771092838954186, 1782771153797129217, 1).Error
	if err != nil {
		t.Fatalf("insert route: %v", err)
	}

	route := findOnlyRoute(t, db, start, end)
	assertInt64Slice(t, "segment_start_times", route.SegmentStartTimes, []int64{1782771092838954186})
	assertInt64Slice(t, "segment_end_times", route.SegmentEndTimes, []int64{1782771153797129217})
	assertInt64Slice(t, "segment_numbers", route.SegmentNumbers, []int64{1})
}

func newRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&Route{}); err != nil {
		t.Fatalf("migrate routes: %v", err)
	}
	return db
}

func findOnlyRoute(t *testing.T, db *gorm.DB, start, end time.Time) Route {
	t.Helper()

	routes, err := FindRoutesByDeviceIDAndTimeRange(db, 4, start.Add(-time.Second), end.Add(time.Second), 5)
	if err != nil {
		t.Fatalf("find routes: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	return routes[0]
}

func assertInt64Slice(t *testing.T, name string, got []int64, want []int64) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s length: got %d, want %d (%#v)", name, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d]: got %d, want %d", name, i, got[i], want[i])
		}
	}
}
