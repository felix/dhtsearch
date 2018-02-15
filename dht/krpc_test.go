package dht

import (
	"testing"
)

func TestStringToCompactNodeInfo(t *testing.T) {

	tests := []struct {
		in  string
		out []byte
	}{
		{in: "192.168.1.1:6881", out: []byte("asdfasdf")},
	}

	for _, tt := range tests {
		r, err := stringToCompactNodeInfo(tt.in)
		if err != nil {
			t.Errorf("stringToCompactNodeInfo failed with %s", err)
		}
		if r != tt.out {
			t.Errorf("stringToCompactNodeInfo(%s) => %s, expected %s", tt.in, r, tt.out)
		}
	}
}
