package dht

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"userspace.com.au/dhtsearch/bencode"
	"userspace.com.au/dhtsearch/infohash"
)

var (
	DefaultBootstraps = []string{
		//"malkmus.yelnah.org:51000",
		"torrents:51000",
		"dht.aelitis.com:6881",
		"dht.libtorrent.org:25401",
		"dht.transmissionbt.com:6881",
		"router.bittorrent.cloud:42069",
		"router.bittorrent.com:6881",
		"router.silotis.us:6881",
		"router.utorrent.com:6881",
	}
)

const defaultPacketSize = 1500

const (
	defaultTickTableMaintenance  = 31 * time.Second
	defaultStaleNodeGrace        = 15 * time.Minute
	defaultTickAuditTransactions = 23 * time.Second
)

// Client joins the DHT network
type Client struct {
	conn          net.PacketConn
	maxPacketSize int

	node       Node
	bootstraps []string
	udpTimeout int
	packetsOut chan packet
	chk        *transactions
	log        *slog.Logger

	table      RoutingTable
	tableState io.Reader

	// Ticker and TTL durations
	tickTableMaintenance  time.Duration
	tickAuditTransactions time.Duration
	staleNodeGrace        time.Duration

	//limiter    *rate.Limiter
	////blacklist  *lru.ARCCache

	// Hooks

	// onPingQuery is called when a ping query is received
	onPingQuery func(*Node)
	// onFindNodeQuery is called when a find_node query is received
	onFindNodeQuery     func(context.Context, *Node, Msg)
	onGetPeersQuery     func(context.Context, *Node, Msg)
	onAnnouncePeerQuery func(context.Context, *Node, Msg)
	OnNodesResponse     func(context.Context, *Node, Msg)
	OnPeersResponse     func(context.Context, *Node, Msg)
	OnSamplesResponse   func(context.Context, *Node, Msg)
}

// NewClient creates a new DHT client
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	var err error

	c := &Client{
		maxPacketSize: defaultPacketSize,
		chk:           newTransRegistry(),
		node: Node{
			ID: infohash.NewRandomID(),
		},
		log:        slog.New(slog.NewTextHandler(os.Stdout, nil)),
		udpTimeout: 1000,
		//limiter:    rate.NewLimiter(rate.Limit(50), 70),

		tickTableMaintenance:  defaultTickTableMaintenance,
		tickAuditTransactions: defaultTickAuditTransactions,
		staleNodeGrace:        defaultStaleNodeGrace,
	}

	// Set variadic options passed
	for _, option := range opts {
		err = option(c)
		if err != nil {
			return nil, err
		}
	}

	if c.table == nil {
		// Default ktable implementation
		table := NewTable(c.node.ID)
		table.log = c.log
		c.table = table
	}

	// After table exists, before import
	if err := configureMetrics(c); err != nil {
		return c, err
	}

	if c.tableState != nil {
		if _, err := c.table.ReadFrom(c.tableState); err != nil {
			c.log.Warn("failed to read table", "error", err)
		}
		c.node.ID = c.table.ID()
	}

	// if n.blacklist == nil {
	// 	n.blacklist, err = lru.NewARC(1000)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	if c.conn == nil {
		if c.conn, err = net.ListenPacket("udp", "0.0.0.0:6881"); err != nil {
			c.log.Error("failed to listen", "error", err)
			return nil, err
		}
	}

	if len(c.bootstraps) == 0 {
		c.bootstraps = DefaultBootstraps
	}

	return c, nil
}

type RoutingTable interface {
	ID() infohash.ID
	Add(*Node) bool
	Has(infohash.ID) bool
	SetSeen(infohash.ID)
	Count() int
	GetClosest(infohash.ID, int) []*Node
	GetStale(int, time.Duration) []*Node
	Remove(infohash.ID) bool

	// Used for import/export
	io.WriterTo
	io.ReaderFrom
}

type Option func(*Client) error

