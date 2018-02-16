package dht

import (
	"fmt"
	"net"
	"strings"
)

func (n *Node) onPingQuery(rn remoteNode, msg map[string]interface{}) {
	t := msg["t"].(string)
	//n.log.Debug("ping", "source", rn)
	n.queueMsg(rn, makeResponse(t, map[string]interface{}{
		"id": string(n.id),
	}))
}

func (n *Node) onGetPeersQuery(rn remoteNode, msg map[string]interface{}) {
	a := msg["a"].(map[string]interface{})
	if err := checkKey(a, "info_hash", "string"); err != nil {
		//n.queueMsg(addr, makeError(t, protocolError, err.Error()))
		return
	}

	// This is the ih of the torrent
	th, err := InfohashFromString(a["info_hash"].(string))
	if err != nil {
		n.log.Warn("invalid torrent", "infohash", a["info_hash"])
	}
	n.log.Debug("get_peers query", "source", rn, "torrent", th)

	token := []byte(*th)[:2]

	id := generateNeighbour(n.id, *th)
	nodes := n.rTable.get(8)
	compactNS := []string{}
	for _, rn := range nodes {
		ns := encodeCompactNodeAddr(rn.address.String())
		if ns == "" {
			n.log.Warn("failed to compact node", "address", rn.address.String())
			continue
		}
		compactNS = append(compactNS, ns)
	}

	t := msg["t"].(string)
	n.queueMsg(rn, makeResponse(t, map[string]interface{}{
		"id":    string(id),
		"token": token,
		"nodes": strings.Join(compactNS, ""),
	}))

	//nodes := n.rTable.get(50)
	/*
		fmt.Printf("sending get_peers for %s to %d nodes\n", *th, len(nodes))
		q := makeQuery(newTransactionID(), "get_peers", map[string]interface{}{
			"id":        string(id),
			"info_hash": string(*th),
		})
		for _, o := range nodes {
			n.queueMsg(*o, q)
		}
	*/
}

func (n *Node) onAnnouncePeerQuery(rn remoteNode, msg map[string]interface{}) {
	a := msg["a"].(map[string]interface{})
	err := checkKeys(a, [][]string{
		{"info_hash", "string"},
		{"port", "int"},
		{"token", "string"},
	})
	if err != nil {
		//n.queueMsg(addr, makeError(t, protocolError, err.Error()))
		return
	}

	n.log.Debug("announce_peer", "source", rn)

	// TODO
	if impliedPort, ok := a["implied_port"]; ok && impliedPort.(int) != 0 {
		// Use the port from the network
	} else {
		// Use the port in the message
		host, _, err := net.SplitHostPort(rn.address.String())
		if err != nil {
			n.log.Warn("failed to split host/port", "error", err)
			return
		}
		newPort := a["port"]
		if newPort == 0 {
			n.log.Warn("sent port 0", "source", rn)
			return
		}
		addr, err := net.ResolveUDPAddr(n.family, fmt.Sprintf("%s:%d", host, newPort))
		rn = remoteNode{address: addr, id: rn.id}
	}

	// TODO do we reply?

	ih, err := InfohashFromString(a["info_hash"].(string))
	if err != nil {
		n.log.Warn("invalid torrent", "infohash", a["info_hash"])
	}

	p := Peer{Node: rn, Infohash: *ih}
	n.log.Info("anounce_peer", p)
	if n.OnAnnouncePeer != nil {
		go n.OnAnnouncePeer(p)
	}
}

func (n *Node) onFindNodeResponse(rn remoteNode, msg map[string]interface{}) {
	r := msg["r"].(map[string]interface{})
	if err := checkKey(r, "id", "string"); err != nil {
		return
	}
	nodes := r["nodes"].(string)
	n.processFindNodeResults(rn, nodes)
}
