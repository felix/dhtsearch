package dht

import (
	"fmt"

	"userspace.com.au/dhtsearch/bencode"
	"userspace.com.au/dhtsearch/infohash"
)

type Msg struct {
	// Query method
	// one of: "ping", "find_node", "get_peers", "announce_peer"
	Query string `bencode:"q,omitzero"`

	// Arguments sent with a query
	Args *MsgArgs `bencode:"a,omitzero"`

	// Transaction ID, required
	TID string `bencode:"t"`

	// Type of message, required
	// one of: q for QUERY, r for RESPONSE, e for ERROR
	Type string `bencode:"y"`

	// Response payload for type 'r'
	Response *Response `bencode:"r,omitzero"`

	// Error payload for type 'e'
	Error *Error `bencode:"e,omitzero"`

	//IP       NodeAddr `bencode:"ip,omitzero"`
	// Node IP address, required for BEP42
	IP string `bencode:"ip,omitzero"`

	// Sender does not respond to queries, BEP43
	ReadOnly bool `bencode:"ro,omitzero"`

	// https://www.libtorrent.org/dht_extensions.html
	ClientId string `bencode:"v,omitzero"`
}

type Response struct {
	ID *infohash.ID `bencode:"id"`

	// K closest nodes
	// from get_peers, find_nodes, get & sample_infohashes
	NodesList  *CompactNodeList  `bencode:"nodes,omitzero"`
	Nodes6List *CompactNode6List `bencode:"nodes6,omitzero"`

	// Token for future announce_peer or put, BEP44
	Token string

	// Torrent peers
	Values []NodeAddr

	// BEP33 (scrapes)
	// BFsd *ScrapeBloomFilter `bencode:"BFsd,omitzero"`
	// BFpe *ScrapeBloomFilter `bencode:"BFpe,omitzero"`

	// BEP51
	Interval int64
	Num      int64
	// Nodes supporting this extension should always include the samples field in the response, even
	// when it is zero-length. This lets indexing nodes to distinguish nodes supporting this
	// extension from those that respond to unknown query types which contain a target field.
	Samples []byte `bencode:"samples,omitzero"`
}

const (
	Want6 = "n6"
	Want4 = "n4"
)

type MsgArgs struct {
	// ID of the querying Node
	ID *infohash.ID `bencode:"id,omitzero"`

	// InfoHash of the torrent
	InfoHash *infohash.ID `bencode:"info_hash,omitzero"`

	// ID of the node sought
	Target *infohash.ID `bencode:"target,omitzero"`

	// Token received from an earlier get_peers query
	// Also used in a BEP44 put
	Token string `bencode:"token,omitzero"`

	// Sender's torrent port
	Port int `bencode:"port,omitzero"`

	// Use senders apparent DHT port
	ImpliedPort bool `bencode:"implied_port,omitzero"`

	// Network family wanted, any of "n4" and "n6", BEP32
	Want []string `bencode:"want,omitzero"`

	// BEP33
	// NoSeed int   `bencode:"noseed,omitzero"`
	// Scrape int   `bencode:"scrape,omitzero"`

	// BEP44
	//V any `bencode:"v,omitzero"`
	// Seq  *int64   `bencode:"seq,omitzero"`
	// Cas  int64    `bencode:"cas,omitzero"`
	// K    [32]byte `bencode:"k,omitzero"`
	// Salt []byte   `bencode:"salt,omitzero"`
	// Sig  [64]byte `bencode:"sig,omitzero"`
}

type Error struct {
	Code int64
	Msg  string
}

func (e *Error) UnmarshalBencode(b []byte) error {
	r := bencode.NewReaderFromBytes(b)
	ok := r.ReadList(func(r *bencode.Reader) bool {
		if !r.ReadInt(&e.Code) {
			return false
		}
		if !r.ReadString(&e.Msg) {
			return false
		}
		return true
	})
	if !ok {
		return fmt.Errorf("error unmarshal failed: %w", r.Err())
	}
	return nil
}

