package dhtsearch

import (
	"net"

	"github.com/felix/logger"
)

type dhtWorker struct {
	pool       chan chan packet
	packetsOut chan<- packet
	peersOut   chan<- peer
	log        logger.Logger
	rTable     *routingTable
}

func (dw *dhtWorker) run(po chan<- packet) error {
	packetsIn := make(chan packet)
	dw.packetsOut = po

	for {
		dw.pool <- packetsIn

		select {
		// Wait for work
		case p := <-packetsIn:
			dw.process(p)
		}
	}
}

// Parse a KRPC packet into a message
func (dw *dhtWorker) process(p packet) {
	data, err := Decode(p.b)
	if err != nil {
		return
	}

	response, err := parseMessage(data)
	if err != nil {
		dw.log.Warn("failed to parse packet", "error", err)
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
func (dw *dhtWorker) sendMsg(raddr net.UDPAddr, data map[string]interface{}) {
	dw.packetsOut <- packet{[]byte(Encode(data)), raddr}
}

// handleRequest handles the requests received from udp.
func (dw *dhtWorker) handleRequest(addr *net.UDPAddr, m map[string]interface{}) (success bool) {

	t := m["t"].(string)

	if err := parseKeys(m, [][]string{{"q", "string"}, {"a", "map"}}); err != nil {

		//d.sendMsg(addr, makeError(t, protocolError, err.Error()))
		return
	}

	q := m["q"].(string)
	a := m["a"].(map[string]interface{})

	if err := parseKey(a, "id", "string"); err != nil {
		//d.sendMsg(addr, makeError(t, protocolError, err.Error()))
		return
	}

	id := a["id"].(string)

	if dw.rTable.id == id {
		return
	}

	if len(id) != 20 {
		//dw.sendMsg(addr, makeError(t, protocolError, "invalid id"))
		return
	}

	var rn *remoteNode
	switch q {
	case "ping":
		rn = newRemoteNode(*addr, id)
		dw.sendMsg(*addr, makeResponse(t, map[string]interface{}{
			"id": dw.rTable.id,
		}))

	case "get_peers":
		if err := parseKey(a, "info_hash", "string"); err != nil {
			//dw.sendMsg(addr, makeError(t, protocolError, err.Error()))
			return
		}
		rn = newRemoteNode(*addr, id)
		ih := a["info_hash"].(string)
		dw.log.Debug("get_peers", "source", rn.String(), "infohash", ih)

		if len(ih) != ihLength {
			//send(dht, addr, makeError(t, protocolError, "invalid info_hash"))
			return
		}

		// Crawling, we have no nodes
		dw.sendMsg(*addr, makeResponse(t, map[string]interface{}{
			"id":    genNeighbour(dw.rTable.id, ih),
			"token": ih[:2],
			"nodes": "",
		}))

	case "announce_peer":
		if err := parseKeys(a, [][]string{
			{"info_hash", "string"},
			{"port", "int"},
			{"token", "string"}}); err != nil {

			//dw.sendMsg(addr, makeError(t, protocolError, err.Error()))
			return
		}

		ih := a["info_hash"].(string)
		rn = newRemoteNode(*addr, ih)
		dw.log.Debug("announce_peer", "source", rn.String(), "infohash", ih)

		// TODO
		if impliedPort, ok := a["implied_port"]; ok &&
			impliedPort.(int) != 0 {
			//port = addr.Port
		}
		// TODO do we reply?
		dw.peersOut <- peer{*addr, ih}

	default:
		//dw.sendMsg(addr, makeError(t, protocolError, "invalid q"))
		return
	}
	dw.rTable.add(rn)
	return true
}

// handleResponse handles responses received from udp.
func (dw *dhtWorker) handleResponse(addr *net.UDPAddr, m map[string]interface{}) (success bool) {

	//t := m["t"].(string)

	// inform transManager to delete the transaction.
	if err := parseKey(m, "r", "map"); err != nil {
		return
	}

	r := m["r"].(map[string]interface{})
	if err := parseKey(r, "id", "string"); err != nil {
		return
	}

	ih := r["id"].(string)
	rn := newRemoteNode(*addr, ih)

	// find_nodes response
	if err := parseKey(r, "nodes", "string"); err == nil {
		nodes := r["nodes"].(string)
		dw.processFindNodeResults(rn, nodes)
		return
	}

	// get_peers response
	if err := parseKey(r, "values", "list"); err == nil {
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
func (dw *dhtWorker) handleError(addr *net.UDPAddr, m map[string]interface{}) (success bool) {
	if err := parseKey(m, "e", "list"); err != nil {
		return
	}

	if e := m["e"].([]interface{}); len(e) != 2 {
		return
	}
	dw.log.Debug("error packet", "ip", addr.IP.String(), "port", addr.Port)

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

	// We got a byte array in groups of 26 or 38
	for i := 0; i < len(nodeList); i += nodeLength {
		id := nodeList[i : i+ihLength]
		addr := compactNodeInfoToString(nodeList[i+ihLength : i+nodeLength])

		if dw.rTable.id == id {
			dw.log.Debug("find_nodes ignoring self")
			continue
		}

		address, err := net.ResolveUDPAddr("udp4", addr)
		if err != nil {
			dw.log.Error("failed to resolve", "error", err)
			continue
		}
		rn := newRemoteNode(*address, id)
		dw.rTable.add(rn)
	}
}
