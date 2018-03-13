package models

import (
	"fmt"
	"net"
	"time"
)

// Peer on DHT network
type Peer struct {
	Addr     net.Addr  `db:"address"`
	Infohash Infohash  `db:"infohash"`
	Updated  time.Time `json:"updated"`
	Created  time.Time `json:"created"`
}

// String implements fmt.Stringer
func (p Peer) String() string {
	return fmt.Sprintf("%s (%s)", p.Infohash, p.Addr.String())
}
