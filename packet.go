package dhtsearch

import "net"

// Unprocessed packet from socket
type packet struct {
	b     []byte
	raddr net.UDPAddr
}
