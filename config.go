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
	MaxBTWorkers    int `toml:"max_bt_workers"`
	MaxDHTWorkers   int `toml:"max_dht_workers"`
	PeerCacheSize   int `toml:"peer_cache_size"`
	ResultsPageSize int `toml:"results_page_size"`
}

// DefaultConfig sets the defaults
func DefaultConfig() Config {
	return Config{
		BasePort:        6881,
		NumNodes:        1,
		Debug:           false,
		DSN:             "postgres://dht:dht@localhost/dht?sslmode=disable",
		NoHTTP:          false,
		HTTPAddress:     "localhost:6880",
		MaxBTWorkers:    10,
		MaxDHTWorkers:   256,
		PeerCacheSize:   200,
		ResultsPageSize: 50,
	}
}
