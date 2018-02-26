package dht

import (
	"fmt"
	"net"

	"github.com/felix/dhtsearch/models"
)

type remoteNode struct {
	addr net.Addr
	id   models.Infohash
}

// String implements fmt.Stringer
func (r remoteNode) String() string {
	return fmt.Sprintf("%s (%s)", r.id.String(), r.addr.String())
}
