package dhtsearch

import (
	"regexp"

	"github.com/felix/logger"
)

const (
	TCPTimeout = 10
	UDPTimeout = 10
)

type Server struct {
	port        int
	nodes       int
	httpAddress string
	tagREs      map[string]*regexp.Regexp
	log         logger.Logger
}

// Option are options for the server
type Option func(*Server) error

func NewServer(dsn string, opts ...Option) (*Server, error) {
	s := &Server{
		port:        6881,
		nodes:       1,
		httpAddress: "localhost:6880",
		tagREs:      make(map[string]*regexp.Regexp),
	}

	// Default logger
	logOpts := &logger.Options{
		Name:  "dhtsearch",
		Level: logger.Info,
	}
	s.log = logger.New(logOpts)

	err := mergeTagRegexps(s.tagREs, tags)
	if err != nil {
		s.log.Error("failed to compile tags", "error", err)
		return nil, err
	}
	err = mergeCharacterTagREs(s.tagREs)
	if err != nil {
		s.log.Error("failed to compile character class tags", "error", err)
		return nil, err
	}

	// Set variadic options passed
	for _, option := range opts {
		err = option(s)
		if err != nil {
			return nil, err
		}
	}

	s.log.Debug("debugging output enabled")

	peers := make(chan peer)

	for i := 0; i < s.nodes; i++ {
		// Consecutive port numbers
		port := s.port + i
		node := &dhtNode{
			id:       genInfoHash(),
			address:  "",
			port:     port,
			workers:  2,
			log:      s.log.Named("dht"),
			peersOut: peers,
		}
		go node.run()
	}

	return s, nil
}

// SetLogger sets the server
func SetLogger(l logger.Logger) Option {
	return func(s *Server) error {
		s.log = l
		return nil
	}
}

// SetPort sets the base port
func SetPort(p int) Option {
	return func(s *Server) error {
		s.port = p
		return nil
	}
}

// SetNodes determines the number of nodes to start
func SetNodes(n int) Option {
	return func(s *Server) error {
		s.nodes = n
		return nil
	}
}

// SetHTTPAddress determines the listening address for HTTP
func SetHTTPAddress(a string) Option {
	return func(s *Server) error {
		s.httpAddress = a
		return nil
	}
}

// SetTags determines the listening address for HTTP
func SetTags(tags map[string]string) Option {
	return func(s *Server) error {
		// Merge user tags
		err := mergeTagRegexps(s.tagREs, tags)
		if err != nil {
			s.log.Error("failed to compile tags", "error", err)
		}
		return err
	}
}

func (s *Server) Stats() Stats {
	s.statlock.RLock()
	defer s.statlock.RUnlock()
	return s.stats
}
