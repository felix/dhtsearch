package dht

import (
	"fmt"
	"net"
)

type remoteNode struct {
	addr net.Addr
	id   Infohash
}

// String implements fmt.Stringer
func (r remoteNode) String() string {
	return fmt.Sprintf("%s (%s)", r.id.String(), r.addr.String())
}
