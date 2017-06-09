package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var basePort, numNodes int
	var debug bool = false
	flag.IntVar(&basePort, "port", 6881, "listen port (and first of multiple ports)")
	flag.IntVar(&numNodes, "nodes", 1, "number of nodes to start")
	flag.BoolVar(&debug, "debug", false, "provide debug output")
	flag.Parse()

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
	db, err := newDB()
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()

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
	//processed := make(chan peer)

	// Create BT nodes
	for i := 0; i < numNodes; i++ {
		btClient := newBTClient(peers, torrents)
		btClient.debug = debug
		err = btClient.run(done)
		if err != nil {
			os.Exit(1)
		}
	}

	var t Torrent
	for {
		select {
		case t = <-torrents:
			length := t.Length
			for _, f := range t.Files {
				length = length + f.Length
			}

			fmt.Printf("Torrrent of size %d named: %s url: magnet:?xt=urn:btih:%s\n", length, t.Name, t.InfoHash)
			// TODO add tags
			err := db.updateTorrent(t)
			if err != nil {
				fmt.Printf("Error saving torrent: %q\n", err)
			}
		}
	}
}
