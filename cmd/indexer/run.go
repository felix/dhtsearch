package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"userspace.com.au/dhtsearch/bencode"
	"userspace.com.au/dhtsearch/bt"
	"userspace.com.au/dhtsearch/dht"
	"userspace.com.au/dhtsearch/infohash"
)

func run(
	ctx context.Context,
	_ []string,
	_ func(string) string,
	stdout io.Writer,
	_ io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(
		ctx,
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	var (
		addrFlag    string
		publicFlag  string
		stateFlag   string
		secureFlag  bool
		metricsFlag bool
		verboseFlag bool
	)

	flag.StringVar(&addrFlag, "addr", "", "listen address:port")
	flag.StringVar(&publicFlag, "public", "", "public address:port")
	flag.StringVar(&stateFlag, "state", "", "state file")
	flag.BoolVar(&secureFlag, "secure", false, "Generate secure infohash")
	flag.BoolVar(&metricsFlag, "metrics", false, "Generate metrics")
	flag.BoolVar(&verboseFlag, "verbose", false, "verbose")
	flag.Parse()

	var lvl = new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	if verboseFlag {
		lvl.Set(slog.LevelDebug)
	}
	logger := slog.New(slog.NewTextHandler(
		stdout,
		&slog.HandlerOptions{Level: lvl},
	))

	opts := []dht.Option{
		dht.WithLogger(logger),
	}
	if addrFlag != "" {
		opts = append(opts, dht.WithListenAddress(addrFlag))
	} else {
		ip := getFirstPublicIP()
		if ip == nil {
			logger.Error("no IP")
			return errors.New("no IP")
		}
		addr := net.JoinHostPort(ip.String(), "6881")
		l, _ := net.ListenPacket("udp", addr)
		opts = append(opts, dht.WithListener(l))

	}
	if publicFlag != "" {
		opts = append(opts, dht.WithPublicAddr(publicFlag, secureFlag))
	}

	if stateFlag != "" {
		b, err := os.ReadFile(stateFlag)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else {
			opts = append(opts, dht.WithState(bytes.NewReader(b)))
		}
	}

	if metricsFlag {
		closer, err := newMeterProvider(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err := closer(context.Background()); err != nil {
				logger.Error(err.Error())
			}
		}()
	}

	//fetchMetadata := fetchMetadata(logger)

	svr, err := dht.NewClient(ctx, opts...)
	if err != nil {
		return err
	}
	gotInfohash := make(map[infohash.ID]bool)
	type toGet struct {
		ih infohash.ID
		ap netip.AddrPort
	}
	hashes := make(chan toGet)

	go func() {
		for tg := range hashes {
			if gotInfohash[tg.ih] {
				continue
			}
			gotInfohash[tg.ih] = true
			t := bt.NewTorrent(tg.ih)
			logger.Info("fetching metadata", "ih", tg.ih, "address", tg.ap)
			b, err := t.FetchMetadata(ctx, tg.ap)
			if err != nil {
				gotInfohash[tg.ih] = false
				logger.Error("failed to fetch metadata", "error", err)
				return
			}
			var info bt.MetaInfo
			err = bencode.Unmarshal(b, &info)
			logger.Info("fetched metadata", "name", info.Name)

		}
	}()
	// port := uint16(m.Args.Port)
	// if m.Args.ImpliedPort {
	// 	port = rn.AddrPort.Port()
	// }
	// ap := netip.AddrPortFrom(rn.AddrPort.Addr(), port)

	getPeers := func(ctx context.Context, rn *dht.Node, m dht.Msg) {
		if m.Response == nil {
			return
		}
		samples := infohash.SplitCompactInfohashes(m.Response.Samples)
		logger.Info("got samples", "count", len(samples))
		for _, ih := range samples {
			svr.GetPeers(rn, ih, func(rn *dht.Node) {
				logger.Info("get_peers callback")
				hashes <- toGet{
					ih: ih,
					ap: rn.AddrPort,
				}
			})
		}
	}
	getSamples := func(_ context.Context, _ *dht.Node, _ dht.Msg) {
		ih := infohash.NewRandomID()
		svr.GetSamples(ih)
	}
	//svr.OnPeersResponse = fetchMetadata
	svr.OnNodesResponse = getSamples
	svr.OnSamplesResponse = getPeers

	var wg sync.WaitGroup
	wg.Go(func() {
		if err := svr.Run(ctx); err != nil {
			logger.Error("dht client failed", "error", err)
		}
	})

	wg.Go(func() {
		tick := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				ih := infohash.NewRandomID()
				svr.GetSamples(ih)
			}
		}
	})
	wg.Wait()

	if stateFlag != "" {
		f, err := os.Create(stateFlag)
		if err != nil {
			return err
		}
		if err := svr.SaveState(f); err != nil {
			return err
		}
		err = f.Close()
	}
	return err
}

func getFirstPublicIP() net.IP {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		ip, _, err := net.ParseCIDR(a.String())
		if err == nil && !ip.IsLoopback() && !ip.IsPrivate() {
			return ip
		}
	}
	return nil
}

func fetchMetadata(log *slog.Logger) func(ctx context.Context, rn *dht.Node, m dht.Msg) {
	return func(ctx context.Context, rn *dht.Node, m dht.Msg) {
		// port := uint16(m.Args.Port)
		// // If it is present and non-zero, the port argument should be
		// // ignored and the source port of the UDP packet should be used
		// // as the peer's port instead.
		// if m.Args.ImpliedPort {
		// 	port = rn.AddrPort.Port()
		// }
		if m.Response == nil || m.Response.ID == nil {
			log.Warn("cannot fetch metadata, missing details")
			return
		}
		// ap := netip.AddrPortFrom(rn.AddrPort.Addr(), port)
		ap := rn.AddrPort

		samples := infohash.SplitCompactInfohashes(m.Response.Samples)
		for _, id := range samples {
			t := bt.NewTorrent(id)
			log.Info("fetching metadata", "ih", id, "address", ap)
			b, err := t.FetchMetadata(ctx, ap)
			if err != nil {
				log.Error("failed to fetch metadata", "error", err)
			}
			var info bt.MetaInfo
			err = bencode.Unmarshal(b, &info)
			log.Info("fetched metadata", "name", info.Name)
		}

	}
}