// WithPublicAddr sets the IP:port if different to the listen IP:port
func WithPublicAddr(s string, secure bool) Option {
	return func(c *Client) error {
		var err error
		c.node.AddrPort, err = netip.ParseAddrPort(s)
		if err != nil {
			return err
		}
		fmt.Println(c.node.AddrPort)
		ip := net.UDPAddrFromAddrPort(c.node.AddrPort)
		c.node.ID = infohash.NewRandomID()
		if secure {
			c.node.ID = infohash.NewSecureID(ip.IP)
		}
		c.node.Secure = infohash.IsSecure(c.node.ID, ip.IP)
		return nil
	}
}

func WithListener(p net.PacketConn) Option {
	return func(c *Client) error {
		c.conn = p
		return nil
	}
}

// WithListenAddress sets the IP:port address to listen on
func WithListenAddress(s string) Option {
	return func(c *Client) error {
		var err error
		c.conn, err = net.ListenPacket("udp", s)
		return err
	}
}

// WithLogger sets the IP:port address to listen on
func WithLogger(l *slog.Logger) Option {
	return func(c *Client) error {
		c.log = l
		return nil
	}
}

// WithBootstraps enables custom bootstrap addresses
func WithBootstraps(addrs ...string) Option {
	return func(c *Client) error {
		c.bootstraps = addrs
		return nil
	}
}

// WithTable enables using a custom node table implementation.
func WithTable(rt RoutingTable) Option {
	return func(c *Client) error {
		c.table = rt
		return nil
	}
}

// WithState enables using a custom node table implementation.
func WithState(r io.Reader) Option {
	return func(c *Client) error {
		c.tableState = r
		return nil
	}
}

// OnAnnoucePeer is called when an announce_peer query is received
func OnAnnouncePeerQuery(f func(context.Context, *Node, Msg)) Option {
	return func(c *Client) error {
		c.onAnnouncePeerQuery = f
		return nil
	}
}

// OnGetPeersQuery is called when a get_peers query is received
func OnGetPeersQuery(f func(context.Context, *Node, Msg)) Option {
	return func(c *Client) error {
		c.onGetPeersQuery = f
		return nil
	}
}

// OnPeersResponse is called when a response has peers
func OnPeersResponse(f func(context.Context, *Node, Msg)) Option {
	return func(c *Client) error {
		c.OnPeersResponse = f
		return nil
	}
}

// OnSamplesResponse is called when a response has samples
func OnSamplesResponse(f func(context.Context, *Node, Msg)) Option {
	return func(c *Client) error {
		c.OnSamplesResponse = f
		return nil
	}
}

// Close stuff
func (c *Client) Close() error {
	c.log.Warn("node closing")
	return nil
}

func (c *Client) SaveState(w io.Writer) error {
	_, err := c.table.WriteTo(w)
	return err
}

// Run starts the node on the DHT
func (c *Client) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Go(func() { c.packetWriter(ctx) })
	wg.Go(func() { c.tableMaintenance(ctx) })
	wg.Go(func() { c.auditTransactions(ctx) })
	wg.Go(func() { c.packetReader(ctx) })
	c.log.Info("listening", "id", c.node.ID, "address", c.node.AddrPort, "listen", c.conn.LocalAddr().String())
	wg.Wait()
	// Close this after packetWriter has finished
	close(c.packetsOut)
	c.log.Debug("client stopped")
	return nil
}

func (c *Client) wants() []string {
	if c.node.AddrPort.Addr().Is4() {
		return []string{Want4}
	} else {
		return []string{Want6}
	}
}

//type PeerStore interface {
//	Add(Node) (bool, error)
//	Get(int) ([]Node, error)
//	//Delete(Node) error
//	Reset() error

//	// Used for import/export
//	io.WriterTo
//	io.ReaderFrom
//}

// func addrPort2Addr(in netip.AddrPort) net.Addr {
// 	return net.UDPAddrFromAddrPort(in)
// }
// func addr2AddrPort(in net.Addr) (netip.AddrPort, error) {
// 	return netip.ParseAddrPort(in.String())
// }

// Unprocessed packet from socket
type packet struct {
	raddr net.Addr
	data  []byte
}

