package dht

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"userspace.com.au/dhtsearch/bencode"
	"userspace.com.au/dhtsearch/infohash"
)

type Table struct {
	sync.RWMutex

	// Local node infohash
	id infohash.ID

	// Root bucket for the tree
	root *bucket

	log *slog.Logger

	// BEP5: Each bucket can only hold K nodes, currently eight, before
	// becoming "full."
	k int
}

type bucket struct {
	nodes     []*Node
	left      *bucket
	right     *bucket
	dontSplit bool

	// BEP5: Each bucket should maintain a "last changed" property to
	// indicate how "fresh" the contents are. When a node in a bucket is
	// pinged and it responds, or a node is added to a bucket, or a node in
	// a bucket is replaced with another node, the bucket's last changed
	// property should be updated. Buckets that have not been changed in
	// 15 minutes should be "refreshed." This is done by picking a random
	// ID in the range of the bucket and performing a find_nodes search on it.
	lastChanged time.Time
}

var _ RoutingTable = (*Table)(nil)

func NewTable(id infohash.ID) *Table {
	out := &Table{
		id:   id,
		root: newBucket(),
		k:    8,
	}
	return out
}

func (t *Table) ID() infohash.ID {
	return t.id
}

func (t *Table) Add(n *Node) bool {
	if t.id.Equal(n.ID) {
		t.log.Debug("not adding self")
		return false
	}
	t.Lock()
	defer t.Unlock()
	return t.addNoLock(n)
}

// Potentially recursive
func (t *Table) addNoLock(n *Node) bool {
	log := t.log.With("id", n.ID, "addr", n.AddrPort)
	b, bitIndex := t.locateBucket(n.ID)
	if b.has(n.ID) {
		//log.Debug("node exists")
		return false
	}
	if len(b.nodes) < t.k {
		b.add(n)
		ktableCalls.Add(
			context.TODO(), 1,
			metric.WithAttributes(attribute.String("method", "add")))
		log.Debug("added node")
		return true
	}
	if b.dontSplit {
		toReplace := b.mostQuestionable(2)
		if toReplace != nil {
			log.Debug(
				"replacing node",
				"old", toReplace.AddrPort,
				"attempts", toReplace.PingAttempts,
				"contact", toReplace.LastContact)
			b.remove(toReplace.ID)
			b.add(n)
			ktableCalls.Add(
				context.TODO(), 1,
				metric.WithAttributes(attribute.String("method", "replace")))
			return true
		}
		//log.Debug("bucket is full")
		return false
	}

	// BEP5: When a bucket is full of known good nodes, no more nodes may be
	// added unless our own node ID falls within the range of the bucket. In
	// that case, the bucket is replaced by two new buckets each with half the
	// range of the old bucket and the nodes from the old bucket are
	// distributed among the two new ones. For a new table with only one
	// bucket, the full bucket is always split into two new buckets covering
	// the ranges 0..2^159 and 2^159..2^160.
	b.split(bitIndex)
	b.farChild(t.id, bitIndex).dontSplit = true
	return t.addNoLock(n)
}

func (t *Table) Has(id infohash.ID) bool {
	t.RLock()
	defer t.RUnlock()
	return t.hasNoLock(id)
}

func (t *Table) hasNoLock(id infohash.ID) bool {
	b, _ := t.locateBucket(id)
	return b.has(id)
}

func (t *Table) Remove(id infohash.ID) bool {
	t.Lock()
	defer t.Unlock()
	return t.removeNoLock(id)
}

func (t *Table) removeNoLock(id infohash.ID) bool {
	b, _ := t.locateBucket(id)
	ktableCalls.Add(
		context.TODO(), 1,
		metric.WithAttributes(attribute.String("method", "remove")))
	return b.remove(id)
}

func (t *Table) Count() int {
	t.RLock()
	defer t.RUnlock()
	n := 0
	for _, bucket := range nonEmptyBuckets(t.root) {
		n += len(bucket.nodes)
	}
	return n
}

// Seen clears a node's questionable flag and updates contact time.
func (t *Table) SetSeen(id infohash.ID) {
	t.Lock()
	defer t.Unlock()
	t.seenNoLock(id)
}

func (t *Table) seenNoLock(id infohash.ID) {
	bucket, _ := t.locateBucket(id)
	bucket.update()
	if n := bucket.find(id); n != nil {
		if n.PingAttempts > 0 {
			t.log.Debug("reseting questionable status", "address", n.AddrPort)
		}
		n.PingAttempts = 0
		n.LastContact = time.Now()
	}
}

func (t *Table) GetClosest(target infohash.ID, limit int) []*Node {
	t.RLock()
	defer t.RUnlock()
	ktableCalls.Add(
		context.TODO(), 1,
		metric.WithAttributes(attribute.String("method", "closest")))
	bitIndex := 0
	buckets := []*bucket{t.root}
	nodes := make([]*Node, 0, limit)
	var bucket *bucket
	for len(buckets) > 0 && len(nodes) < limit {
		bucket, buckets = buckets[len(buckets)-1], buckets[:len(buckets)-1]
		if bucket.nodes == nil {
			near := bucket.nearChild(target, bitIndex)
			far := bucket.farChild(target, bitIndex)
			buckets = append(buckets, far, near)
			bitIndex++
		} else {
			nodes = append(nodes, bucket.nodes...)
		}
	}
	// We have less than requested
	if length := len(nodes); limit > length {
		limit = length
	}
	slices.SortFunc(nodes, func(a, b *Node) int {
		aDist := target.Xor(a.ID)
		bDist := target.Xor(b.ID)
		return bytes.Compare(aDist, bDist)
	})
	return nodes[:limit]
}

