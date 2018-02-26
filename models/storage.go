package models

type torrentSearcher interface {
	torrentsByHash(hashes Infohash, offset, limit int) (*Torrent, error)
	torrentsByName(query string, offset, limit int) ([]*Torrent, error)
	torrentsByTags(tags []string, offset, limit int) ([]*Torrent, error)
}

type peerStore interface {
	savePeer(*Peer) error
}

type torrentStore interface {
	saveTorrent(*Torrent) error
}
