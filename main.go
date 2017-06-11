package main

import (
	"flag"
	"fmt"
	"github.com/pkg/profile"
	"os"
)

func main() {
	defer profile.Start(profile.CPUProfile).Stop()
	var basePort, numNodes int
	var debug bool = false
	var dsn string
	flag.IntVar(&basePort, "port", 6881, "listen port (and first of multiple ports)")
	flag.IntVar(&numNodes, "nodes", 1, "number of nodes to start")
	flag.BoolVar(&debug, "debug", false, "provide debug output")
	flag.StringVar(&dsn, "dsn", "postgres://dht:dht@localhost/dht?sslmode=disable", "DB DSN")
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
	db, err := newDB(dsn)
	if err != nil {
		os.Exit(1)
	}
	db.debug = debug
	defer db.Close()

	// Initialise tags
	for tag, _ := range tags {
		_, err := db.createTag(tag)
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
	processed := make(chan peer)

	// Create BT nodes
	for i := 0; i < numNodes; i++ {
		btClient := newBTClient(processed, torrents)
		btClient.debug = debug
		err = btClient.run(done)
		if err != nil {
			os.Exit(1)
		}
	}

	// Simple cache of most recent
	cache := make(map[string]bool)
	var p peer
	var t Torrent

	for {
		select {
		case p = <-peers:
			if ok := cache[p.id]; ok {
				//fmt.Printf("Torrent in cache, skipping\n")
				continue
			}
			if len(cache) > 2000 {
				fmt.Printf("Flushing cache\n")
				cache = make(map[string]bool)
			}
			cache[p.id] = true
			if db.torrentExists(p.id) {
				continue
			}
			processed <- p

		case t = <-torrents:
			length := t.Length
			for _, f := range t.Files {
				length = length + f.Length
			}

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
				continue
			}

			fmt.Printf("Torrrent length: %d, name: %s, tags: %s, url: magnet:?xt=urn:btih:%s\n", length, t.Name, t.Tags, t.InfoHash)

			err := db.saveTorrent(t)
			if err != nil {
				fmt.Printf("Error saving torrent: %q\n", err)
			}
		}
	}
}
