package dht

import (
	"fmt"
	"net"
)

type remoteNode struct {
	address net.Addr
	id      Infohash
	//lastSeen time.Time
}

// String implements fmt.Stringer
func (r remoteNode) String() string {
	return fmt.Sprintf("%s (%s)", r.id.String(), r.address.String())
}
