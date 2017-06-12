package main

import (
	"fmt"
	"sync"
)

const kTableLimit = 1000

// Keep it simple for now
type kTable struct {
	sync.Mutex
	nodes []*remoteNode
}

func newKTable() kTable {
	k := kTable{}
	k.refresh()
	return k
}

func (k *kTable) add(rn *remoteNode) {
	k.Lock()
	defer k.Unlock()
	if rn == nil || rn.id == "" {
		fmt.Println("Trying to add invalid rn")
		return
	}
	if k.isFull() {
		k.refresh()
	}
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
