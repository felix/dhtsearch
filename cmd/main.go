package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/felix/dhtsearch/bt"
	"github.com/felix/dhtsearch/db"
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
	tagREs   map[string]*regexp.Regexp
	skipTags string
)

// Store vars
var (
	dsn           string
	ihBlacklist   map[string]bool
	peerBlacklist map[string]bool
)

func main() {
	flag.IntVar(&port, "port", 6881, "listen port (and first for multiple nodes")
	flag.BoolVar(&debug, "debug", false, "show debug output")
	flag.BoolVar(&ipv6, "6", false, "listen on IPv6 also")
	flag.IntVar(&dhtNodes, "dht-nodes", 1, "number of DHT nodes to start")

	flag.IntVar(&btNodes, "bt-nodes", 3, "number of BT nodes to start")
	flag.StringVar(&skipTags, "skip-tags", "xxx", "tags of torrents to skip")

	flag.StringVar(&dsn, "dsn", "postgres://dhtsearch@localhost/dhtsearch", "database DSN")

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

	store, err := db.NewStore(dsn)
	defer store.Close()
	if err != nil {
		log.Error("failed to connect store", "error", err)
		os.Exit(1)
	}

	createTagRegexps()

	ihBlacklist = make(map[string]bool)
	peerBlacklist = make(map[string]bool)
	// TODO read in existing blacklist
	// TODO populate bloom filter

	go startDHTNodes(store)

	go startBTWorkers(store)

	go processPendingPeers(store)

	for {
		select {
		case <-time.After(300 * time.Second):
			fmt.Println("mark")
		}
	}
}

func startDHTNodes(s models.PeerStore) {
	log.Debug("starting dht nodes")
	nodes := make([]*dht.Node, dhtNodes)

	for i := 0; i < dhtNodes; i++ {
		dht, err := dht.NewNode(
			dht.SetLogger(log.Named("dht")),
			dht.SetPort(port+i),
			dht.SetIPv6(ipv6),
			dht.SetOnAnnouncePeer(func(p models.Peer) {
				if black := ihBlacklist[p.Infohash.String()]; black {
					log.Debug("ignoring blacklisted infohash", "peer", p)
					return
				}
				if black := peerBlacklist[p.Addr.String()]; black {
					log.Debug("ignoring blacklisted peer", "peer", p)
					return
				}
				log.Debug("peer announce", "peer", p)
				err := s.SavePeer(&p)
				if err != nil {
					log.Error("failed to save peer", "error", err)
				}
			}),
		)
		if err != nil {
			log.Error("failed to create node", "error", err)
			//return nodes, err
		}
		go dht.Run()
		nodes[i] = dht
	}
	//return nodes, err
}

func processPendingPeers(s models.InfohashStore) {
	log.Debug("processing pending peers")
	for {
		peers, err := s.PendingInfohashes(1)
		if err != nil {
			log.Debug("failed to get pending peer", "error", err)
			time.Sleep(time.Second * 1)
			continue
		}
		for _, p := range peers {
			//log.Debug("pending peer retrieved", "peer", *p)
			select {
			case w := <-pool:
				//log.Debug("assigning peer to bt worker")
				w <- *p
			}
		}
	}
}

func startBTWorkers(s models.TorrentStore) {
	log.Debug("starting bittorrent workers")
	pool = make(chan chan models.Peer)
	torrents = make(chan models.Torrent)

	onNewTorrent := func(t models.Torrent) {
		// Add tags
		tags := tagTorrent(t, tagREs)
		for _, skipTag := range strings.Split(skipTags, ",") {
			for _, tg := range tags {
				if skipTag == tg {
					log.Debug("skipping torrent", "infohash", t.Infohash, "tags", tags)
					ihBlacklist[t.Infohash.String()] = true
					return
				}
			}
		}
		t.Tags = tags
		log.Debug("torrent tagged", "infohash", t.Infohash, "tags", tags)
		err := s.SaveTorrent(&t)
		if err != nil {
			log.Error("failed to save torrent", "error", err)
		}
		log.Info("torrent added", "name", t.Name, "size", t.Size, "tags", t.Tags)
	}

	/*
		onBadPeer := func(p models.Peer) {
			log.Debug("removing peer", "peer", p)
			err := s.RemovePeer(&p)
			if err != nil {
				log.Error("failed to remove peer", "peer", p, "error", err)
			}
			peerBlacklist[p.Addr.String()] = true
		}
	*/

	for i := 0; i < btNodes; i++ {
		w, err := bt.NewWorker(
			pool,
			bt.SetLogger(log.Named("bt")),
			bt.SetPort(port+i),
			bt.SetIPv6(ipv6),
			bt.SetOnNewTorrent(onNewTorrent),
			//bt.SetOnBadPeer(onBadPeer),
		)
		if err != nil {
			log.Error("failed to create bt worker", "error", err)
			return
		}
		log.Debug("running bt node", "index", i)
		go w.Run()
	}
}

// Filter on words, existing
func createTagRegexps() {
	tagREs = make(map[string]*regexp.Regexp)
	for tag, re := range tags {
		tagREs[tag] = regexp.MustCompile("(?i)" + re)
	}
	// Add character classes
	for cc, _ := range unicode.Scripts {
		if cc == "Latin" || cc == "Common" {
			continue
		}
		className := strings.ToLower(cc)
		// Test for 3 or more characters per character class
		tagREs[className] = regexp.MustCompile(fmt.Sprintf(`(?i)\p{%s}{3,}`, cc))
	}
	// Merge user tags
	/*
		for tag, re := range Config.Tags {
			if !Config.Quiet {
				fmt.Printf("Adding user tag: %s = %s\n", tag, re)
			}
			tagREs[tag] = regexp.MustCompile("(?i)" + re)
		}
	*/
}
