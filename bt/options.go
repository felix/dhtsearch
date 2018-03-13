package bt

import (
	"github.com/felix/dhtsearch/models"
	"github.com/felix/logger"
)

type Option func(*Worker) error

// SetOnNewTorrent sets the callback
func SetOnNewTorrent(f func(models.Torrent)) Option {
	return func(w *Worker) error {
		w.OnNewTorrent = f
		return nil
	}
}

// SetOnBadPeer sets the callback
func SetOnBadPeer(f func(models.Peer)) Option {
	return func(w *Worker) error {
		w.OnBadPeer = f
		return nil
	}
}

// SetPort sets the port to listen on
func SetPort(p int) Option {
	return func(w *Worker) error {
		w.port = p
		return nil
	}
}

// SetIPv6 enables IPv6
func SetIPv6(b bool) Option {
	return func(w *Worker) error {
		if b {
			w.family = "tcp6"
		}
		return nil
	}
}

// SetUDPTimeout sets the number of seconds to wait for UDP connections
func SetTCPTimeout(s int) Option {
	return func(w *Worker) error {
		w.tcpTimeout = s
		return nil
	}
}

// SetLogger sets the logger
func SetLogger(l logger.Logger) Option {
	return func(w *Worker) error {
		w.log = l
		return nil
	}
}
