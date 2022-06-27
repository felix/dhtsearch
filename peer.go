package dhtsearch

import (
	"fmt"
	"net"
	"time"
)

// Peer on DHT network
type Peer struct {
	Addr     net.Addr  `db:"address"`
	Infohash Infohash  `db:"infohash"`
	Created  time.Time `db:"created" json:"created"`
	Updated  time.Time `db:"updated" json:"updated"`
}

// String implements fmt.Stringer
func (p Peer) String() string {
	return fmt.Sprintf("%s (%s)", p.Infohash, p.Addr.String())
}
