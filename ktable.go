package main

const kTableLimit = 5000

// Keep it simple for now
type kTable struct {
	nodes []*remoteNode
}

func newKTable() kTable {
	return kTable{make([]*remoteNode, 0)}
}

func (k *kTable) add(rn *remoteNode) {
	if len(k.nodes) >= kTableLimit {
		return
	}
	k.nodes = append(k.nodes, rn)
}

// For now
func (k *kTable) refresh() {
	k.nodes = make([]*remoteNode, 0)
}
