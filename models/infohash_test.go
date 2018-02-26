package models

import (
	"encoding/hex"
	"testing"
)

func TestInfohashImport(t *testing.T) {

	tests := []struct {
		str string
		ok  bool
	}{
		{str: "5a3ce1c14e7a08645677bbd1cfe7d8f956d53256", ok: true},
		{str: "5a3ce1c14e7a08645677bbd1cfe7d8f956d53256000", ok: false},
	}

	for _, tt := range tests {
		ih, err := InfohashFromString(tt.str)
		if tt.ok {
			if err != nil {
				t.Errorf("FromString failed with %s", err)
			}

			idBytes, err := hex.DecodeString(tt.str)
			if err != nil {
				t.Errorf("failed to decode %s to hex", tt.str)
			}
			ih2 := Infohash(idBytes)
			if !ih.Equal(ih2) {
				t.Errorf("expected %s to equal %s", ih, ih2)
			}
		} else {
			if err == nil {
				t.Errorf("FromString should have failed for %s", tt.str)
			}
		}
	}
}

func TestInfohashLength(t *testing.T) {
	ih := GenInfohash()
	if len(ih) != 20 {
		t.Errorf("%s as string should be length 20, got %d", ih, len(ih))
	}
}

func TestInfohashDistance(t *testing.T) {
	id := "d1c5676ae7ac98e8b19f63565905105e3c4c37a2"

	var tests = []struct {
		ih       string
		other    string
		distance int
	}{
		{id, id, 160},
		{id, "d1c5676ae7ac98e8b19f63565905105e3c4c37a3", 159},
	}

	ih, err := InfohashFromString(id)
	if err != nil {
		t.Errorf("Failed to create Infohash: %s", err)
	}

	for _, tt := range tests {
		other, err := InfohashFromString(tt.other)
		if err != nil {
			t.Errorf("Failed to create Infohash: %s", err)
		}

		dist := ih.Distance(*other)
		if dist != tt.distance {
			t.Errorf("Distance() => %d, expected %d", dist, tt.distance)
		}
	}
}
