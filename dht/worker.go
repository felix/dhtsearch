package dht

import (
	"net"

	"github.com/felix/dhtsearch/bencode"
	"github.com/felix/logger"
)

type dhtWorker struct {
	pool       chan chan packet
	packetsOut chan<- packet
	peersOut   chan<- Peer
	log        logger.Logger
	rTable     *routingTable
	quit       chan struct{}
}

func (dw *dhtWorker) run() error {
	packetsIn := make(chan packet)

	for {
		dw.pool <- packetsIn

		// Wait for work or shutdown
		select {
		case p := <-packetsIn:
			dw.process(p)
		case <-dw.quit:
			dw.log.Warn("worker closing")
			break
		}
	}
}

func (dw dhtWorker) stop() {
	go func() {
		dw.quit <- struct{}{}
	}()
}

// Parse a KRPC packet into a message
func (dw *dhtWorker) process(p packet) {
	data, err := bencode.Decode(p.data)
	if err != nil {
		return
	}

	response, err := parseMessage(data)
	if err != nil {
		dw.log.Debug("failed to parse packet", "error", err)
		return
	}

	switch response["y"].(string) {
	case "q":
		dw.handleRequest(&p.raddr, response)
	case "r":
		dw.handleResponse(&p.raddr, response)
	case "e":
		dw.handleError(&p.raddr, response)
	default:
		dw.log.Warn("missing request type")
		return
	}
}

// bencode data and send
func (dw *dhtWorker) queueMsg(raddr net.UDPAddr, data map[string]interface{}) error {
	b, err := bencode.Encode(data)
	if err != nil {
		return err
	}
	dw.packetsOut <- packet{
		data:  b,
		raddr: raddr,
	}
	return nil
}

// handleRequest handles the requests received from udp.
func (dw *dhtWorker) handleRequest(addr *net.UDPAddr, m map[string]interface{}) (success bool) {

	t := m["t"].(string)

	if err := checkKeys(m, [][]string{{"q", "string"}, {"a", "map"}}); err != nil {

		//d.queueMsg(addr, makeError(t, protocolError, err.Error()))
		return
	}

	q := m["q"].(string)
	a := m["a"].(map[string]interface{})

	if err := checkKey(a, "id", "string"); err != nil {
		//d.queueMsg(addr, makeError(t, protocolError, err.Error()))
		return
	}

	var ih Infohash
	err := ih.FromString(a["id"].(string))
	if err != nil {
		dw.log.Warn("invalid packet", "infohash", a["id"])
	}

	if dw.rTable.id.Equal(ih) {
		return
	}

	var rn *remoteNode
	switch q {
	case "ping":
		rn = &remoteNode{address: *addr, id: ih}
		dw.log.Debug("ping", "source", rn, "infohash", ih)
		dw.queueMsg(*addr, makeResponse(t, map[string]interface{}{
			"id": string(dw.rTable.id),
		}))

	case "get_peers":
		if err := checkKey(a, "info_hash", "string"); err != nil {
			//dw.queueMsg(addr, makeError(t, protocolError, err.Error()))
			return
		}
		rn = &remoteNode{address: *addr, id: ih}
		err = ih.FromString(a["info_hash"].(string))
		if err != nil {
			dw.log.Warn("invalid packet", "infohash", a["id"])
		}
		dw.log.Debug("get_peers", "source", rn, "infohash", ih)

		// Crawling, we have no nodes
		id := dw.rTable.id.GenNeighbour(ih)
		dw.queueMsg(*addr, makeResponse(t, map[string]interface{}{
			"id":    string(id),
			"token": ih[:2],
			"nodes": "",
		}))

	case "announce_peer":
		if err := checkKeys(a, [][]string{
			{"info_hash", "string"},
			{"port", "int"},
			{"token", "string"}}); err != nil {

			//dw.queueMsg(addr, makeError(t, protocolError, err.Error()))
			return
		}

		rn = &remoteNode{address: *addr, id: ih}
		dw.log.Debug("announce_peer", "source", rn, "infohash", ih)

		// TODO
		if impliedPort, ok := a["implied_port"]; ok &&
			impliedPort.(int) != 0 {
			//port = addr.Port
		}
		// TODO do we reply?
		dw.peersOut <- Peer{*addr, ih}

	default:
		//dw.queueMsg(addr, makeError(t, protocolError, "invalid q"))
		return
	}
	dw.rTable.add(rn)
	return true
}

// handleResponse handles responses received from udp.
func (dw *dhtWorker) handleResponse(addr *net.UDPAddr, m map[string]interface{}) (success bool) {

	//t := m["t"].(string)

	if err := checkKey(m, "r", "map"); err != nil {
		return
	}

	r := m["r"].(map[string]interface{})
	if err := checkKey(r, "id", "string"); err != nil {
		return
	}

	var ih Infohash
	ih.FromString(r["id"].(string))
	rn := &remoteNode{address: *addr, id: ih}

	// find_nodes response
	if err := checkKey(r, "nodes", "string"); err == nil {
		nodes := r["nodes"].(string)
		dw.processFindNodeResults(rn, nodes)
		return
	}

	// get_peers response
	if err := checkKey(r, "values", "list"); err == nil {
		values := r["values"].([]interface{})
		for _, v := range values {
			addr := compactNodeInfoToString(v.(string))
			dw.log.Debug("unhandled get_peer request", "addres", addr)
			// TODO new peer
			// dw.peersManager.Insert(ih, p)
		}
	}
	dw.rTable.add(rn)
	return true
}

// handleError handles errors received from udp.
func (dw *dhtWorker) handleError(addr *net.UDPAddr, m map[string]interface{}) bool {
	if err := checkKey(m, "e", "list"); err != nil {
		return false
	}

	e := m["e"].([]interface{})
	if len(e) != 2 {
		return false
	}
	code := e[0].(int64)
	msg := e[1].(string)
	dw.log.Debug("error packet", "ip", addr.IP.String(), "port", addr.Port, "code", code, "error", msg)

	return true
}

// Process another node's response to a find_node query.
func (dw *dhtWorker) processFindNodeResults(rn *remoteNode, nodeList string) {
	nodeLength := 26
	/*
		if d.config.proto == "udp6" {
			nodeList = m.R.Nodes6
			nodeLength = 38
		} else {
			nodeList = m.R.Nodes
		}

		// Not much to do
		if nodeList == "" {
			return
		}
	*/

	if len(nodeList)%nodeLength != 0 {
		dw.log.Error("node list is wrong length", "length", len(nodeList))
		return
	}

	var ih Infohash
	var err error

	//dw.log.Debug("got node list", "length", len(nodeList))

	// We got a byte array in groups of 26 or 38
	for i := 0; i < len(nodeList); i += nodeLength {
		id := nodeList[i : i+ihLength]
		addr := compactNodeInfoToString(nodeList[i+ihLength : i+nodeLength])

		err = ih.FromString(id)
		if err != nil {
			dw.log.Warn("invalid node list")
			continue
		}

		if dw.rTable.id.Equal(ih) {
			continue
		}

		address, err := net.ResolveUDPAddr("udp4", addr)
		if err != nil {
			dw.log.Error("failed to resolve", "error", err)
			continue
		}
		rn := &remoteNode{address: *address, id: ih}
		dw.rTable.add(rn)
	}
}
