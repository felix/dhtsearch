package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
)

type config struct {
	BasePort    int `toml:"base-port"`
	Debug       bool
	Quiet       bool
	NoHttp      bool   `toml:"no-http"`
	HttpAddress string `toml:"http-address"`
	Dsn         string
	NumNodes    int `toml:"num-nodes"`
	Tags        map[string]string
	SkipTags    []string `toml:"skip-tags"`
	Advanced    advancedConfig
}

type advancedConfig struct {
	RoutingTableSize int `toml:"routing-table-size"`
	MaxBtWorkers     int `toml:"max-bt-workers"`
	MaxDhtWorkers    int `toml:"max-dht-workers"`
	PeerCacheSize    int `toml:"peer-cache-size"`
	UdpTimeout       int `toml:"udp-timeout"`
	TcpTimeout       int `toml:"tcp-timeout"`
	ResultsPageSize  int `toml:"results-page-size"`
}

// Global
var Config config

func loadConfig() {
	// Defaults
	Config = config{
		BasePort:    6881,
		NumNodes:    1,
		Debug:       false,
		Quiet:       false,
		Dsn:         "postgres://dht:dht@localhost/dht?sslmode=disable",
		NoHttp:      false,
		HttpAddress: "localhost:6880",
		Advanced: advancedConfig{
			RoutingTableSize: 4000,
			MaxBtWorkers:     256,
			MaxDhtWorkers:    256,
			PeerCacheSize:    200,
			UdpTimeout:       10,
			TcpTimeout:       10,
			ResultsPageSize:  50,
		},
	}
	/*
			ex, err := os.Executable()
		    if err != nil {
				fmt.Printf("Failed to get executable: %q\n", err)
				os.Exit(1)
		    }
		    exPath := path.Dir(ex)
	*/

	flag.IntVar(&Config.BasePort, "base-port", Config.BasePort, "listen port (and first of multiple ports)")
	flag.IntVar(&Config.NumNodes, "num-nodes", Config.NumNodes, "number of nodes to start")
	flag.BoolVar(&Config.Debug, "debug", Config.Debug, "provide debug output")
	flag.BoolVar(&Config.Quiet, "quiet", Config.Quiet, "log only errors")
	flag.StringVar(&Config.Dsn, "dsn", Config.Dsn, "Database DSN")
	flag.BoolVar(&Config.NoHttp, "no-http", Config.NoHttp, "no HTTP service")
	flag.StringVar(&Config.HttpAddress, "http-address", Config.HttpAddress, "HTTP listen address:port")

	// Advanced
	flag.IntVar(&Config.Advanced.RoutingTableSize, "routing-table-size", Config.Advanced.RoutingTableSize, "number of remote nodes in routing table")
	flag.IntVar(&Config.Advanced.MaxBtWorkers, "max-bt-workers", Config.Advanced.MaxBtWorkers, "max number of BT workers")
	flag.IntVar(&Config.Advanced.MaxDhtWorkers, "max-dht-workers", Config.Advanced.MaxDhtWorkers, "max number of DHT workers")
	flag.IntVar(&Config.Advanced.PeerCacheSize, "peer-cache-size", Config.Advanced.PeerCacheSize, "memory cache of seen peers")
	flag.IntVar(&Config.Advanced.UdpTimeout, "udp-timeout", Config.Advanced.UdpTimeout, "UDP timeout in seconds")
	flag.IntVar(&Config.Advanced.TcpTimeout, "tcp-timeout", Config.Advanced.TcpTimeout, "TCP timeout in seconds")
	flag.IntVar(&Config.Advanced.ResultsPageSize, "results-page-size", Config.Advanced.ResultsPageSize, "number of items per page")

	cfgPath, _ := filepath.Abs("./config.toml")
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		// fmt.Printf("Using configuration from %s\n", cfgPath)
		md, err := toml.DecodeFile(cfgPath, &Config)
		if err != nil {
			fmt.Printf("Failed to read configuration: %q\n", err)
			os.Exit(1)
		}
		if len(md.Undecoded()) > 0 {
			fmt.Printf("Extraneous configuration keys: %q\n", md.Undecoded())
		}
	}

	// Leave it to the user to get around this
	if len(Config.SkipTags) == 0 {
		fmt.Println("Using default skip tags")
		Config.SkipTags = []string{"adult"}
	}

	flag.Parse()

	initTagRegexps()

	if !Config.Quiet {
		fmt.Printf("Skipping tags: %q\n", Config.SkipTags)
	}
}
