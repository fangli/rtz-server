package v1

import (
	"testing"

	"github.com/USA-RedDragon/rtz-server/internal/config"
	"github.com/USA-RedDragon/rtz-server/internal/db/models"
)

func TestRouteResponseURLUsesExistingRouteURL(t *testing.T) {
	cfg := &config.Config{}
	route := models.Route{RouteID: "2c33d381dc", URL: "https://example.com/route"}

	got := routeResponseURL(cfg, "071d7c74c4b94d38", route)

	if got != route.URL {
		t.Fatalf("routeResponseURL() = %q, want %q", got, route.URL)
	}
}

func TestRouteResponseURLFallsBackToUploadedRouteBase(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.BackendURL = "https://api.cloudsyn.com/"
	route := models.Route{RouteID: "0000000d--2c33d381dc"}

	got := routeResponseURL(cfg, "071d7c74c4b94d38", route)
	want := "https://api.cloudsyn.com/v1/route_file/071d7c74c4b94d38/0000000d--2c33d381dc"

	if got != want {
		t.Fatalf("routeResponseURL() = %q, want %q", got, want)
	}
}

func TestSegmentFromUploadDirMatchesFullNewRouteID(t *testing.T) {
	segment, ok := segmentFromUploadDir("0000000d--2c33d381dc", "0000000d--2c33d381dc--3")
	if !ok {
		t.Fatal("expected full new route id to match upload dir")
	}
	if segment != 3 {
		t.Fatalf("segment = %d, want %d", segment, 3)
	}
}

func TestSegmentFromUploadDirMatchesShortNewRouteIDForExistingRows(t *testing.T) {
	segment, ok := segmentFromUploadDir("2c33d381dc", "0000000d--2c33d381dc--3")
	if !ok {
		t.Fatal("expected short new route id to match upload dir")
	}
	if segment != 3 {
		t.Fatalf("segment = %d, want %d", segment, 3)
	}
}
