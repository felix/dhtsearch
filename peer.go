package dhtsearch

import "net"

// Annouced peer
type peer struct {
	address net.UDPAddr
	id      string
}
