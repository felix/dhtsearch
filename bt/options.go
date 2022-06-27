package bt

import (
	"src.userspace.com.au/dhtsearch"
	"src.userspace.com.au/logger"
)

type Option func(*Worker) error

// SetOnNewTorrent sets the callback
func SetOnNewTorrent(f func(dhtsearch.Torrent)) Option {
	return func(w *Worker) error {
		w.OnNewTorrent = f
		return nil
	}
}

// SetOnBadPeer sets the callback
func SetOnBadPeer(f func(dhtsearch.Peer)) Option {
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

// SetLogger sets the logger
func SetLogger(l logger.Logger) Option {
	return func(w *Worker) error {
		w.log = l
		return nil
	}
}
