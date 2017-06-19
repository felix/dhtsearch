# DHT Search

This is a Mainline DHT crawler and BitTorrent client which also provides an
HTTP interface to query the indexed data.

Distributed Hash Table (DHT) is a distributed system storing key/value pairs,
in this case it is specifically Mainline DHT, the type used by BitTorrent
clients. The crawler also implements a number of extensions which enable it to
get the metadata for the torrent enabling indexing and later searching.

The crawler joins the DHT network and listens to the conversations between
nodes, keeping track of interesting packets. The most interesting packets are
those where another node announces they have a torrent available.

This BitTorrent client only downloads the torrent metadata. The actual files
hosted by the remote nodes are not retrieved.

## Features

- **Tagging** of torrents metadata is fetched. The torrent is tagged using a
  set of regular expressions matched against the torrent name and the files in
  the torrent.

- **Filtering** can be done by tags. By default all torrents tagged 'adult' are
  not indexed. See the SkipTags option in the configuration file.

- **Full Text Search** using PostgreSQL's text search vectors. Torrent names
  are weighted more than file names.

- **Statistics** for the crawler process are available when the HTTP server is
  enabled. Fetch the JSON from the `/status` endpoint.

- **Custom tags** can be defined in the configuration file.

## Installation

All dependencies have been vendored using the
[dep](https://github.com/golang/dep) tool so installation with a recent Go
version should be as simple as:

```shell
$ go build
```

## Usage

You will need to create a PostgreSQL database using the `schema.sql` file
provided. You are going to need to sort out any port forwarding if you are
behind NAT so remote nodes can get to yours.

Configuration is done via a [TOML](https://github.com/toml-lang/toml) formatted
file or via flags passed to the daemon.

The following command line flags are available:

      -base-port int
            listen port (and first of multiple ports) (default 6881)
      -debug
            provide debug output
      -dsn string
            Database DSN (default "postgres://dht:dht@localhost/dht?sslmode=disable")
      -http-address string
            HTTP listen address:port (default "localhost:6880")
      -no-http
            no HTTP service
      -num-nodes int
            number of nodes to start (default 1)
      -quiet
            log only errors

and the following "advanced" options:

      -max-bt-workers int
            max number of BT workers (default 256)
      -max-dht-workers int
            max number of DHT workers (default 256)
      -peer-cache-size int
            memory cache of seen peers (default 200)
      -routing-table-size int
            number of remote nodes in routing table (default 1000)
      -tcp-timeout int
            TCP timeout in seconds (default 10)
      -udp-timeout int
            UDP timeout in seconds (default 10)

These options enable you to start a number of DHT nodes thus implementing a
small scale [Sybil attack](https://en.wikipedia.org/wiki/Sybil_attack). The
first DHT node will take the port specified and each subsequent port is for the
following nodes.

## TODO

- Enable rate limiting.
- Improve our manners on the DHT network (replies etc.).
- Improve the routing table implementation.
- Add results pagination.
- Add tests!
