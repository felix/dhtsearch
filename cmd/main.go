package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/felix/dhtsearch/bt"
	"github.com/felix/dhtsearch/dht"
	"github.com/felix/dhtsearch/models"
	"github.com/felix/logger"
)

var (
	version string
	log     logger.Logger
)

// DHT vars
var (
	debug       bool
	port        int
	ipv6        bool
	dhtNodes    int
	showVersion bool
)

// Torrent vars
var (
	pool     chan chan models.Peer
	torrents chan models.Torrent
	btNodes  int
)

func main() {
	flag.IntVar(&port, "port", 6881, "listen port (and first for multiple nodes")
	flag.BoolVar(&debug, "debug", false, "show debug output")
	flag.BoolVar(&ipv6, "6", false, "listen on IPv6 also")
	flag.IntVar(&dhtNodes, "dht-nodes", 1, "number of DHT nodes to start")

	flag.IntVar(&btNodes, "bt-nodes", 3, "number of BT nodes to start")

	flag.BoolVar(&showVersion, "v", false, "show version")

	flag.Parse()

	if showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	logOpts := &logger.Options{
		Name:  "dhtsearch",
		Level: logger.Info,
	}

	if debug {
		logOpts.Level = logger.Debug
	}
	log = logger.New(logOpts)
	log.Info("version", version)
	log.Debug("debugging")

	go startDHTNodes(saveInfohash)

	go startBTWorkers(saveTorrent)

	for {
		select {
		case <-time.After(300 * time.Second):
			fmt.Println("mark")
		}
	}
}

func startDHTNodes(f func(p models.Peer)) (nodes []*dht.Node, err error) {
	nodes = make([]*dht.Node, dhtNodes)

	for i := 0; i < dhtNodes; i++ {
		dht, err := dht.NewNode(
			dht.SetLogger(log.Named("dht")),
			dht.SetPort(port+i),
			dht.SetOnAnnouncePeer(f),
			dht.SetIPv6(ipv6),
		)
		if err != nil {
			log.Error("failed to create node", "error", err)
			return nodes, err
		}
		go dht.Run()
		nodes[i] = dht
	}
	return nodes, err
}

func saveInfohash(p models.Peer) {
	//log.Info("announce", "peer", p)
	// Blocks
	w := <-pool
	w <- p
	return
}

func saveTorrent(t models.Torrent) {
	fmt.Printf("Torrent added, size: %d, name: %q, tags: %s, url: magnet:?xt=urn:btih:%s\n", t.Size, t.Name, t.Tags, t.Infohash)
	// Add tags
	//tagTorrent(&t)

	/*
		// Not sure if I like continue labels, so this
		var discard = false
		for _, tag := range Config.SkipTags {
			if hasTag(t, tag) {
				fmt.Printf("Skipping torrent tagged '%s': %q\n", tag, t.Name)
				discard = true
				break
			}
		}
		if discard {
			continue
		}
	*/
}

func startBTWorkers(f func(t models.Torrent)) {
	pool = make(chan chan models.Peer)
	torrents = make(chan models.Torrent)

	for i := 0; i < btNodes; i++ {
		w, err := bt.NewWorker(
			pool,
			bt.SetLogger(log.Named("bt")),
			bt.SetPort(port+i),
			bt.SetIPv6(ipv6),
			bt.SetOnNewTorrent(f),
		)
		if err != nil {
			log.Error("failed to create bt worker", "error", err)
			return
		}
		go w.Run()
	}
}
