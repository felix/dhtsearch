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
	RemovePeer(*Peer) error
}

type TorrentStore interface {
	SaveTorrent(*Torrent) error
	RemoveTorrent(*Torrent) error
}

type InfohashStore interface {
	PendingInfohashes(int) ([]*Peer, error)
}
