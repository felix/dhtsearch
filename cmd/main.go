package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felix/dhtsearch"
	"github.com/felix/dhtsearch/crawler"
	"github.com/felix/logger"
	"github.com/jawher/mow.cli"
)

var (
	version string
)

func main() {
	app := cli.App("dhtsearch", "Crawl the DHT network")

	app.Version("version", version)

	var (
		port = app.Int(cli.IntOpt{
			Name:   "p port",
			Value:  6881,
			EnvVar: "PORT",
			Desc:   "listen port (and first of multiple ports)",
		})
		nodes = app.Int(cli.IntOpt{
			Name:   "n nodes",
			Vale:   1,
			EnvVar: "NODES",
			Desc:   "number of nodes to start",
		})
		debug = app.Bool(cli.BoolOpt{
			Name:   "d debug",
			Value:  false,
			EnvVar: "DEBUG",
			Desc:   "show debug output",
		})
		dsn = app.String(cli.StringOpt{
			Name:   "dsn",
			Value:  "postgres://dht:dht@localhost/dht?sslmode=disable",
			EnvVar: "DSN",
			Desc:   "postgres DSN",
		})
		httpHost = app.String(cli.StringOpt{
			Name:   "http-host",
			Value:  "localhost:6880",
			EnvVar: "HTTP_HOST",
			Desc:   "HTTP listen address",
		})
		httpPort = app.Int(cli.IntOpt{
			Name:   "http-port",
			Value:  6880,
			EnvVar: "HTTP_PORT",
			Desc:   "HTTP listen port",
		})
		httpOff = app.Bool(cli.BoolOpt{
			Name:   "disable-http",
			Value:  false,
			EnvVar: "DISABLE_HTTP",
			Desc:   "disable HTTP",
		})
		filterOff = app.Bool(cli.BoolOpt{
			Name:   "disable-filter",
			Value:  false,
			EnvVar: "DISABLE_FILTER",
			Desc:   "disable HTTP",
		})
		tagFile = app.String(cli.StringOpt{
			Name:   "tag-file",
			Value:  "",
			EnvVar: "TAG_FILE",
			Desc:   "file containingn custom tags",
		})
		skipTags = app.String(cli.StringOpt{
			Name:   "skip-tags",
			Value:  "adult",
			EnvVar: "SKIP_TAGS",
			Desc:   "comma separated list of tags to skip",
		})
	)

	app.Action = func() {
		logOpts := &logger.Options{
			Name:  "dhtsearch",
			Level: logger.Info,
		}

		if debug {
			logOpts.Level = logger.Debug
		}
		log = logger.New(logOpts)
		log.Info("version", version)

		crawler, err := crawler.New(
			dsn,
			dhtsearch.SetLogger(log),
			dhtsearch.SetPort(port),
			dhtsearch.SetNodes(nodes),
		)
		if err != nil {
			log.Error("failed to create crawler", "error", err)
			cli.Exit(1)
		}

		err = server.Run()
		if err != nil {
			log.Error("failed to run c", "error", err)
			cli.Exit(1)
		}
		cli.Exit(0)
	}

	app.Run(os.Args)
}
