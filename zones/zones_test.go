package zones

import (
	"testing"
)

func TestZonesLoad(t *testing.T) {
	zs, err := LoadZones("zones.geojson")
	if err != nil {
		t.Fatalf("LoadZones failed: %v", err)
	}
	if len(zs) != 8 {
		t.Errorf("expected 8 zones, got %d", len(zs))
	}
	for _, z := range zs {
		if z.Name == "" {
			t.Errorf("zone missing name: %+v", z)
		}
		if z.ZoneType == "" {
			t.Errorf("zone %q missing zone_type", z.Name)
		}
		if z.SilenceToleranceS == 0 {
			t.Errorf("zone %q has zero silence_tolerance_s", z.Name)
		}
	}
}

func TestZoneProperties(t *testing.T) {
	zs, err := LoadZones("zones.geojson")
	if err != nil {
		t.Fatalf("LoadZones failed: %v", err)
	}
	validTypes := map[string]bool{"coastal": true, "offshore": true, "open_ocean": true}
	for _, z := range zs {
		if !validTypes[z.ZoneType] {
			t.Errorf("zone %q has invalid zone_type %q", z.Name, z.ZoneType)
		}
		if z.SilenceToleranceS <= 0 {
			t.Errorf("zone %q has non-positive silence_tolerance_s %d", z.Name, z.SilenceToleranceS)
		}
		if z.BoundaryBufferKm <= 0 {
			t.Errorf("zone %q has non-positive boundary_buffer_km %f", z.Name, z.BoundaryBufferKm)
		}
		if len(z.Coordinates) == 0 {
			t.Errorf("zone %q has no coordinates", z.Name)
		}
	}
}
