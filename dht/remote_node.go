package dht

import (
	"fmt"
	"net"
	//"time"
)

type remoteNode struct {
	address net.UDPAddr
	id      Infohash
	//lastSeen time.Time
}

// String implements fmt.Stringer
func (r *remoteNode) String() string {
	return fmt.Sprintf("%s:%d", r.address.IP.String(), r.address.Port)
}
