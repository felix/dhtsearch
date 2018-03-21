package dht

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/felix/dhtsearch/bencode"
	"github.com/felix/dhtsearch/krpc"
	"github.com/felix/dhtsearch/models"
	"github.com/felix/logger"
	"github.com/hashicorp/golang-lru"
	"golang.org/x/time/rate"
)

var (
	routers = []string{
		"dht.libtorrent.org:25401",
		"router.bittorrent.com:6881",
		"dht.transmissionbt.com:6881",
		"router.utorrent.com:6881",
		"dht.aelitis.com:6881",
	}
)

// Node joins the DHT network
type Node struct {
	id         models.Infohash
	family     string
	address    string
	port       int
	conn       net.PacketConn
	pool       chan chan packet
	rTable     *routingTable
	udpTimeout int
	packetsOut chan packet
	log        logger.Logger
	limiter    *rate.Limiter
	blacklist  *lru.ARCCache

	// OnAnnoucePeer is called for each peer that announces itself
	OnAnnouncePeer func(models.Peer)
	// OnBadPeer is called for each bad peer
	OnBadPeer func(models.Peer)
}

// NewNode creates a new DHT node
func NewNode(opts ...Option) (*Node, error) {
	var err error
	id := models.GenInfohash()

	n := &Node{
		id:         id,
		family:     "udp4",
		port:       6881,
		udpTimeout: 10,
		limiter:    rate.NewLimiter(rate.Limit(100000), 2000000),
		log:        logger.New(&logger.Options{Name: "dht"}),
	}

	n.rTable, err = newRoutingTable(id, 2000)
	if err != nil {
		n.log.Error("failed to create routing table", "error", err)
		return nil, err
	}

	// Set variadic options passed
	for _, option := range opts {
		err = option(n)
		if err != nil {
			return nil, err
		}
	}

	if n.blacklist == nil {
		n.blacklist, err = lru.NewARC(1000)
		if err != nil {
			return nil, err
		}
	}

	if n.family != "udp4" {
		n.log.Debug("trying udp6 server")
		n.conn, err = net.ListenPacket("udp6", fmt.Sprintf("[%s]:%d", net.IPv6zero.String(), n.port))
		if err == nil {
			n.family = "udp6"
		}
	}
	if n.conn == nil {
		n.conn, err = net.ListenPacket("udp4", fmt.Sprintf("%s:%d", net.IPv4zero.String(), n.port))
		if err == nil {
			n.family = "udp4"
		}
	}
	if err != nil {
		n.log.Error("failed to listen", "error", err)
		return nil, err
	}
	n.log.Info("listening", "id", n.id, "network", n.family, "address", n.conn.LocalAddr().String())

	return n, nil
}

// Close stuff
func (n *Node) Close() error {
	n.log.Warn("node closing")
	return nil
}

// Run starts the node on the DHT
func (n *Node) Run() {
	// Packets onto the network
	n.packetsOut = make(chan packet, 1024)

	// Create a slab for allocation
	byteSlab := newSlab(8192, 10)

	n.log.Debug("starting packet writer")
	go n.packetWriter()

	// Find neighbours
	go n.makeNeighbours()

	n.log.Debug("starting packet reader")
	for {
		b := byteSlab.alloc()
		c, addr, err := n.conn.ReadFrom(b)
		if err != nil {
			n.log.Warn("UDP read error", "error", err)
			return
		}

		// Chop and process
		n.processPacket(packet{
			data:  b[0:c],
			raddr: addr,
		})
		byteSlab.free(b)
	}
}

func (n *Node) makeNeighbours() {
	// TODO configurable
	ticker := time.Tick(5 * time.Second)

	n.bootstrap()

	for {
		select {
		case <-ticker:
			if n.rTable.isEmpty() {
				n.bootstrap()
			} else {
				// Send to all nodes
				nodes := n.rTable.get(0)
				for _, rn := range nodes {
					n.findNode(rn, models.GenerateNeighbour(n.id, rn.id))
				}
				n.rTable.flush()
			}
		}
	}
}

func (n *Node) bootstrap() {
	n.log.Debug("bootstrapping")
	for _, s := range routers {
		addr, err := net.ResolveUDPAddr(n.family, s)
		if err != nil {
			n.log.Error("failed to parse bootstrap address", "error", err)
			continue
		}
		rn := &remoteNode{addr: addr}
		n.findNode(rn, n.id)
	}
}

