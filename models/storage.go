package models

import ()

type migratable interface {
	MigrateSchema() error
}

type torrentSearcher interface {
	TorrentsByHash(hash Infohash) (*Torrent, error)
	TorrentsByName(query string, offset, limit int) ([]*Torrent, error)
	TorrentsByTags(tags []string, offset, limit int) ([]*Torrent, error)
}

type PeerStore interface {
	SavePeer(*Peer) error
}

type TorrentStore interface {
	SaveTorrent(*Torrent) error
	// TODO
	RemovePeer(*Peer) error
}

type InfohashStore interface {
	PendingInfohashes(int) ([]*Peer, error)
}
