package main

import (
	"expvar"
	"fmt"
	"time"
	//"github.com/pkg/profile"
	"net"
	"net/http"
	"os"
)

// Exported vars
var (
	dhtPacketsIn      = expvar.NewInt("dht_packets_in")
	dhtPacketsOut     = expvar.NewInt("dht_packets_out")
	dhtPacketsDropped = expvar.NewInt("dht_packets_dropped")
	dhtErrorPackets   = expvar.NewInt("dht_error_packets")
	dhtCachedPeers    = expvar.NewInt("dht_cached_peers")
	dhtBytesIn        = expvar.NewInt("dht_bytes_in")
	dhtBytesOut       = expvar.NewInt("dht_bytes_out")
	dhtWorkers        = expvar.NewInt("dht_workers")
	btBytesIn         = expvar.NewInt("bt_bytes_in")
	btBytesOut        = expvar.NewInt("bt_bytes_out")
	btWorkers         = expvar.NewInt("bt_workers")
	peersAnnounced    = expvar.NewInt("peers_announced")
	peersSkipped      = expvar.NewInt("peers_skipped")
	torrentsSkipped   = expvar.NewInt("torrents_skipped")
	torrentsSaved     = expvar.NewInt("torrents_saved")
	torrentsTotal     = expvar.NewInt("torrents_total")
	start             = time.Now()
)

// Global
var Config config
var DB *database

func uptime() interface{} {
	return int64(time.Since(start).Seconds())
}

func main() {
	//defer profile.Start(profile.CPUProfile).Stop()
	expvar.Publish("uptime", expvar.Func(uptime))

	loadConfig()

	// Slice of channels for DHT node output
	torrents := make(chan Torrent)
	peers := make(chan peer)

	// Close upstreams channels
	done := make(chan struct{})
	defer close(done)

	// Persistence
	var err error
	DB, err = newDB(Config.Dsn)
	if err != nil {
		os.Exit(1)
	}
	defer DB.Close()

	// Initialise tags
	for tag, _ := range tags {
		_, err := createTag(tag)
		if err != nil {
			fmt.Printf("Error creating tag %s: %q\n", tag, err)
		}
	}

	// Create DHT nodes
	if Config.Debug {
		fmt.Printf("Starting %d instance(s)\n", Config.NumNodes)
	}
	for i := 0; i < Config.NumNodes; i++ {
		// Consecutive port numbers
		port := Config.BasePort + i
		dht := newDHTNode("", port, peers)
		err = dht.run(done)
		if err != nil {
			os.Exit(1)
		}
	}

	// Filter torrents
	filteredPeers := make(chan peer)

	// Create BT node
	btClient := newBTClient(filteredPeers, torrents)
	err = btClient.run(done)
	if err != nil {
		os.Exit(1)
	}

	// HTTP Server
	if !Config.NoHttp {
		http.HandleFunc("/", indexHandler)
		http.HandleFunc("/stats", statsHandler)
		http.HandleFunc("/search", searchHandler)
		sock, _ := net.Listen("tcp", Config.HttpAddress)
		go func() {
			if !Config.Quiet {
				fmt.Printf("HTTP now available at %s\n", Config.HttpAddress)
			}
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

			// Not sure if I like continue labels, so this
			var notWanted = false
			for _, tag := range Config.SkipTags {
				if hasTag(t, tag) {
					if !Config.Quiet {
						fmt.Printf("Skipping torrent: %q\n", t.Name)
					}
					notWanted = true
					break
				}
			}
			if notWanted {
				torrentsSkipped.Add(1)
				continue
			}

			err := t.save()
			if err != nil {
				fmt.Printf("Error saving torrent: %q\n", err)
				continue
			}
			if !Config.Quiet {
				fmt.Printf("Torrrent added, length: %d, name: %q, tags: %s, url: magnet:?xt=urn:btih:%s\n", length, t.Name, t.Tags, t.InfoHash)
			}
			torrentsSaved.Add(1)
			torrentsTotal.Add(1)
		}
	}
}
