package dht

import "net"

// Arbitrary packet types
// Order these lowest to highest priority for use in
// priority queue heap
const (
	_ int = iota
	pktQPing
	pktRPing
	pktQFindNode
	pktRAnnouncePeer
	pktRGetPeers
)

var pktName = map[int]string{
	pktQFindNode:     "find_node",
	pktQPing:         "ping",
	pktRPing:         "ping",
	pktRAnnouncePeer: "annouce_peer",
	pktRGetPeers:     "get_peers",
}

// Unprocessed packet from socket
type packet struct {
	// The packet type
	//priority int
	// Required by heap interface
	//index int
	data  []byte
	raddr net.Addr
}
