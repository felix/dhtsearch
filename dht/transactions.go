package dht

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"time"
)

type transactions struct {
	sync.Mutex
	ttl      time.Duration
	requests map[string]trans
	tid      uint64
	buf      [binary.MaxVarintLen64]byte
}

type trans struct {
	ap  netip.AddrPort
	msg string
	ts  time.Time
	cb  func(*Node)
}

func newTransRegistry() *transactions {
	return &transactions{
		ttl:      30 * time.Second,
		requests: make(map[string]trans),
	}
}

type TransactionCallback func(*Node)

func (c *Client) registerTransaction(rn *Node, m *Msg, cb TransactionCallback) {
	c.chk.Lock()
	defer c.chk.Unlock()
	c.chk.tid++

	n := binary.PutUvarint(c.chk.buf[:], c.chk.tid)
	m.TID = string(c.chk.buf[:n])

	if _, ok := c.chk.requests[m.TID]; ok {
		panic(fmt.Sprintf("duplicate transaction ID %q", m.TID))
	}
	c.chk.requests[m.TID] = trans{
		ap:  rn.AddrPort,
		ts:  time.Now(),
		msg: m.Query,
		cb:  cb,
	}
}

func (c *Client) checkTransaction(m Msg) (*trans, error) {
	c.chk.Lock()
	defer c.chk.Unlock()
	if t, ok := c.chk.requests[m.TID]; ok {
		delete(c.chk.requests, m.TID)
		return &t, nil
	}
	return nil, errors.New("unsolicited transaction")
}

// func (c *Client) inflightQueries() int {
// 	c.chk.Lock()
// 	defer c.chk.Unlock()
// 	return len(c.chk.requests)
// }

func (c *Client) auditTransactions(ctx context.Context) {
	c.log.Info("starting transaction auditing", "interval", c.tickAuditTransactions)
	ticker := time.Tick(c.tickAuditTransactions)
	for {
		select {
		case <-ctx.Done():
			c.log.Info("stopping transaction auditing")
			return
		case <-ticker:
			c.chk.Lock()
			before := len(c.chk.requests)
			for tid, t := range c.chk.requests {
				if time.Since(t.ts) > c.chk.ttl {
					delete(c.chk.requests, tid)
				}
			}
			after := len(c.chk.requests)
			c.chk.Unlock()
			c.log.Debug("cleared transactions", "size", after, "removed", before-after)
		}
	}
}
