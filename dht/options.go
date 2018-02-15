package dht

import (
	"github.com/felix/logger"
)

type Option func(*Node) error

func SetOnAnnouncePeer(f func(Peer)) Option {
	return func(n *Node) error {
		n.OnAnnouncePeer = f
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
