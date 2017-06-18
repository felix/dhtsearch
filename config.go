package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

type config struct {
	BasePort    int
	Debug       bool
	Quiet       bool
	NoHttp      bool
	HttpAddress string
	Dsn         string
	NumNodes    int
	Tags        map[string]string
	SkipTags    []string
	Advanced    advancedConfig
}

type advancedConfig struct {
	RoutingTableSize int
	MaxBtWorkers     int
	MaxDhtWorkers    int
	PeerCacheSize    int
	UdpTimeout       int
	TcpTimeout       int
	SlabAllocations  int
}

func loadConfig() {
	if _, err := os.Stat("config.toml"); !os.IsNotExist(err) {
		md, err := toml.DecodeFile("config.toml", &Config)
		if err != nil {
			fmt.Printf("Failed to read configuration: %q\n", err)
			os.Exit(1)
		}

		if !md.IsDefined("SkipTags") {
			Config.SkipTags = []string{"adult"}
		}
	}

	flag.IntVar(&Config.BasePort, "base-port", 6881, "listen port (and first of multiple ports)")
	flag.IntVar(&Config.NumNodes, "num-nodes", 1, "number of nodes to start")
	flag.BoolVar(&Config.Debug, "debug", false, "provide debug output")
	flag.BoolVar(&Config.Quiet, "quiet", false, "log only errors")
	flag.StringVar(&Config.Dsn, "dsn", "postgres://dht:dht@localhost/dht?sslmode=disable", "Database DSN")
	flag.BoolVar(&Config.NoHttp, "no-http", false, "no HTTP service")
	flag.StringVar(&Config.HttpAddress, "http-address", "localhost:6880", "HTTP listen address:port")

	// Advanced
	flag.IntVar(&Config.Advanced.RoutingTableSize, "routing-table-size", 1000, "number of remote nodes in routing table")
	flag.IntVar(&Config.Advanced.MaxBtWorkers, "max-bt-workers", 256, "max number of BT workers")
	flag.IntVar(&Config.Advanced.MaxDhtWorkers, "max-dht-workers", 256, "max number of DHT workers")
	flag.IntVar(&Config.Advanced.PeerCacheSize, "peer-cache-size", 200, "memory cache of seen peers")
	flag.IntVar(&Config.Advanced.UdpTimeout, "udp-timeout", 10, "UDP timeout in seconds")
	flag.IntVar(&Config.Advanced.TcpTimeout, "tcp-timeout", 10, "TCP timeout in seconds")
	flag.IntVar(&Config.Advanced.SlabAllocations, "slab-allocations", 10, "number of memory blocks to allocate for DHT client")

	flag.Parse()

	initTagRegexps()

	if !Config.Quiet {
		fmt.Printf("Skipping tags: %q\n", Config.SkipTags)
	}
}
