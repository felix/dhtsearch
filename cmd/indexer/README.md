# DHT indexer

The general process:

- generate random infohash
- on tick: send sample request to close nodes
- on find_nodes response: send sample request to found
- on samples response: send get_peers request to sender
- on get_peers response: download metadata

