package dhtsearch

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/felix/logger"
)

func uptime() interface{} {
	return int64(time.Since(start).Seconds())
}

func main() {
	//defer profile.Start(profile.CPUProfile).Stop()
	expvar.Publish("uptime", expvar.Func(uptime))

	log := logger.New(&logger.Options{
		Name:  "dht",
		Level: logger.Info,
	})

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
	for tag := range tags {
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
	bt := &btClient{}
	bt.log = log.Named("bt")
	err = btClient.run(torrents)
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

	// Filter peers
	go func() {
		for {
			select {
			case p = <-peers:
				if ok := cache[p.id]; ok {
					peersSkipped.Add(1)
					continue
				}
				peersAnnounced.Add(1)
				if len(cache) > Config.Advanced.PeerCacheSize {
					fmt.Printf("Flushing peer cache\n")
					cache = make(map[string]bool)
				}
				cache[p.id] = true
				if torrentExists(p.id) {
					peersSkipped.Add(1)
					continue
				}
				filteredPeers <- p
				dhtCachedPeers.Set(int64(len(cache)))
			}
		}
	}()

	for {
		select {
		case t = <-torrents:
			length := t.Size

			// Add tags
			tagTorrent(&t)

			// Not sure if I like continue labels, so this
			var notWanted = false
			for _, tag := range Config.SkipTags {
				if hasTag(t, tag) {
					if !Config.Quiet {
						fmt.Printf("Skipping torrent tagged '%s': %q\n", tag, t.Name)
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