func (c *Client) tableMaintenance(ctx context.Context) {
	c.log.Info("starting table maintenance", "interval", c.tickTableMaintenance, "grace", c.staleNodeGrace)

	pingStale := func() {
		stale := c.table.GetStale(8, c.staleNodeGrace)
		for _, n := range stale {
			// if n.FailedResponses > 2 {
			// 	c.table.Remove(n.ID)
			// 	removed++
			// 	continue
			// }
			n.PingAttempts += 1
			// Don't just ping, get more nodes
			//c.SendMsg(*n, NewPingQuery(c.node.ID))
			rID := infohash.NewCloseID(c.node.ID)
			c.sendMsg(n, NewFindNodeQuery(c.node.ID, rID, c.wants()))
		}
		c.log.Debug("contacted stale nodes", "sent", len(stale))
	}

	makeNeighbours := func() {
		n := c.table.Count()
		if n == 0 {
			c.bootstrap()
			return
		}
		rID := infohash.NewRandomID()
		nodes := c.table.GetClosest(c.node.ID, 8)
		for _, rn := range nodes {
			c.sendMsg(rn, NewFindNodeQuery(c.node.ID, rID, c.wants()))
		}
		c.log.Debug("making neighbours", "sent", len(nodes))
	}

	// Once at startup
	makeNeighbours()

	ticker := time.Tick(c.tickTableMaintenance)
	for {
		select {
		case <-ctx.Done():
			c.log.Info("stopping table maintenance")
			return
		case <-ticker:
			makeNeighbours()
			pingStale()
		}
	}
}

func (c *Client) bootstrap() {
	for _, s := range c.bootstraps {
		host, portStr, err := net.SplitHostPort(s)
		if err != nil {
			c.log.Warn("failed to parse bootstrap entry", "address", s, "error", err)
			continue
		}
		log := c.log.With(slog.String("host", host))
		port, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			c.log.Warn("failed to parse bootstrap port", "error", err)
			continue
		}
		ips, err := net.LookupIP(host)
		if err != nil {
			log.Warn("failed to resolve", "error", err)
			continue
		}
		for _, b := range ips {
			ip, ok := netip.AddrFromSlice(b)
			if !ok {
				log.Warn("failed to convert address")
				continue
			}
			if c.node.AddrPort.Addr().Is4() != ip.Is4() {
				log.Debug("different network family")
				continue
			}
			ap := netip.AddrPortFrom(ip, uint16(port))
			rn := &Node{
				ID:       infohash.NewRandomID(),
				AddrPort: ap,
			}
			log.Info("bootstrapping", "address", ap)
			c.sendMsg(rn, NewFindNodeQuery(c.node.ID, rn.ID, c.wants()))
		}
	}
}

// GetPeers sends a get_peers message
func (c *Client) GetPeers(rn *Node, ih infohash.ID, cb TransactionCallback) {
	msg := &Msg{
		Query: "get_peers",
		Type:  "q",
		Args: &MsgArgs{
			ID:       &c.node.ID,
			InfoHash: &ih,
		},
	}
	c.log.Debug("sending get_peers", "ih", ih, "raddr", rn.AddrPort, "id", rn.ID)
	c.sendMsgWithCallback(rn, msg, cb)
}

// GetSamples sends a sample_infohashes message
func (c *Client) GetSamples(ih infohash.ID) {
	nodes := c.table.GetClosest(ih, 8)
	msg := &Msg{
		Query: "sample_infohashes",
		Type:  "q",
		Args: &MsgArgs{
			// The querying node
			ID: &c.node.ID,
			// The ID sought
			Target: &ih,
		},
	}
	for _, rn := range nodes {
		c.sendMsg(rn, msg)
	}
}

func (c *Client) sendMsg(rn *Node, m *Msg) {
	c.sendMsgWithCallback(rn, m, nil)
}

// sendMsg sends a KRPC message to the network
func (c *Client) sendMsgWithCallback(rn *Node, m *Msg, cb TransactionCallback) {
	log := c.log.With(
		"type", m.Query,
		"addr", rn.AddrPort,
	)
	// Don't send to self
	if rn.ID.Equal(c.node.ID) {
		log.Warn("not sending to self")
		return
	}
	if m.Type == "q" {
		log = c.log.With(
			"qType", m.Type,
			"query", m.Query,
		)
		if !rn.canSend(m.Query) {
			log.Warn("ratelimited")
			return
		}
	}
	addr := net.UDPAddrFromAddrPort(rn.AddrPort)
	if m.Type == "q" {
		c.registerTransaction(rn, m, cb)
	}
	log = log.With("tid", m.TID)
	b, err := bencode.Marshal(m)
	if err != nil {
		log.Warn("failed to marshal", "error", err)
	}
	log.Debug("sending message")
	c.packetsOut <- packet{
		data:  b,
		raddr: addr,
	}
}