func NewFindNodeQuery(id, target infohash.ID, wants []string) *Msg {
	return &Msg{
		Query: "find_node",
		Type:  "q",
		Args: &MsgArgs{
			// The querying node
			ID: &id,
			// The ID sought
			Target: &target,
			Want:   wants,
		},
	}
}

func NewPingQuery(id infohash.ID) *Msg {
	return &Msg{
		Query: "ping",
		Args: &MsgArgs{
			ID: &id,
		},
	}
}

// func NewGetPeersQuery(id infohash.ID) *Msg {
// 	msg := &Msg{
// 		Query: "get_peers",
// 		Type:  "q",
// 		Args: &MsgArgs{
// 			ID:       &c.node.ID,
// 			InfoHash: &ih,
// 		},
// 	}
// 	return &Msg{
// 		Query: "ping",
// 		TID:   newTransactionID(),
// 		Args: &MsgArgs{
// 			ID: &id,
// 		},
// 	}
// }

// func NewSampleInfohashesQuery(id, target infohash.ID) *Msg {
// 	return &Msg{
// 		Query: "sample_infohashes",
// 		TID:   newTransactionID(),
// 		Type:  "q",
// 		Args: &MsgArgs{
// 			// The querying node
// 			ID: &id,
// 			// The ID sought
// 			Target: &target,
// 		},
// 	}
// }

// const (
// 	transIDBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
// )

// makeQuery returns a query-formed data.
// func MakeQuery(tid, query string, data map[string]any) map[string]any {
// 	return map[string]any{
// 		"t": tid,
// 		"y": "q",
// 		"q": query,
// 		"a": data,
// 	}
// }

// makeResponse returns a response-formed data.
// func MakeResponse(tid string, data map[string]any) map[string]any {
// 	return map[string]any{
// 		"t": tid,
// 		"y": "r",
// 		"r": data,
// 	}
// }

// func DecodeCompactNodeAddr(cni string) string {
// 	if len(cni) == 6 {
// 		return fmt.Sprintf("%d.%d.%d.%d:%d", cni[0], cni[1], cni[2], cni[3], (uint16(cni[4])<<8)|uint16(cni[5]))
// 	} else if len(cni) == 18 {
// 		b := []byte(cni[:16])
// 		return fmt.Sprintf("[%s]:%d", net.IP.String(b), (uint16(cni[16])<<8)|uint16(cni[17]))
// 	} else {
// 		return ""
// 	}
// }

// func EncodeCompactNodeAddr(addr string) string {
// 	var a []uint8
// 	host, port, _ := net.SplitHostPort(addr)
// 	ip := net.ParseIP(host)
// 	if ip == nil {
// 		return ""
// 	}
// 	aa, _ := strconv.ParseUint(port, 10, 16)
// 	c := uint16(aa)
// 	if ip2 := net.IP.To4(ip); ip2 != nil {
// 		a = make([]byte, net.IPv4len+2, net.IPv4len+2)
// 		copy(a, ip2[0:net.IPv4len]) // ignore bytes IPv6 bytes if it's IPv4.
// 		a[4] = byte(c >> 8)
// 		a[5] = byte(c)
// 	} else {
// 		a = make([]byte, net.IPv6len+2, net.IPv6len+2)
// 		copy(a, ip)
// 		a[16] = byte(c >> 8)
// 		a[17] = byte(c)
// 	}
// 	return string(a)
// }

// func int2bytes(val int64) []byte {
// 	data, j := make([]byte, 8), -1
// 	for i := 0; i < 8; i++ {
// 		shift := uint64((7 - i) * 8)
// 		data[i] = byte((val & (0xff << shift)) >> shift)

// 		if j == -1 && data[i] != 0 {
// 			j = i
// 		}
// 	}

// 	if j != -1 {
// 		return data[j:]
// 	}
// 	return data[:1]
// }
