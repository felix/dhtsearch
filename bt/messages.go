package bt

// BEP4 Core protocol Message IDs

const (
	BTMsgChoke uint8 = iota
	BTMsgUnchoke
	BTMsgInterested
	BTMsgNotInterested
	BTMsgHave
	BTMsgBitfield
	BTMsgRequest
	BTMsgPiece
	BTMsgCancel
	BTMsgPort
	_
	_
	_
	BTMsgSuggest // 13
	BTMsgHaveAll
	BTMsgHaveNone
	BTMsgRejectRequest
	BTMsgAllowedFast
	_
	_
	BTMsgExtended // 20 Extended messages BEP10
)
const (
	// BEP10 send message types
	ExtMsgTypeHandshake = uint8(0)
	// >0 is extension dependant
)

const (
	// BEP9 extension message types

	// ExtMsgTypeRequest marks a request message type
	ExtMsgTypeRequest = 0
	// extMsgTypeData marks a data message type
	ExtMsgTypeData = 1
	// extMsgTypeReject marks a reject message type
	ExtMsgTypeReject = 2
)

type ExtMsg struct {
	Type      int `bencode:"msg_type"`
	Piece     int `bencode:"piece"`
	TotalSize int `bencode:"total_size,omitzero"`
}

type ExtMsgHandshake struct {
	// Dictionary of supported extension messages which maps names of
	// extensions to an extended message ID for each extension message.
	Messages map[string]int `bencode:"m"`
	// Local TCP listen port. Allows each side to learn about the TCP port
	// number of the other side. Note that there is no need for the receiving
	// side of the connection to send this extension message, since its port
	// number is already known.
	Port int `bencode:"p,omitzero"`
	// Client name and version (as a utf-8 string).
	Version string `bencode:"v,omitzero"`
	// A string containing the compact representation of the ip address this
	// peer sees you as. i.e. this is the receiver's external ip address
	// (no port is included). This may be either an IPv4 (4 bytes) or an
	// IPv6 (16 bytes) address.
	IP string `bencode:"yourip,omitzero"`
	// If this peer has an IPv6 interface, this is the compact representation
	// of that address (16 bytes).
	IPv6 string `bencode:"ipv6,omitzero"`
	// If this peer has an IPv4 interface, this is the compact representation
	// of that address (4 bytes).
	IPv4 string `bencode:"ipv4,omitzero"`
	// An integer, the number of outstanding request messages this client
	// supports without dropping any. The default in in libtorrent is 250.
	Qsize int `bencode:"reqq,omitzero"`

	// BEP9
	// Specifies an integer value of the number of bytes of the metadata.
	MetadataSize int `bencode:"metadata_size,omitzero"`
}

type MetaInfo struct {
	PieceLength int    `bencode:"piece_length"`
	Pieces      string `bencode:"pieces"`
	Private     bool   `bencode:"private"`

	Name string `bencode:"name"`

	// Single file
	Length int    `bencode:"length,omitzero"`
	MD5Sum string `bencode:"md5sum,omitzero"`

	// Multiple files
	Files []struct {
		Length int    `bencode:"length"`
		MD5Sum string `bencode:"md5sum"`
		Path   string `bencode:"path"`
	} `bencode:"files,omitzero"`
}
