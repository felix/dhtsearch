package main

import (
	"flag"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/felix/dhtsearch"
	"github.com/felix/logger"
	"github.com/jawher/mow.cli"
	"os"
	"path/filepath"
	"time"
)

var (
	cfg     dhtsearch.Config
	log     logger.Logger
	version string
	debug   *bool
)

func main() {

	app := cli.App("dhtsearch", "Crawl the DHT network")

	cfg = dhtsearch.DefaultConfig()

	debug = app.Bool(cli.BoolOpt{Name: "d debug", Value: cfg.Debug, Desc: "show debug output"})
	cfgFile := app.String(cli.StringOpt{Name: "c config", Value: "./config.toml", Desc: "path to configuration file", EnvVar: "CONFIG"})

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

	app.Version("version", version)

	app.Before = func() {
		cfgPath, _ := filepath.Abs(*cfgFile)
		if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
			if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
				fmt.Printf("failed to read configuration file: %s", err)
				cli.Exit(1)
			}
		}

		logOpts := &logger.Options{
			Name:  "dhtsearch",
			Level: logger.Info,
		}

		err := envconfig.Process("", &cfg)
		if err != nil {
			fmt.Printf("failed to parse environment: %s", err)
			cli.Exit(1)
		}

		if *debug {
			cfg.Debug = true
			logOpts.Level = logger.Debug
		}
		log = logger.New(logOpts)
		log.Info("version", version)
	}

	app.Action = func() {
		// Leave it to the user to get around this
		if len(cfg.SkipTags) == 0 {
			fmt.Println("Using default skip tags")
			cfg.SkipTags = []string{"adult"}
		}

		server, err := dhtsearch.NewServer(
			cfg,
			dhtsearch.SetLogger(log),
		)
		if err != nil {
			cli.Exit(1)
		}

		if cfg.StatFreq > 0 {
			statLog := log.Named("stats")
			go func() {
				// Grab the initial stats
				prev := loader.Stats()

				for {
					time.Sleep(time.Duration(cfg.StatFreq) * time.Second)
					stats := loader.Stats()
					diff := stats.Sub(&prev)
					statLog.Info(fmt.Sprintf("%ds diff", cfg.StatFreq), "messages_in", diff.MessagesIn, "bytes_in", diff.BytesIn, "messages_out", diff.MessagesOut, "bytes_out", diff.BytesOut)
					prev = stats
				}
			}()
		}

		err = server.Run()
		if err != nil {
			log.Error("failed to run loader", "error", err)
			cli.Exit(1)
		}
		cli.Exit(0)
	}

	app.Run(os.Args)
}
