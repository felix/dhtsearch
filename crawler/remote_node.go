package crawler

import (
	"fmt"
	"net"
	//"time"
)

type remoteNode struct {
	address net.UDPAddr
	id      string
	//lastSeen time.Time
}

func newRemoteNode(addr net.UDPAddr, id string) *remoteNode {
	return &remoteNode{
		address: addr,
		id:      id,
	}
}

func (r *remoteNode) String() string {
	return fmt.Sprintf("%s:%d", r.address.IP.String(), r.address.Port)
}