func (c *Client) packetWriter(ctx context.Context) {
	c.log.Debug("starting packet writer")
	// Packets onto the network
	c.packetsOut = make(chan packet, 2024)
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-c.packetsOut:
			if p.raddr.String() == c.conn.LocalAddr().String() {
				continue
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
			n, err := c.conn.WriteTo(p.data, p.raddr)
			if err != nil {
				//n.blacklist.Add(p.raddr.String(), true)
				// TODO reduce limit
				c.log.Warn("failed to write packet", "error", err)
			}
			netPackets.Add(
				ctx, 1,
				metric.WithAttributes(
					attribute.String("network.io.direction", "transmit")))
			netOctets.Add(
				ctx, int64(n),
				metric.WithAttributes(
					attribute.String("network.io.direction", "transmit")))
		}
	}
}

func (c *Client) packetReader(ctx context.Context) {
	c.log.Info("starting packet reader")
	pool := sync.Pool{
		New: func() any {
			out := make([]byte, c.maxPacketSize)
			return &out
		},
	}

	for {
		select {
		case <-ctx.Done():
			c.log.Info("stopping packet reader")
			_ = c.conn.Close()
			return
		default:
			b := pool.Get().(*[]byte)
			_ = c.conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			//c.log.Debug("waiting UDP read")
			n, addr, err := c.conn.ReadFrom(*b)
			netOctets.Add(
				ctx, int64(n),
				metric.WithAttributes(
					attribute.String("network.io.direction", "receive")))
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				c.log.Warn("UDP read error", "error", err)
				return
			}
			netPackets.Add(
				ctx, 1,
				metric.WithAttributes(
					attribute.String("network.io.direction", "receive")))

			var m Msg
			if err := bencode.Unmarshal((*b)[:n], &m); err != nil {
				c.log.Warn("krpc unmarshal msg failed", "error", err)
				return
			}
			pool.Put(b)
			rn := &Node{
				LastContact: time.Now(),
				AddrPort:    addr.(*net.UDPAddr).AddrPort(),
			}
			go c.processMessage(ctx, rn, m)
		}
	}
}

// Parse a KRPC packet into a message
// Called in goroutine
func (c *Client) processMessage(ctx context.Context, rn *Node, m Msg) {
	log := c.log.With("type", m.Type, "raddr", rn.AddrPort)
	// if _, black := n.blacklist.Get(p.raddr.String()); black {
	// 	return fmt.Errorf("blacklisted: %s", p.raddr.String())
	// }
	var err error
	if m.Type == "q" {
		err = c.handleQuery(ctx, rn, m)
	} else {
		err = c.handleResponse(ctx, rn, m)
	}
	if err != nil {
		log.Warn("failed to process message", "error", err)
		//n.blacklist.Add(p.raddr.String(), true)
	}
}

