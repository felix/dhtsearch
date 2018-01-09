package crawler

import (
	"math"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/felix/logger"
)

var (
	routers = []string{
		"router.bittorrent.com:6881",
		"dht.transmissionbt.com:6881",
		"router.utorrent.com:6881",
	}
)

type dhtNode struct {
	id         string
	address    string
	port       int
	conn       *net.UDPConn
	pool       chan chan packet
	workers    int
	tid        uint32
	packetsOut chan packet
	peersOut   chan<- peer
	log        logger.Logger
	//table      routingTable
}

func (d *dhtNode) run() {
	listener, err := net.ListenPacket("udp4", d.address+":"+strconv.Itoa(d.port))
	if err != nil {
		d.log.Error("failed to listen", "error", err)
		return
	}
	d.conn = listener.(*net.UDPConn)
	d.port = d.conn.LocalAddr().(*net.UDPAddr).Port

	d.log.Info("listening", "address", d.address, "port", d.port)

	d.pool = make(chan chan packet)

	// Packets onto the network
	d.packetsOut = make(chan packet, 512)

	// Create a slab for allocation
	byteSlab := newSlab(8192, 10)

	rTable := newRoutingTable(d.id)

	// Start our workers
	for i := 0; i < d.workers; i++ {
		w := &dhtWorker{
			pool:       d.pool,
			packetsOut: d.packetsOut,
			peersOut:   d.peersOut,
			rTable:     rTable,
		}
	}

	// Start writing packets from channel to DHT
	go func() {
		var p packet
		for {
			select {
			case p = <-d.packetsOut:
				d.conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(UDPTimeout)))
				b, err := d.conn.WriteToUDP(p.b, &p.raddr)
				if err != nil {
					// TODO remove from routing or add to blacklist?
					d.log.Error("failed to write packet", "error", err)
				}
			}
		}
	}()

	// TODO configurable
	ticker := time.Tick(5 * time.Second)

	// Send packets from conn to workers
	for {
		b := byteSlab.Alloc()
		c, addr, err := d.conn.ReadFromUDP(b)
		if err != nil {
			d.log.Warn("read error", "error", err)
			continue
		}

		select {
		case pCh := <-d.pool:
			// Chop and send
			pCh <- packet{b[0:c], *addr}
			byteSlab.Free(b)

		case <-ticker:
			go func() {
				d.log.Debug("making neighbours")
				if rTable.isEmpty() {
					d.bootstrap()
				} else {
					for _, rn := range rTable.getNodes() {
						d.findNode(rn, rn.id)
					}
					rTable.refresh()
				}
			}()
		}
	}
	return
}

func (d *dhtNode) bootstrap() {
	d.log.Debug("bootstrapping")
	for _, s := range routers {
		addr, err := net.ResolveUDPAddr("udp4", s)
		if err != nil {
			d.log.Error("failed to parse bootstrap address", "error", err)
			return
		}
		rn := newRemoteNode(*addr, "")
		d.findNode(rn, "")
	}
}

func (d dhtNode) findNode(rn *remoteNode, target string) {
	var id string
	if target == "" {
		id = d.id
	} else {
		id = genNeighbour(d.id, target)
	}
	d.sendQuery(rn, "find_node", map[string]interface{}{
		"id":     id,
		"target": genInfoHash(),
	})
}

// ping sends ping query to the chan.
func (d *dhtNode) ping(rn *remoteNode) {
	d.sendQuery(rn, "ping", map[string]interface{}{
		"id": genNeighbour(d.id, rn.id),
	})
}

func (d dhtNode) sendQuery(rn *remoteNode, qType string, a map[string]interface{}) {

	// Stop if sending to self
	if rn.id == d.id {
		return
	}

	t := d.newTransactionId()

	d.sendMsg(rn.address, makeQuery(t, qType, a))
}

// bencode data and send
func (d *dhtNode) sendMsg(raddr net.UDPAddr, data map[string]interface{}) {
	d.packetsOut <- packet{[]byte(Encode(data)), raddr}
}

func (d *dhtNode) newTransactionId() string {
	t := atomic.AddUint32(&d.tid, 1)
	t = t % math.MaxUint16
	return strconv.Itoa(int(t))
}
