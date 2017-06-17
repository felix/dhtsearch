package main

import (
	"expvar"
	"flag"
	"fmt"
	"time"
	//"github.com/pkg/profile"
	"net"
	"net/http"
	"os"
)

// Exported vars
var (
	dhtPacketsIn    = expvar.NewInt("dht_packets_in")
	dhtPacketsOut   = expvar.NewInt("dht_packets_out")
	dhtErrorPackets = expvar.NewInt("dht_error_packets")
	dhtBytesIn      = expvar.NewInt("dht_bytes_in")
	dhtBytesOut     = expvar.NewInt("dht_bytes_out")
	btBytesIn       = expvar.NewInt("bt_bytes_in")
	btBytesOut      = expvar.NewInt("bt_bytes_out")
	peersAnnounced  = expvar.NewInt("peers_announced")
	peersSkipped    = expvar.NewInt("peers_skipped")
	torrentsSkipped = expvar.NewInt("torrents_skipped")
	torrentsSaved   = expvar.NewInt("torrents_saved")
	torrentsTotal   = expvar.NewInt("torrents_total")
	start           = time.Now()
)

// Global
var DB *database

func uptime() interface{} {
	return int64(time.Since(start).Seconds())
}

func main() {
	//defer profile.Start(profile.CPUProfile).Stop()
	expvar.Publish("uptime", expvar.Func(uptime))

	var basePort, numNodes int
	var debug bool
	var noHttp bool
	var dsn, httpAddress string
	flag.IntVar(&basePort, "port", 6881, "listen port (and first of multiple ports)")
	flag.IntVar(&numNodes, "nodes", 1, "number of nodes to start")
	flag.BoolVar(&debug, "debug", false, "provide debug output")
	flag.StringVar(&dsn, "dsn", "postgres://dht:dht@localhost/dht?sslmode=disable", "DB DSN")
	flag.BoolVar(&noHttp, "no-http", false, "no HTTP service")
	flag.StringVar(&httpAddress, "http", "localhost:6880", "HTTP listen address:port")
	flag.Parse()

	initTagRegexps()

	// Slice of channels for DHT node output
	torrents := make(chan Torrent)
	peers := make(chan peer)

	// Close upstreams channels
	done := make(chan struct{})
	defer close(done)

	// Start DHTnodes
	if debug {
		fmt.Printf("Starting %d instance(s)\n", numNodes)
	}

	// Persistence
	var err error
	DB, err = newDB(dsn)
	if err != nil {
		os.Exit(1)
	}
	DB.debug = debug
	defer DB.Close()

	// Initialise tags
	for tag, _ := range tags {
		_, err := createTag(tag)
		if err != nil {
			fmt.Printf("Error creating tag %s: %q\n", tag, err)
		}
	}

	// Create DHT nodes
	for i := 0; i < numNodes; i++ {
		// Consecutive port numbers
		port := basePort + i
		dht := newDHTNode("", port, peers)
		dht.debug = debug
		err = dht.run(done)
		if err != nil {
			os.Exit(1)
		}
	}

	// Filter torrents
	filteredPeers := make(chan peer)

	// Create BT node
	btClient := newBTClient(filteredPeers, torrents)
	btClient.debug = debug
	err = btClient.run(done)
	if err != nil {
		os.Exit(1)
	}

	// HTTP Server
	if !noHttp {
		http.HandleFunc("/", indexHandler)
		http.HandleFunc("/stats", statsHandler)
		http.HandleFunc("/search", searchHandler)
		sock, _ := net.Listen("tcp", httpAddress)
		go func() {
			fmt.Printf("HTTP now available at %s\n", httpAddress)
			http.Serve(sock, nil)
		}()
	}

	// Simple cache of most recent
	cache := make(map[string]bool)
	var p peer
	var t Torrent

	for {
		select {
		case p = <-peers:
			if ok := cache[p.id]; ok {
				peersSkipped.Add(1)
				continue
			}
			peersAnnounced.Add(1)
			if len(cache) > 2000 {
				fmt.Printf("Flushing cache\n")
				cache = make(map[string]bool)
			}
			cache[p.id] = true
			if torrentExists(p.id) {
				peersSkipped.Add(1)
				continue
			}
			filteredPeers <- p

		case t = <-torrents:
			length := t.Size

			// Add tags
			tagTorrent(&t)

			var notWanted = false
			for _, tag := range t.Tags {
				if tag == "adult" {
					fmt.Printf("Skipping %s\n", t.Name)
					notWanted = true
				}
			}
			if notWanted {
				torrentsSkipped.Add(1)
				continue
			}

			fmt.Printf("Torrrent length: %d, name: %s, tags: %s, url: magnet:?xt=urn:btih:%s\n", length, t.Name, t.Tags, t.InfoHash)

			err := t.save()
			if err != nil {
				fmt.Printf("Error saving torrent: %q\n", err)
			}
			torrentsSaved.Add(1)
			torrentsTotal.Add(1)
		}
	}
}
