package dht

import (
	lru "github.com/hashicorp/golang-lru"
	"src.userspace.com.au/dhtsearch"
	"src.userspace.com.au/logger"
)

type Option func(*Node) error

func SetOnAnnouncePeer(f func(dhtsearch.Peer)) Option {
	return func(n *Node) error {
		n.OnAnnouncePeer = f
		return nil
	}
}

func SetOnBadPeer(f func(dhtsearch.Peer)) Option {
	return func(n *Node) error {
		n.OnBadPeer = f
		return nil
	}
}

// SetAddress sets the IP address to listen on
func SetAddress(ip string) Option {
	return func(n *Node) error {
		n.address = ip
		return nil
	}
}

// SetPort sets the port to listen on
func SetPort(p int) Option {
	return func(n *Node) error {
		n.port = p
		return nil
	}
}

// SetIPv6 enables IPv6
func SetIPv6(b bool) Option {
	return func(n *Node) error {
		if b {
			n.family = "udp6"
		}
		return nil
	}
}

// SetUDPTimeout sets the number of seconds to wait for UDP connections
func SetUDPTimeout(s int) Option {
	return func(n *Node) error {
		n.udpTimeout = s
		return nil
	}
}

// SetLogger sets the logger
func SetLogger(l logger.Logger) Option {
	return func(n *Node) error {
		n.log = l
		return nil
	}
}

// SetBlacklist sets the size of the node blacklist
func SetBlacklist(bl *lru.ARCCache) Option {
	return func(n *Node) (err error) {
		n.blacklist = bl
		return err
	}
}
