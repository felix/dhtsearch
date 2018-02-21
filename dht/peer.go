package dht

import (
	"fmt"
	"net"
)

// Peer on DHT network
type Peer struct {
	Addr     net.Addr
	ID       Infohash
	Infohash Infohash
}

// String implements fmt.Stringer
func (p Peer) String() string {
	return fmt.Sprintf("%s (%s)", p.Infohash, p.Addr.String())
}