func (n *Node) packetWriter() {
	for p := range n.packetsOut {
		if p.raddr.String() == n.conn.LocalAddr().String() {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := n.limiter.WaitN(ctx, len(p.data)); err != nil {
			n.log.Warn("rate limited", "error", err)
			continue
		}
		//n.log.Debug("writing packet", "dest", p.raddr.String())
		_, err := n.conn.WriteTo(p.data, p.raddr)
		if err != nil {
			n.blacklist.Add(p.raddr.String(), true)
			// TODO reduce limit
			n.log.Warn("failed to write packet", "error", err)
			if n.OnBadPeer != nil {
				peer := models.Peer{Addr: p.raddr}
				go n.OnBadPeer(peer)
			}
		}
	}
}

func (n *Node) findNode(rn *remoteNode, id models.Infohash) {
	target := models.GenInfohash()
	n.sendQuery(rn, "find_node", map[string]interface{}{
		"id":     string(id),
		"target": string(target),
	})
}

// ping sends ping query to the chan.
func (n *Node) ping(rn *remoteNode) {
	id := models.GenerateNeighbour(n.id, rn.id)
	n.sendQuery(rn, "ping", map[string]interface{}{
		"id": string(id),
	})
}

func (n *Node) sendQuery(rn *remoteNode, qType string, a map[string]interface{}) error {
	// Stop if sending to self
	if rn.id.Equal(n.id) {
		return nil
	}

	t := krpc.NewTransactionID()

	data := krpc.MakeQuery(t, qType, a)
	b, err := bencode.Encode(data)
	if err != nil {
		return err
	}
	//fmt.Printf("sending %s to %s\n", qType, rn.String())
	n.packetsOut <- packet{
		data:  b,
		raddr: rn.addr,
	}
	return nil
}

// Parse a KRPC packet into a message
func (n *Node) processPacket(p packet) error {
	response, _, err := bencode.DecodeDict(p.data, 0)
	if err != nil {
		return err
	}

	y, err := krpc.GetString(response, "y")
	if err != nil {
		return err
	}

	if _, black := n.blacklist.Get(p.raddr.String()); black {
		return fmt.Errorf("blacklisted: %s", p.raddr.String())
	}

	switch y {
	case "q":
		err = n.handleRequest(p.raddr, response)
	case "r":
		err = n.handleResponse(p.raddr, response)
	case "e":
		err = n.handleError(p.raddr, response)
	default:
		err = fmt.Errorf("missing request type")
	}
	if err != nil {
		n.log.Warn("failed to process packet", "error", err)
		n.blacklist.Add(p.raddr.String(), true)
	}
	return err
}

// bencode data and send
func (n *Node) queueMsg(rn remoteNode, data map[string]interface{}) error {
	b, err := bencode.Encode(data)
	if err != nil {
		return err
	}
	n.packetsOut <- packet{
		data:  b,
		raddr: rn.addr,
	}
	return nil
}

// handleRequest handles the requests received from udp.
func (n *Node) handleRequest(addr net.Addr, m map[string]interface{}) error {
	q, err := krpc.GetString(m, "q")
	if err != nil {
		return err
	}

	a, err := krpc.GetMap(m, "a")
	if err != nil {
		return err
	}

	id, err := krpc.GetString(a, "id")
	if err != nil {
		return err
	}

	ih, err := models.InfohashFromString(id)
	if err != nil {
		return err
	}

	if n.id.Equal(*ih) {
		return nil
	}

	rn := &remoteNode{addr: addr, id: *ih}

	switch q {
	case "ping":
		err = n.onPingQuery(*rn, m)

	case "get_peers":
		err = n.onGetPeersQuery(*rn, m)

	case "announce_peer":
		n.onAnnouncePeerQuery(*rn, m)

	default:
		//n.queueMsg(addr, makeError(t, protocolError, "invalid q"))
		return nil
	}
	n.rTable.add(rn)
	return err
}

// handleResponse handles responses received from udp.
func (n *Node) handleResponse(addr net.Addr, m map[string]interface{}) error {
	r, err := krpc.GetMap(m, "r")
	if err != nil {
		return err
	}
	id, err := krpc.GetString(r, "id")
	if err != nil {
		return err
	}
	ih, err := models.InfohashFromString(id)
	if err != nil {
		return err
	}

	rn := &remoteNode{addr: addr, id: *ih}

	nodes, err := krpc.GetString(r, "nodes")
	// find_nodes/get_peers response with nodes
	if err == nil {
		n.onFindNodeResponse(*rn, m)
		n.processFindNodeResults(*rn, nodes)
		n.rTable.add(rn)
		return nil
	}

	values, err := krpc.GetList(r, "values")
	// get_peers response
	if err == nil {
		n.log.Debug("get_peers response", "source", rn)
		for _, v := range values {
			addr := krpc.DecodeCompactNodeAddr(v.(string))
			n.log.Debug("unhandled get_peer request", "addres", addr)

			// TODO new peer needs to be matched to previous get_peers request
			// n.peersManager.Insert(ih, p)
		}
		n.rTable.add(rn)
	}
	return nil
}

// handleError handles errors received from udp.
func (n *Node) handleError(addr net.Addr, m map[string]interface{}) error {
	e, err := krpc.GetList(m, "e")
	if err != nil {
		return err
	}

	if len(e) != 2 {
		return fmt.Errorf("error packet wrong length %d", len(e))
	}
	code := e[0].(int64)
	msg := e[1].(string)
	n.log.Debug("error packet", "address", addr.String(), "code", code, "error", msg)

	return nil
}

// Process another node's response to a find_node query.
func (n *Node) processFindNodeResults(rn remoteNode, nodeList string) {
	nodeLength := krpc.IPv4NodeAddrLen
	if n.family == "udp6" {
		nodeLength = krpc.IPv6NodeAddrLen
	}

	if len(nodeList)%nodeLength != 0 {
		n.log.Error("node list is wrong length", "length", len(nodeList))
		n.blacklist.Add(rn.addr.String(), true)
		return
	}

	//fmt.Printf("%s sent %d nodes\n", rn.address.String(), len(nodeList)/nodeLength)

	// We got a byte array in groups of 26 or 38
	for i := 0; i < len(nodeList); i += nodeLength {
		id := nodeList[i : i+models.InfohashLength]
		addrStr := krpc.DecodeCompactNodeAddr(nodeList[i+models.InfohashLength : i+nodeLength])

		ih, err := models.InfohashFromString(id)
		if err != nil {
			n.log.Warn("invalid infohash in node list")
			continue
		}

		addr, err := net.ResolveUDPAddr(n.family, addrStr)
		if err != nil || addr.Port == 0 {
			//n.log.Warn("unable to resolve", "address", addrStr, "error", err)
			continue
		}

		rn := &remoteNode{addr: addr, id: *ih}
		n.rTable.add(rn)
	}
}
