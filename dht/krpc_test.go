package dht

import (
	"testing"

	"userspace.com.au/dhtsearch/bencode"
	"userspace.com.au/dhtsearch/infohash"
)

func TestKRPCMsg(t *testing.T) {
	id := infohash.ID([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
	tests := []struct {
		in  Msg
		out string
	}{{
		in:  Msg{Args: &MsgArgs{ID: &id}},
		out: "d1:ad2:id20:\x01\x02\x03\x04\x05\x06\a\b\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14e1:t0:1:y0:e",
	}, {
		in:  Msg{Args: &MsgArgs{Want: []string{"n4", "n6"}}},
		out: "d1:ad4:wantl2:n42:n6ee1:t0:1:y0:e",
	}}

	for _, tt := range tests {
		t.Run("marshal", func(t *testing.T) {
			t.Logf("pre msg: %v", tt.in)
			b, err := bencode.Marshal(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tt.out {
				t.Fatalf("got %q, want %q", string(b), tt.out)
			}
		})
		// t.Run("unmarshal", func(t *testing.T) {
		// 	var got Msg
		// 	err := bencode.Unmarshal([]byte(tt.out), &got)
		// 	if err != nil {
		// 		t.Fatal(err)
		// 	}
		// 	if got != tt.in {
		// 		t.Fatalf("got %v, want %v", got, tt.in)
		// 	}
		// })
	}
}

func TestError(t *testing.T) {
	in := "d1:eli201e17:too many requestse1:t2:rz1:y1:re"
	var m Msg
	if err := bencode.Unmarshal([]byte(in), &m); err != nil {
		t.Fatal(err)
	}
	if m.Error.Code != 201 {
		t.Errorf("got %v, want %d", m.Error.Code, 201)
	}
	if m.Error.Msg != "too many requests" {
		t.Errorf("got %v, want %d", m.Error.Code, 201)
	}
}

// func TestCompactNode(t *testing.T) {
// 	ih := "infohashinfohash1234"
// 	idIn, _ := infohash.FromString(ih)
// 	tests := []struct {
// 		ip   string
// 		port uint16
// 	}{
// 		{ip: "127.0.0.1", port: 6881},
// 		{ip: "[2404:6800:4006:814::200e]", port: 6881},
// 	}
//
// 	type compactNode interface {
// 		ID() (*infohash.ID, error)
// 		AddrPort() (netip.AddrPort, error)
// 	}
//
// 	for _, tt := range tests {
// 		apIn, err := netip.ParseAddrPort(fmt.Sprintf("%s:%d", tt.ip, tt.port))
// 		if err != nil {
// 			panic(err)
// 		}
// 		s := NewCompactNodeInfo(*idIn, apIn.Addr(), tt.port)
// 		t.Logf("CompactNodeInfo: %d %v", len(s), []byte(s))
//
// 		var cni compactNode
// 		if apIn.Addr().Is6() {
// 			cni = CompactNode6Info(s)
// 		} else {
// 			cni = CompactNodeInfo(s)
// 		}
// 		ap, err := cni.AddrPort()
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if ap != apIn {
// 			t.Errorf("got %v, want %v", ap, apIn)
// 		}
// 		id, err := cni.ID()
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if !id.Equal(*idIn) {
// 			t.Errorf("got %v, want %v", id, idIn)
// 		}
// 	}
// }