func (t *Table) GetStale(limit int, d time.Duration) []*Node {
	t.RLock()
	defer t.RUnlock()
	nodes := make([]*Node, 0)
	ktableCalls.Add(
		context.TODO(), 1,
		metric.WithAttributes(attribute.String("method", "stale")))
	buckets := nonEmptyBuckets(t.root)
	slices.SortFunc(buckets, func(a, b *bucket) int {
		// Oldest first
		return b.lastChanged.Compare(a.lastChanged)
	})
	for _, b := range buckets {
		for _, n := range b.nodes {
			if len(nodes) < limit && time.Since(n.LastContact) > d {
				nodes = append(nodes, n)
			}
		}
	}
	return nodes
}

// recursive
func nonEmptyBuckets(b *bucket) []*bucket {
	if b == nil {
		return nil
	}
	var out []*bucket
	if len(b.nodes) > 0 {
		out = append(out, b)
	}
	out = append(out, nonEmptyBuckets(b.left)...)
	out = append(out, nonEmptyBuckets(b.right)...)
	return out
}

func (t *Table) locateBucket(id infohash.ID) (bucket *bucket, bitIndex int) {
	bucket = t.root
	for bucket.nodes == nil {
		bucket = bucket.nearChild(id, bitIndex)
		bitIndex++
	}
	return
}

func newBucket() *bucket {
	return &bucket{
		nodes: make([]*Node, 0),
		//lastChanged: time.Now(),
	}
}

func (b *bucket) update() {
	b.lastChanged = time.Now()
}

func (b *bucket) add(n *Node) {
	b.nodes = append(b.nodes, n)
	b.update()
}

func (b *bucket) split(bitIndex int) {
	b.left = newBucket()
	b.right = newBucket()
	for _, n := range b.nodes {
		b.nearChild(n.ID, bitIndex).add(n)
	}
	b.nodes = nil
}

func (b *bucket) has(id infohash.ID) bool {
	return b.indexOf(id) >= 0
}

func (b *bucket) nearChild(id infohash.ID, bitIndex int) *bucket {
	bitIndexWithinByte := bitIndex % 8
	desiredByte := id[bitIndex/8]
	if desiredByte&(1<<(uint(7-bitIndexWithinByte))) == 1 {
		return b.right
	}
	return b.left
}

func (b *bucket) farChild(id infohash.ID, bitIndex int) *bucket {
	if c := b.nearChild(id, bitIndex); c == b.right {
		return b.left
	}
	return b.right
}

func (b *bucket) remove(id infohash.ID) bool {
	var out bool
	b.nodes = slices.DeleteFunc(b.nodes, func(n *Node) bool {
		out = n.ID.Equal(id)
		return out
	})
	return out
}

func (b *bucket) indexOf(id infohash.ID) int {
	for i, c := range b.nodes {
		if id.Equal(c.ID) {
			return i
		}
	}
	return -1
}

func (b *bucket) find(id infohash.ID) *Node {
	if index := b.indexOf(id); index > -1 {
		return b.nodes[index]
	}
	return nil
}

// mostQuestionable finds the node with the most ping attempts
func (b *bucket) mostQuestionable(attempts int) *Node {
	var out *Node
	for _, n := range b.nodes {
		if n.PingAttempts > attempts {
			out = n
			attempts = n.PingAttempts
		}
	}
	return out
}

func (t *Table) WriteTo(w io.Writer) (int64, error) {
	t.RLock()
	defer t.RUnlock()

	var c int64
	var n int
	var err error

	// Infohash first
	b, err := bencode.EncodeString(string(t.id[:]))
	if err != nil {
		return c, err
	}
	n, err = w.Write(b)
	if err != nil {
		return c, err
	}
	c += int64(n)
	t.log.Debug("exporting table", "id", t.id)

	// This should write n4 nodes correctly too
	var l int
	for _, bu := range nonEmptyBuckets(t.root) {
		for _, node := range bu.nodes {
			cni, err := node.MarshalBinary()
			if err != nil {
				return c, err
			}
			b, err := bencode.EncodeString(string(cni))
			if err != nil {
				return c, err
			}
			n, err = w.Write(b)
			if err != nil {
				return c, err
			}
			l += 1
			c += int64(n)
			t.log.Debug("exported node", "id", node.ID, "address", node.AddrPort)
		}
	}
	t.log.Info("exported table", "id", t.id, "count", l)
	return c, err
}

func (t *Table) ReadFrom(r io.Reader) (int64, error) {
	t.Lock()
	defer t.Unlock()

	br := bencode.NewReader(bufio.NewReader(r))
	var ih string
	if !br.ReadString(&ih) {
		return br.Count(), br.Err()
	}
	t.id = infohash.ID([]byte(ih))
	t.log.Debug("importing table", "id", t.id)
	var nodes []Node
	for br.Err() == nil {
		var cni string
		if br.ReadString(&cni) {
			var node Node
			if err := node.UnmarshalBinary([]byte(cni)); err != nil {
				return br.Count(), err
			}
			nodes = append(nodes, node)
			t.log.Debug("imported node", "id", node.ID, "address", node.AddrPort)
		}
	}
	for _, node := range nodes {
		t.addNoLock(&node)
	}
	t.log.Info("imported table", "id", t.id, "count", len(nodes))
	return br.Count(), nil
}
