package dht

import "net"

// Peer on DHT network
type Peer struct {
	Address net.UDPAddr
	ID      Infohash
}
