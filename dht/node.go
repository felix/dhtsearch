package dht

import (
	"errors"
	"net/netip"
	"sync"
	"time"

	"userspace.com.au/dhtsearch/infohash"
)

type Node struct {
	sync.Mutex
	ID       infohash.ID
	AddrPort netip.AddrPort
	Secure   bool
	Family   string

	// To assist with rate limiting
	NextQuery map[string]time.Time

	// Incremented when selected for ping, cleared on pong
	PingAttempts int

	// Used by table
	LastContact time.Time
}

func (n *Node) String() string {
	return n.AddrPort.String()
}

func (n *Node) canSend(query string) bool {
	if len(n.NextQuery) == 0 {
		return true
	}
	n.Lock()
	defer n.Unlock()
	t, ok := n.NextQuery[query]
	return !ok || time.Now().After(t)
}

func (n *Node) rateQuery(q string, s int64) {
	n.Lock()
	defer n.Unlock()
	if len(n.NextQuery) == 0 {
		n.NextQuery = make(map[string]time.Time)
	}
	if s == 0 {
		n.NextQuery[q] = time.Now()
		return
	}
	if s == -1 {
		// TODO delay one day
		s = 24 * 60 * 60
	}
	n.NextQuery[q] = time.Now().Add(time.Duration(s) * time.Second)
}

func (n *Node) MarshalBinary() ([]byte, error) {
	b, _ := NodeAddr(n.AddrPort).MarshalBinary()
	return append(n.ID.Bytes(), b...), nil
}

func (n *Node) UnmarshalBinary(b []byte) error {
	n.ID = infohash.ID(b[:infohash.Length])
	var na NodeAddr
	if err := na.UnmarshalBinary(b[infohash.Length:]); err != nil {
		return nil
	}
	n.AddrPort = netip.AddrPort(na)
	if !n.AddrPort.IsValid() || n.AddrPort.Port() == 0 {
		return errors.New("invalid AddrPort")
	}
	return nil
}
