package dht

import "testing"

func TestCompactNodeList(t *testing.T) {
	type nodeInfo struct {
		id string
		addr string
	}
	tests := map[string]struct {
		in string
		n  int
		items []nodeInfo
	}{
		"contrived": {
			in: "52:\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x02\x03\x04\x05\x06" +
				"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x03\x04\x05\x06\x07",
			n: 2,
			items: []nodeInfo{{
				id: "0000000000000000000000000000000000000000",
				addr: "1.2.3.4:1286",
			},{
				id: "0000000000000000000000000000000000000000",
				addr: "2.3.4.5:1543",
			}},
		},
		"captured": {
			in: "208:\xb4\x99d\xcf)\xaa\x11\x14\x8d\r\x8b\x8dL̑b\xf6\x14/2\xaf\xb1\xcc+P\xb6\xb4\x99d\xcf)\xaa\x11\x14\x8d\r\x8b\x8dL̑b\xf6\x14/2\xaf\xb1\xcc+P\xb6\xb4\x99d\xcf)\xaa\x11\x14\x8d\r\x8b\x8dL̑b\xf6\x14/2\xaf\xb1\xcc+P\xb6\xb4\x99d\xcf)\xaa\x11\x14\x8d\r\x8b\x8dL̑b\xf6\x14/2\xaf\xb1\xcc+P\xb6\xb4\x99d\xcf)\xaa\x11\x14\x8d\r\x8b\x8dL̑b\xf6\x14/2\xaf\xb1\xcc+P\xb6\xb4\x99d\xcf)\xaa\x11\x14\x8d\r\x8b\x8dL̑b\xf6\x14/2\xaf\xb1\xcc+P\xb6\xb4\x99d\xcf)\xaa\x11\x14\x8d\r\x8b\x8dL̑b\xf6\x14/2\xaf\xb1\xcc+P\xb6\xb4\x99d\xcf)\xaa\x11\x14\x8d\r\x8b\x8dL̑b\xf6\x14/2\xaf\xb1\xcc+P\xb6",
			n:  8,
			items: []nodeInfo{{
				id: "b49964cf29aa11148d0d8b8d4ccc9162f6142f32",
				addr: "175.177.204.43:20662",
			},{
				id: "b49964cf29aa11148d0d8b8d4ccc9162f6142f32",
				addr: "175.177.204.43:20662",
			}},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Logf("input length=%d", len([]byte(tt.in)))
			var cnl CompactNodeList
			if err := cnl.UnmarshalBencode([]byte(tt.in)); err != nil {
				t.Fatal(err)
			}
			if len(cnl.Nodes) != tt.n {
				t.Fatalf("got %d, want %d", len(cnl.Nodes), tt.n)
			}
			for i, ni := range tt.items {
				id := cnl.Nodes[i].ID
				if id.String() != ni.id {
					t.Fatalf("got %q, want %q", id.String(), ni.id)
				}
				addr := cnl.Nodes[i]
				if addr.AddrPort.String() != ni.addr {
					t.Fatalf("got %q, want %q", addr.AddrPort.String(), ni.addr)
				}
			}
		})
	}

}
