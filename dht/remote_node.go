package dht

import (
	"fmt"
	"net"

	"src.userspace.com.au/dhtsearch"
)

type remoteNode struct {
	addr net.Addr
	id   dhtsearch.Infohash
}

// String implements fmt.Stringer
func (r remoteNode) String() string {
	return fmt.Sprintf("%s (%s)", r.id.String(), r.addr.String())
}
