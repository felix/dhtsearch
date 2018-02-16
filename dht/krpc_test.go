package dht

import (
	"encoding/hex"
	"testing"
)

func TestCompactNodeAddr(t *testing.T) {

	tests := []struct {
		in  string
		out string
	}{
		{in: "192.168.1.1:6881", out: "c0a801011ae1"},
		{in: "[2001:9372:434a:800::2]:6881", out: "20019372434a080000000000000000021ae1"},
	}

	for _, tt := range tests {
		r := encodeCompactNodeAddr(tt.in)
		out, _ := hex.DecodeString(tt.out)
		if r != string(out) {
			t.Errorf("encodeCompactNodeAddr(%s) => %x, expected %s", tt.in, r, tt.out)
		}

		s := decodeCompactNodeAddr(r)
		if s != tt.in {
			t.Errorf("decodeCompactNodeAddr(%x) => %s, expected %s", r, s, tt.in)
		}
	}
}
