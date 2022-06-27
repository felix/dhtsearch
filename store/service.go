package store

import "src.userspace.com.au/dhtsearch"

type migratable interface {
	MigrateSchema() error
}

type torrentSearcher interface {
	TorrentsByHash(hash dhtsearch.Infohash) (*dhtsearch.Torrent, error)
	TorrentsByName(query string, offset, limit int) ([]*dhtsearch.Torrent, error)
	TorrentsByTags(tags []string, offset, limit int) ([]*dhtsearch.Torrent, error)
}

type PeerStore interface {
	SavePeer(*dhtsearch.Peer) error
	RemovePeer(*dhtsearch.Peer) error
}

type TorrentStore interface {
	SaveTorrent(*dhtsearch.Torrent) error
	RemoveTorrent(*dhtsearch.Torrent) error
	RemovePeer(*dhtsearch.Peer) error
}

type InfohashStore interface {
	PendingInfohashes(int) ([]*dhtsearch.Peer, error)
}
