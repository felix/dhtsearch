package dht

import "fmt"

// Peer on DHT network
type Peer struct {
	Node     remoteNode
	Infohash Infohash
}

func (p Peer) String() string {
	return fmt.Sprintf("%s (%s)", p.Infohash, p.Node)
}
