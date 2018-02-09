package dht

import (
	"net"
	"strconv"
	"time"

	"github.com/felix/dhtsearch/bencode"
	"github.com/felix/logger"
)

var (
	routers = []string{
		"router.bittorrent.com:6881",
		"dht.transmissionbt.com:6881",
		"router.utorrent.com:6881",
		"dht.aelitis.com:6881",
	}
)

// Node joins the DHT network
type Node struct {
	id         Infohash
	address    string
	port       int
	conn       *net.UDPConn
	pool       chan chan packet
	rTable     *routingTable
	workers    []*dhtWorker
	udpTimeout int
	packetsOut chan packet
	peersOut   chan Peer
	closing    chan chan error
	log        logger.Logger
	//table      routingTable

	// OnAnnoucePeer is called for each peer that announces itself
	OnAnnoucePeer func(p *Peer)
}

// NewNode creates a new DHT node
func NewNode(opts ...Option) (n *Node, err error) {

	id := randomInfoHash()

	n = &Node{
		id:       id,
		address:  "0.0.0.0",
		port:     6881,
		rTable:   newRoutingTable(id),
		workers:  make([]*dhtWorker, 1),
		closing:  make(chan chan error),
		log:      logger.New(&logger.Options{Name: "dht"}),
		peersOut: make(chan Peer),
	}

	// Set variadic options passed
	for _, option := range opts {
		err = option(n)
		if err != nil {
			return nil, err
		}
	}

	return n, nil
}

// Close stuff
func (n *Node) Close() error {
	n.log.Warn("node closing")
	errCh := make(chan error)
	n.closing <- errCh
	// Signal workers
	for _, w := range n.workers {
		w.stop()
	}
	return <-errCh
}

// Run starts the node on the DHT
func (n *Node) Run() chan Peer {
	listener, err := net.ListenPacket("udp4", n.address+":"+strconv.Itoa(n.port))
	if err != nil {
		n.log.Error("failed to listen", "error", err)
		return nil
	}
	n.conn = listener.(*net.UDPConn)
	n.port = n.conn.LocalAddr().(*net.UDPAddr).Port
	n.log.Info("listening", "id", n.id, "address", n.address, "port", n.port)

	// Worker pool
	n.pool = make(chan chan packet)
	// Packets onto the network
	n.packetsOut = make(chan packet, 512)

	// Create a slab for allocation
	byteSlab := newSlab(8192, 10)

	// Start our workers
	n.log.Debug("starting workers", "count", len(n.workers))
	for i := 0; i < len(n.workers); i++ {
		w := &dhtWorker{
			pool:       n.pool,
			packetsOut: n.packetsOut,
			peersOut:   n.peersOut,
			rTable:     n.rTable,
			quit:       make(chan struct{}),
			log:        n.log.Named("worker"),
		}
		go w.run()
		n.workers[i] = w
	}

	n.log.Debug("starting packet writer")
	// Start writing packets from channel to DHT
	go func() {
		var p packet
		for {
			select {
			case p = <-n.packetsOut:
				//n.conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(n.udpTimeout)))
				_, err := n.conn.WriteToUDP(p.data, &p.raddr)
				if err != nil {
					// TODO remove from routing or add to blacklist?
					n.log.Warn("failed to write packet", "error", err)
				}
			}
		}
	}()

	n.log.Debug("starting packet reader")
	// Start reading packets
	go func() {
		n.bootstrap()

		// TODO configurable
		ticker := time.Tick(10 * time.Second)

		// Send packets from conn to workers
		for {
			select {
			case errCh := <-n.closing:
				// TODO
				errCh <- nil
			case pCh := <-n.pool:
				go func() {
					b := byteSlab.Alloc()
					c, addr, err := n.conn.ReadFromUDP(b)
					if err != nil {
						n.log.Warn("UDP read error", "error", err)
						return
					}

					// Chop and send
					pCh <- packet{
						data:  b[0:c],
						raddr: *addr,
					}
					byteSlab.Free(b)
				}()

			case <-ticker:
				go func() {
					if n.rTable.isEmpty() {
						n.bootstrap()
					} else {
						n.makeNeighbours()
					}
				}()
			}
		}
	}()
	return n.peersOut
}

func (n *Node) bootstrap() {
	n.log.Debug("bootstrapping")
	for _, s := range routers {
		addr, err := net.ResolveUDPAddr("udp4", s)
		if err != nil {
			n.log.Error("failed to parse bootstrap address", "error", err)
			return
		}
		rn := &remoteNode{address: *addr}
		n.findNode(rn, n.id)
	}
}

func (n *Node) makeNeighbours() {
	n.log.Debug("making neighbours")
	for _, rn := range n.rTable.getNodes() {
		n.findNode(rn, n.id)
	}
	n.rTable.refresh()
}

func (n Node) findNode(rn *remoteNode, id Infohash) {
	target := randomInfoHash()
	n.sendMsg(rn, "find_node", map[string]interface{}{
		"id":     string(id),
		"target": string(target),
	})
}

// ping sends ping query to the chan.
func (n *Node) ping(rn *remoteNode) {
	id := n.id.GenNeighbour(rn.id)
	n.sendMsg(rn, "ping", map[string]interface{}{
		"id": string(id),
	})
}

func (n Node) sendMsg(rn *remoteNode, qType string, a map[string]interface{}) error {
	// Stop if sending to self
	if rn.id.Equal(n.id) {
		return nil
	}

	t := newTransactionID()
	//n.log.Debug("sending message", "type", qType, "remote", rn)

	data := makeQuery(t, qType, a)
	b, err := bencode.Encode(data)
	if err != nil {
		return err
	}
	n.packetsOut <- packet{
		data:  b,
		raddr: rn.address,
	}
	return nil
}
