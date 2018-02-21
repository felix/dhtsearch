package dht

import (
	"fmt"
	"net"

	"github.com/felix/dhtsearch/krpc"
)

func (n *Node) onPingQuery(rn remoteNode, msg map[string]interface{}) error {
	t, err := krpc.GetString(msg, "t")
	if err != nil {
		return err
	}
	n.queueMsg(rn, krpc.MakeResponse(t, map[string]interface{}{
		"id": string(n.id),
	}))
	return nil
}

func (n *Node) onGetPeersQuery(rn remoteNode, msg map[string]interface{}) error {
	a, err := krpc.GetMap(msg, "a")
	if err != nil {
		return err
	}

	// This is the ih of the torrent
	torrent, err := krpc.GetString(a, "info_hash")
	if err != nil {
		return err
	}
	th, err := InfohashFromString(torrent)
	if err != nil {
		return err
	}
	n.log.Debug("get_peers query", "source", rn, "torrent", th)

	token := torrent[:2]
	neighbour := generateNeighbour(n.id, *th)
	/*
		nodes := n.rTable.get(8)
		compactNS := []string{}
		for _, rn := range nodes {
			ns := encodeCompactNodeAddr(rn.addr.String())
			if ns == "" {
				n.log.Warn("failed to compact node", "address", rn.address.String())
				continue
			}
			compactNS = append(compactNS, ns)
		}
	*/

	t := msg["t"].(string)
	n.queueMsg(rn, krpc.MakeResponse(t, map[string]interface{}{
		"id":    string(neighbour),
		"token": token,
		"nodes": "",
		//"nodes": strings.Join(compactNS, ""),
	}))

	//nodes := n.rTable.get(50)
	/*
		fmt.Printf("sending get_peers for %s to %d nodes\n", *th, len(nodes))
		q := krpc.MakeQuery(newTransactionID(), "get_peers", map[string]interface{}{
			"id":        string(id),
			"info_hash": string(*th),
		})
		for _, o := range nodes {
			n.queueMsg(*o, q)
		}
	*/
	return nil
}

func (n *Node) onAnnouncePeerQuery(rn remoteNode, msg map[string]interface{}) error {
	a, err := krpc.GetMap(msg, "a")
	if err != nil {
		return err
	}

	n.log.Debug("announce_peer", "source", rn)

	if impliedPort, err := krpc.GetInt(a, "implied_port"); err == nil {
		if impliedPort != 0 {
			// Use the port in the message
			host, _, err := net.SplitHostPort(rn.addr.String())
			if err != nil {
				return err
			}
			newPort := a["port"]
			if newPort == 0 {
				return fmt.Errorf("ignoring port 0")
			}
			addr, err := net.ResolveUDPAddr(n.family, fmt.Sprintf("%s:%d", host, newPort))
			rn = remoteNode{addr: addr, id: rn.id}
		}
	}

	// TODO do we reply?

	ihStr, err := krpc.GetString(a, "info_hash")
	if err != nil {
		return err
	}
	ih, err := InfohashFromString(ihStr)
	if err != nil {
		n.log.Warn("invalid torrent", "infohash", ihStr)
	}

	p := Peer{Addr: rn.addr, ID: rn.id, Infohash: *ih}
	if n.OnAnnouncePeer != nil {
		go n.OnAnnouncePeer(p)
	}
	return nil
}

func (n *Node) onFindNodeResponse(rn remoteNode, msg map[string]interface{}) {
	r := msg["r"].(map[string]interface{})
	nodes := r["nodes"].(string)
	n.processFindNodeResults(rn, nodes)
}
