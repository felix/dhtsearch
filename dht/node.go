package dht

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/felix/dhtsearch/bencode"
	"github.com/felix/logger"
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
	id         Infohash
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
	//table      routingTable

	// OnAnnoucePeer is called for each peer that announces itself
	OnAnnouncePeer func(p Peer)
}

// NewNode creates a new DHT node
func NewNode(opts ...Option) (n *Node, err error) {
	id := randomInfoHash()

	k, err := newRoutingTable(id, 2000)
	if err != nil {
		n.log.Error("failed to create routing table", "error", err)
		return nil, err
	}

	n = &Node{
		id:         id,
		family:     "udp4",
		port:       6881,
		udpTimeout: 10,
		rTable:     k,
		limiter:    rate.NewLimiter(rate.Limit(100000), 2000000),
		log:        logger.New(&logger.Options{Name: "dht"}),
	}

	// Set variadic options passed
	for _, option := range opts {
		err = option(n)
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
					n.findNode(rn, generateNeighbour(n.id, rn.id))
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
		rn := &remoteNode{address: addr}
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
			// TODO remove from routing or add to blacklist?
			// TODO reduce limit
			n.log.Warn("failed to write packet", "error", err)
		}
	}
}

func (n *Node) findNode(rn *remoteNode, id Infohash) {
	target := randomInfoHash()
	n.sendQuery(rn, "find_node", map[string]interface{}{
		"id":     string(id),
		"target": string(target),
	})
}

// ping sends ping query to the chan.
func (n *Node) ping(rn *remoteNode) {
	id := generateNeighbour(n.id, rn.id)
	n.sendQuery(rn, "ping", map[string]interface{}{
		"id": string(id),
	})
}

func (n *Node) sendQuery(rn *remoteNode, qType string, a map[string]interface{}) error {
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
	//fmt.Printf("sending %s to %s\n", qType, rn.String())
	n.packetsOut <- packet{
		data:  b,
		raddr: rn.address,
	}
	return nil
}

// Parse a KRPC packet into a message
func (n *Node) processPacket(p packet) {
	data, err := bencode.Decode(p.data)
	if err != nil {
		return
	}

	response, ok := data.(map[string]interface{})
	if !ok {
		n.log.Debug("failed to parse packet", "error", "response is not dict")
		return
	}

	if err := checkKeys(response, [][]string{{"t", "string"}, {"y", "string"}}); err != nil {
		n.log.Debug("failed to parse packet", "error", err)
		return
	}

	switch response["y"].(string) {
	case "q":
		err = n.handleRequest(p.raddr, response)
	case "r":
		err = n.handleResponse(p.raddr, response)
	case "e":
		n.handleError(p.raddr, response)
	default:
		n.log.Warn("missing request type")
		return
	}
	if err != nil {
		n.log.Warn("failed to process packet", "error", err)
	}
}

// bencode data and send
func (n *Node) queueMsg(rn remoteNode, data map[string]interface{}) error {
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

// handleRequest handles the requests received from udp.
func (n *Node) handleRequest(addr net.Addr, m map[string]interface{}) error {
	q, err := getStringKey(m, "q")
	if err != nil {
		return err
	}

	a, err := getMapKey(m, "a")
	if err != nil {
		return err
	}

	id, err := getStringKey(a, "id")
	if err != nil {
		return err
	}

	ih, err := InfohashFromString(id)
	if err != nil {
		return err
	}

	if n.id.Equal(*ih) {
		return nil
	}

	rn := &remoteNode{address: addr, id: *ih}

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
	r, err := getMapKey(m, "r")
	if err != nil {
		return err
	}
	id, err := getStringKey(r, "id")
	if err != nil {
		return err
	}
	ih, err := InfohashFromString(id)
	if err != nil {
		return err
	}

	rn := &remoteNode{address: addr, id: *ih}

	nodes, err := getStringKey(r, "nodes")
	// find_nodes/get_peers response with nodes
	if err == nil {
		n.onFindNodeResponse(*rn, m)
		n.processFindNodeResults(*rn, nodes)
		n.rTable.add(rn)
		return nil
	}

	values, err := getListKey(r, "values")
	// get_peers response
	if err == nil {
		n.log.Debug("get_peers response", "source", rn)
		for _, v := range values {
			addr := decodeCompactNodeAddr(v.(string))
			n.log.Debug("unhandled get_peer request", "addres", addr)

			// TODO new peer needs to be matched to previous get_peers request
			// n.peersManager.Insert(ih, p)
		}
		n.rTable.add(rn)
	}
	return nil
}

// handleError handles errors received from udp.
func (n *Node) handleError(addr net.Addr, m map[string]interface{}) bool {
	if err := checkKey(m, "e", "list"); err != nil {
		return false
	}

	e := m["e"].([]interface{})
	if len(e) != 2 {
		return false
	}
	code := e[0].(int64)
	msg := e[1].(string)
	n.log.Debug("error packet", "address", addr.String(), "code", code, "error", msg)

	return true
}

// Process another node's response to a find_node query.
func (n *Node) processFindNodeResults(rn remoteNode, nodeList string) {
	nodeLength := 26
	if n.family == "udp6" {
		nodeLength = 38
	}

	if len(nodeList)%nodeLength != 0 {
		n.log.Error("node list is wrong length", "length", len(nodeList))
		return
	}

	//fmt.Printf("%s sent %d nodes\n", rn.address.String(), len(nodeList)/nodeLength)

	// We got a byte array in groups of 26 or 38
	for i := 0; i < len(nodeList); i += nodeLength {
		id := nodeList[i : i+ihLength]
		addrStr := decodeCompactNodeAddr(nodeList[i+ihLength : i+nodeLength])

		ih, err := InfohashFromString(id)
		if err != nil {
			n.log.Warn("invalid infohash in node list")
			continue
		}

		addr, err := net.ResolveUDPAddr(n.family, addrStr)
		if err != nil || addr.Port == 0 {
			//n.log.Warn("unable to resolve", "address", addrStr, "error", err)
			continue
		}

		rn := &remoteNode{address: addr, id: *ih}
		n.rTable.add(rn)
	}
}
