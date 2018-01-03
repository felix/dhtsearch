package dhtsearch

// Config -
type Config struct {
	BasePort    int               `toml:"base_port"`
	Debug       bool              `toml:"debug"`
	NoHTTP      bool              `toml:"no_http"`
	HTTPAddress string            `toml:"http_address"`
	DSN         string            `toml:"dsn"`
	NumNodes    int               `toml:"num_nodes"`
	Tags        map[string]string `toml:"tags"`
	SkipTags    []string          `toml:"skip_tags"`

	// Advanced
	RoutingTableSize int `toml:"routing_table_size"`
	MaxBTWorkers     int `toml:"max_bt_workers"`
	MaxDHTWorkers    int `toml:"max_dht_workers"`
	PeerCacheSize    int `toml:"peer_cache_size"`
	UDPTimeout       int `toml:"udp_timeout"`
	TCPTimeout       int `toml:"tcp_timeout"`
	ResultsPageSize  int `toml:"results_page_size"`
}

// DefaultConfig sets the defaults
func DefaultConfig() Config {
	return Config{
		BasePort:         6881,
		NumNodes:         1,
		Debug:            false,
		Quiet:            false,
		DSN:              "postgres://dht:dht@localhost/dht?sslmode=disable",
		NoHTTP:           false,
		HTTPAddress:      "localhost:6880",
		RoutingTableSize: 4000,
		MaxBTWorkers:     10,
		MaxDHTWorkers:    256,
		PeerCacheSize:    200,
		UDPTimeout:       10,
		TCPTimeout:       10,
		ResultsPageSize:  50,
	}
}
