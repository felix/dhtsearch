package dht

import (
	"bytes"
	"net/netip"
	"testing"

	"userspace.com.au/dhtsearch/infohash"
)

func TestNodeMarshaling(t *testing.T) {
	tests := []Node{
		{
			ID:       infohash.MustParseString("0000000000000000000000000000000000000000"),
			AddrPort: netip.MustParseAddrPort("[2001:19f0:5:6d01:5400:2ff:feec:644a]:6881")},
		{
			ID:       infohash.MustParseString("c4301bf4b0a4f83eb7afe1eeee8434fb1893cc15"),
			AddrPort: netip.MustParseAddrPort("[2001:f90:4090:1230:2697:edff:fe27:d081]:50000")},
		{
			ID:       infohash.MustParseString("c17f24479d8c5ad3ad7a731a600ff323ecb26ebd"),
			AddrPort: netip.MustParseAddrPort("[2001:e68:542f:5959:11b7:2243:527d:1c86]:19264")},
	}

	for _, tt := range tests {
		t.Run(tt.ID.String(), func(t *testing.T) {
			b, err := tt.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			n2 := new(Node)
			if err := n2.UnmarshalBinary(b); err != nil {
				t.Fatal(err)
			}
			if tt.AddrPort.Compare(n2.AddrPort) != 0 {
				t.Fatalf("got %s, want %s", n2.AddrPort, tt.AddrPort)
			}
			if !bytes.Equal(n2.ID[:], tt.ID[:]) {
				t.Fatalf("got %s, want %s", n2.ID, tt.ID)
			}
		})
	}
}
