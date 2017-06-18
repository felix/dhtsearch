package main

import (
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"sync/atomic"
)

func (d *DHTNode) newTransactionId() string {
	t := atomic.AddUint32(&d.tid, 1)
	t = t % math.MaxUint16
	return strconv.Itoa(int(t))
}

var handlers = map[string]func(*DHTNode, *net.UDPAddr, map[string]interface{}) bool{
	"q": handleRequest,
	"r": handleResponse,
	"e": handleError,
}

// makeQuery returns a query-formed data.
func makeQuery(t, q string, a map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": t,
		"y": "q",
		"q": q,
		"a": a,
	}
}

// makeResponse returns a response-formed data.
func makeResponse(t string, r map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"t": t,
		"y": "r",
		"r": r,
	}
}

// Parse a KRPC packet into a message
func (d *DHTNode) processPacket(p packet) {
	// Check max rather than blocking
	if len(d.workerTokens) >= Config.Advanced.MaxDhtWorkers {
		dhtPacketsDropped.Add(1)
		return
	}

	d.workerTokens <- struct{}{}

	go func() {
		dhtWorkers.Add(1)
		defer func() {
			<-d.workerTokens
			dhtWorkers.Add(-1)
		}()

		data, err := Decode(p.b)
		if err != nil {
			return
		}

		response, err := parseMessage(data)
		if err != nil {
			return
		}

		if f, ok := handlers[response["y"].(string)]; ok {
			f(d, &p.raddr, response)
		}
	}()
}

// parseKeys parses keys. It just wraps parseKey.
func parseKeys(data map[string]interface{}, pairs [][]string) error {
	for _, args := range pairs {
		key, t := args[0], args[1]
		if err := parseKey(data, key, t); err != nil {
			return err
		}
	}
	return nil
}

// parseKey parses the key in dict data. `t` is type of the keyed value.
// It's one of "int", "string", "map", "list".
func parseKey(data map[string]interface{}, key string, t string) error {
	val, ok := data[key]
	if !ok {
		return errors.New("lack of key")
	}

	switch t {
	case "string":
		_, ok = val.(string)
	case "int":
		_, ok = val.(int)
	case "map":
		_, ok = val.(map[string]interface{})
	case "list":
		_, ok = val.([]interface{})
	default:
		panic("invalid type")
	}

	if !ok {
		return errors.New("invalid key type")
	}

	return nil
}

// parseMessage parses the basic data received from udp.
// It returns a map value.
func parseMessage(data interface{}) (map[string]interface{}, error) {
	response, ok := data.(map[string]interface{})
	if !ok {
		return nil, errors.New("response is not dict")
	}

	if err := parseKeys(response, [][]string{{"t", "string"}, {"y", "string"}}); err != nil {
		return nil, err
	}

	return response, nil
}

func (d DHTNode) sendQuery(rn *remoteNode, qType string, a map[string]interface{}) {

	// Stop if sending to self
	if rn.id == d.id {
		return
	}

	t := d.newTransactionId()

	d.sendMsg(rn.address, makeQuery(t, qType, a))
}

// bencode data and send
func (d *DHTNode) sendMsg(raddr net.UDPAddr, data map[string]interface{}) {
	d.packetsOut <- packet{[]byte(Encode(data)), raddr}
}

// Swiped from nictuku
func compactNodeInfoToString(cni string) string {
	if len(cni) == 6 {
		return fmt.Sprintf("%d.%d.%d.%d:%d",
			cni[0], cni[1], cni[2], cni[3],
			(uint16(cni[4])<<8)|uint16(cni[5]))
	} else if len(cni) == 18 {
		b := []byte(cni[:16])
		return fmt.Sprintf("[%s]:%d",
			net.IP.String(b),
			(uint16(cni[16])<<8)|uint16(cni[17]))
	} else {
		return ""
	}
}

// handleRequest handles the requests received from udp.
func handleRequest(d *DHTNode, addr *net.UDPAddr, m map[string]interface{}) (success bool) {

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

	if d.id == id {
		return
	}

	if len(id) != 20 {
		//d.sendMsg(addr, makeError(t, protocolError, "invalid id"))
		return
	}

	var rn *remoteNode
	switch q {
	case "ping":
		rn = newRemoteNode(*addr, id)
		d.sendMsg(*addr, makeResponse(t, map[string]interface{}{
			"id": d.id,
		}))

	case "get_peers":
		if err := parseKey(a, "info_hash", "string"); err != nil {
			//d.sendMsg(addr, makeError(t, protocolError, err.Error()))
			return
		}
		rn = newRemoteNode(*addr, id)
		ih := a["info_hash"].(string)
		if Config.Debug {
			fmt.Printf("get_peers from %s for %x\n", rn.String(), ih)
		}

		if len(ih) != ihLength {
			//send(dht, addr, makeError(t, protocolError, "invalid info_hash"))
			return
		}

		// Crawling, we have no nodes
		d.sendMsg(*addr, makeResponse(t, map[string]interface{}{
			"id":    genNeighbour(d.id, ih),
			"token": ih[:2],
			"nodes": "",
		}))

	case "announce_peer":
		if err := parseKeys(a, [][]string{
			{"info_hash", "string"},
			{"port", "int"},
			{"token", "string"}}); err != nil {

			//d.sendMsg(addr, makeError(t, protocolError, err.Error()))
			return
		}

		ih := a["info_hash"].(string)
		rn = newRemoteNode(*addr, ih)
		if Config.Debug {
			fmt.Printf("announce_peer from %s for %x\n", rn.String(), ih)
		}

		// TODO
		if impliedPort, ok := a["implied_port"]; ok &&
			impliedPort.(int) != 0 {
			//port = addr.Port
		}
		// TODO do we reply?
		d.peerChan <- peer{*addr, ih}

	default:
		//d.sendMsg(addr, makeError(t, protocolError, "invalid q"))
		return
	}
	d.kTable.add(rn)
	return true
}

// handleResponse handles responses received from udp.
func handleResponse(d *DHTNode, addr *net.UDPAddr, m map[string]interface{}) (success bool) {

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
		d.processFindNodeResults(rn, nodes)
		return
	}

	// get_peers response
	if err := parseKey(r, "values", "list"); err == nil {
		values := r["values"].([]interface{})
		for _, v := range values {
			addr := compactNodeInfoToString(v.(string))
			if Config.Debug {
				fmt.Printf("Unhandled get_peer request %s\n", addr)
			}
			// TODO new peer
			// d.peersManager.Insert(ih, p)
		}
	}
	d.kTable.add(rn)
	return true
}

// handleError handles errors received from udp.
func handleError(d *DHTNode, addr *net.UDPAddr, m map[string]interface{}) (success bool) {

	if err := parseKey(m, "e", "list"); err != nil {
		return
	}

	if e := m["e"].([]interface{}); len(e) != 2 {
		return
	}
	if Config.Debug {
		fmt.Printf("Error packet from %s:%d\n", addr.IP.String(), addr.Port)
	}

	return true
}
