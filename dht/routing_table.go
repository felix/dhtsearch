package dht

import (
	"net"
	"sync"
)

// Keep it simple for now
type routingTable struct {
	id      Infohash
	address net.UDPAddr
	nodes   []*remoteNode
	max     int
	sync.Mutex
}

func newRoutingTable(id Infohash) *routingTable {
	k := &routingTable{id: id, max: 4000}
	k.refresh()
	return k
}

func (k *routingTable) add(rn *remoteNode) {
	k.Lock()
	defer k.Unlock()

	// Check IP and ports are valid and not self
	if (rn.address.String() == k.address.String() && rn.address.Port == k.address.Port) || !rn.id.Valid() || rn.id.Equal(k.id) {
		return
	}
	k.nodes = append(k.nodes, rn)
}

func (k *routingTable) getNodes() []*remoteNode {
	k.Lock()
	defer k.Unlock()
	return k.nodes
}

func (k *routingTable) isEmpty() bool {
	k.Lock()
	defer k.Unlock()
	return len(k.nodes) == 0
}

func (k *routingTable) isFull() bool {
	k.Lock()
	defer k.Unlock()
	return len(k.nodes) >= k.max
}

// For now
func (k *routingTable) refresh() {
	k.Lock()
	defer k.Unlock()
	k.nodes = make([]*remoteNode, 0)
}
