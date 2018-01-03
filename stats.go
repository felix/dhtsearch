package dhtsearch

type Stats struct {
	DHTPacketsIn      int `json:"dht_packets_in"`
	DHTPacketsOut     int `json:"dht_packets_out"`
	DHTPacketsDropped int `json:"dht_packets_dropped"`
	DHTErrors         int `json:"dht_errors"`
	DHTCachedPeers    int `json:"dht_cached_peers"`
	DHTBytesIn        int `json:"dht_bytes_in"`
	DHTBytesOut       int `json:"dht_bytes_out"`
	DHTWorkers        int `json:"dht_workers"`
	BTBytesInt        int `json:"bt_bytes_int"`
	BTBytesOut        int `json:"bt_bytes_out"`
	BTWorkers         int `json:"bt_workers"`
	PeersAnnounced    int `json:"peers_announced"`
	PeersSkipped      int `json:"peers_skipped"`
	TorrentsSkipped   int `json:"torrents_skipped"`
	TorrentsSaved     int `json:"torrents_saved"`
	TorrentsTotal     int `json:"torrents_total"`
}

func (s *Stats) Sub(other *Stats) Stats {
	if other == nil {
		return *s
	}
	var diff Stats
	diff.MessagesIn = s.MessagesIn - other.MessagesIn
	diff.BytesIn = s.BytesIn - other.BytesIn
	diff.MessagesOut = s.MessagesOut - other.MessagesOut
	diff.BytesOut = s.BytesOut - other.BytesOut
	return diff
}
