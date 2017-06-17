package main

import (
	"sync"
)

const kTableLimit = 1000

// Keep it simple for now
type kTable struct {
	sync.Mutex
	id    string
	nodes []*remoteNode
}

func newKTable(id string) kTable {
	k := kTable{id: id}
	k.refresh()
	return k
}

func (k *kTable) add(rn *remoteNode) {
	k.Lock()
	defer k.Unlock()
	k.nodes = append(k.nodes, rn)
}

func (k *kTable) getNodes() []*remoteNode {
	k.Lock()
	defer k.Unlock()
	return k.nodes
}

func (k *kTable) isEmpty() bool {
	k.Lock()
	defer k.Unlock()
	return len(k.nodes) == 0
}

func (k *kTable) isFull() bool {
	k.Lock()
	defer k.Unlock()
	return len(k.nodes) >= kTableLimit
}

// For now
func (k *kTable) refresh() {
	k.Lock()
	defer k.Unlock()
	k.nodes = make([]*remoteNode, 0)
}
