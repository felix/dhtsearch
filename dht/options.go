package dht

import (
	"github.com/felix/logger"
)

type Option func(*Node) error

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

// SetWorkers sets the number of workers
func SetWorkers(c int) Option {
	return func(n *Node) error {
		n.workers = make([]*dhtWorker, c)
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

// SetLogger sets the number of workers
func SetLogger(l logger.Logger) Option {
	return func(n *Node) error {
		n.log = l
		return nil
	}
}
