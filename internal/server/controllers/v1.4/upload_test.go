package v1dot4

import "testing"

func TestNewRouteInfoFromPathKeepsFullRouteID(t *testing.T) {
	info, ok := newRouteInfoFromPath("0000000d--2c33d381dc--3/qlog.zst")
	if !ok {
		t.Fatal("expected new route path to match")
	}
	if info.Motonic != "0000000d" {
		t.Fatalf("Motonic = %q, want %q", info.Motonic, "0000000d")
	}
	if info.Route != "0000000d--2c33d381dc" {
		t.Fatalf("Route = %q, want %q", info.Route, "0000000d--2c33d381dc")
	}
	if info.Segment != "3" {
		t.Fatalf("Segment = %q, want %q", info.Segment, "3")
	}
}