// handleResponse handles responses received from udp.
func (c *Client) handleResponse(ctx context.Context, rn *Node, m Msg) error {
	log := c.log.With(
		"raddr", rn.AddrPort,
		"id", rn.ID,
	)

	trans, err := c.checkTransaction(m)
	if m.Type == "e" {
		log = log.With("code", m.Error.Code, "error", m.Error.Msg, "tid", m.TID, "qType", trans.msg)
		if m.Error.Code == 204 {
			// Don't query this node like this again
			rn.rateQuery(trans.msg, -1)
			log.Info("rate limiting node")
		}
		return nil
	}
	if err != nil {
		return err
	}

	if m.Response == nil {
		return errors.New("missing reponse")
	}
	rn.ID = *m.Response.ID
	if !c.table.Has(rn.ID) {
		_ = c.table.Add(rn)
	}
	c.table.SetSeen(rn.ID)
	log.Debug("received response")
	r := *m.Response

	// Add new nodes to our routing table
	var newNodes []*Node
	if r.NodesList != nil {
		newNodes = append(newNodes, r.NodesList.Nodes...)
	}
	if r.Nodes6List != nil {
		newNodes = append(newNodes, r.Nodes6List.Nodes...)
	}
	if len(newNodes) > 0 {
		for _, cn := range newNodes {
			_ = c.table.Add(cn)
		}
		if c.OnNodesResponse != nil {
			log.Debug("on nodes")
			// TODO set deadline
			go c.OnNodesResponse(ctx, rn, m)
		}
	}

	if len(r.Values) > 0 {
		rn.rateQuery(trans.msg, r.Interval)
		if c.OnPeersResponse != nil {
			log.Info("on peers")
			// TODO set deadline
			go c.OnPeersResponse(ctx, rn, m)
		}
	}

	if len(r.Samples) > 0 {
		log.Info("on samples")
		if r.Interval > 0 {
			rn.NextQuery["sample_infohashes"] = time.Now().Add(time.Duration(r.Interval) * time.Second)
		}
		// if m.Response != nil && m.Response.ID != nil {
		// 	samples := infohash.SplitCompactInfohashes(
		// 		m.Response.Samples,
		// 	)
		// 	for _, ih := range samples {
		// 		c.GetPeers(ih)
		// 	}
		// }
		if c.OnSamplesResponse != nil {
			// TODO set deadline
			go c.OnSamplesResponse(ctx, rn, m)
		}
	}
	if trans.cb != nil {
		go trans.cb(rn)
	}

	return nil
}

// handleQuery handles incoming queries
func (c *Client) handleQuery(ctx context.Context, rn *Node, m Msg) error {
	if m.Args.ID == nil {
		return errors.New("query missing ID")
	}
	rn.ID = *m.Args.ID
	log := c.log.With(
		"query", m.Query,
		"raddr", rn.AddrPort,
		"id", rn.ID,
	)
	c.table.Add(rn)
	log.Debug("received query")

	switch m.Query {
	case "ping":
		c.sendMsg(rn, &Msg{
			Type: "r",
			TID:  m.TID,
			Response: &Response{
				ID: &c.node.ID,
			},
		})
		if c.onPingQuery != nil {
			log.Info("on ping")
			// TODO set deadline
			go c.onPingQuery(rn)
		}
	case "find_node":
		if m.Args.Target == nil {
			return errors.New("query missing target ID")
		}
		target := *m.Args.Target
		out := &Msg{
			Type: "r",
			TID:  m.TID,
			Response: &Response{
				ID:    &c.node.ID,
				Token: m.Args.Token,
			},
		}
		nodes := c.table.GetClosest(target, 8)
		switch c.node.Family {
		case Want4:
			out.Response.NodesList.Nodes = nodes
		case Want6:
			out.Response.Nodes6List.Nodes = nodes
		}
		c.sendMsg(rn, out)
		if c.onFindNodeQuery != nil {
			log.Info("on find_node")
			// TODO set deadline
			go c.onFindNodeQuery(ctx, rn, m)
		}
	case "get_peers":
		if m.Args.InfoHash == nil {
			return errors.New("query missing infohash ID")
		}
		target := *m.Args.InfoHash
		out := &Msg{
			Type: "r",
			TID:  m.TID,
			Response: &Response{
				ID:    &c.node.ID,
				Token: m.Args.Token,
			},
		}
		nodes := c.table.GetClosest(target, 8)
		switch c.node.Family {
		case Want4:
			out.Response.NodesList.Nodes = nodes
		case Want6:
			out.Response.Nodes6List.Nodes = nodes
		}
		c.sendMsg(rn, out)
		if c.onGetPeersQuery != nil {
			log.Info("on get_peers")
			// TODO set deadline
			go c.onGetPeersQuery(ctx, rn, m)
		}
	case "announce_peer":
		// port := uint16(m.Args.Port)
		// // If it is present and non-zero, the port argument should be
		// // ignored and the source port of the UDP packet should be used
		// // as the peer's port instead.
		// if m.Args.ImpliedPort {
		// 	port = rn.AddrPort.Port()
		// }
		if c.onAnnouncePeerQuery != nil {
			log.Info("on announce_peer")
			// TODO set deadline
			go c.onAnnouncePeerQuery(ctx, rn, m)
		}
	default:
		log.Warn("unknown type")
	}
	return nil
}
