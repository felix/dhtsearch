package dht

import (
	"container/heap"
	"sync"

	"src.userspace.com.au/dhtsearch"
)

type rItem struct {
	value    *remoteNode
	distance int
	index    int // Index in heap
}

type priorityQueue []*rItem

type routingTable struct {
	id        dhtsearch.Infohash
	max       int
	items     priorityQueue
	addresses map[string]*remoteNode
	sync.Mutex
}

func newRoutingTable(id dhtsearch.Infohash, max int) (*routingTable, error) {
	k := &routingTable{
		id:  id,
		max: max,
	}
	k.flush()
	heap.Init(&k.items)
	return k, nil
}

// Len implements sort.Interface
func (pq priorityQueue) Len() int { return len(pq) }

// Less implements sort.Interface
func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].distance > pq[j].distance
}

// Swap implements sort.Interface
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push implements heap.Interface
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*rItem)
	item.index = n
	*pq = append(*pq, item)
}

// Pop implements heap.Interface
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (k *routingTable) add(rn *remoteNode) {
	// Check IP and ports are valid and not self
	if !rn.id.Valid() || rn.id.Equal(k.id) {
		return
	}

	k.Lock()
	defer k.Unlock()

	if _, ok := k.addresses[rn.addr.String()]; ok {
		return
	}
	k.addresses[rn.addr.String()] = rn

	item := &rItem{
		value:    rn,
		distance: k.id.Distance(rn.id),
	}

	heap.Push(&k.items, item)

	if len(k.items) > k.max {
		for i := k.max - 1; i < len(k.items); i++ {
			old := k.items[i]
			delete(k.addresses, old.value.addr.String())
			heap.Remove(&k.items, i)
		}
	}
}

func (k *routingTable) get(n int) (out []*remoteNode) {
	if n == 0 {
		n = len(k.items)
	}
	for i := 0; i < n && i < len(k.items); i++ {
		out = append(out, k.items[i].value)
	}
	return out
}

func (k *routingTable) flush() {
	k.Lock()
	defer k.Unlock()

	k.items = make(priorityQueue, 0)
	k.addresses = make(map[string]*remoteNode, k.max)
}

func (k *routingTable) isEmpty() bool {
	k.Lock()
	defer k.Unlock()
	return len(k.items) == 0
}
