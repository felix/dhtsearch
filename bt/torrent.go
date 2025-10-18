package bt

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"time"

	"userspace.com.au/dhtsearch/bencode"
	"userspace.com.au/dhtsearch/infohash"
)

type Torrent struct {
	ih infohash.ID
}

func NewTorrent(ih infohash.ID) *Torrent {
	return &Torrent{ih: ih}
}

func (t *Torrent) FetchMetadata(ctx context.Context, ap netip.AddrPort) ([]byte, error) {
	tcpTimeout := 5 * time.Second
	tcpAddr := net.TCPAddrFromAddrPort(ap)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// conn.SetLinger(0)
	// conn.SetNoDelay(true)

	local := conn.LocalAddr().(*net.TCPAddr).AddrPort()

	// Collect reads here
	buf := new(bytes.Buffer)
	//buf.Grow(blockSize)

	// Handshake
	conn.SetWriteDeadline(time.Now().Add(tcpTimeout))
	hsLength, err := sendHandshake(conn, t.ih)
	if err != nil {
		return nil, err
	}
	conn.SetReadDeadline(time.Now().Add(tcpTimeout))
	if err := read(conn, hsLength, buf); err != nil {
		return nil, fmt.Errorf("handshake failed: %w", err)
	}
	resp := buf.Next(hsLength)
	// BEP10 Check for extension protocol
	// The bit selected for the extension protocol is bit 20 from the right
	// (counting starts at 0). So (reserved_byte[5] & 0x10) is the
	// expression to use for checking if the client supports extended messaging.
	// See https://www.libtorrent.org/extension_protocol.html
	extBit := 1 + len(protocolString) + 6
	if resp[extBit]&0x10 != 0 {
		return nil, errors.New("extension protocol not supported")
	}
	if err = sendExtHandshake(conn, local, ap); err != nil {
		return nil, err
	}

	var (
		// extended message type, set in first message
		extMsgType uint8
		// Expected size of all metadata, set in first message
		// Used to determine length of last piece
		metadataSize int
		// Collection of piece data, the output
		pieces [][]byte
		// Expected number of pieces, derived from metadatasize
		totalPieces int
		// Current count of pieces
		gotPieces int
	)

	// Loop over incoming messages
	for {
		buf.Reset()
		if err := readMessage(conn, buf); err != nil {
			return nil, err
		}
		length := buf.Len()
		if length == 0 {
			continue
		}
		// We are only interested in bt extension protocol (20)
		msgProtocol, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		if msgProtocol != BTMsgExtended {
			continue
		}

		// Read the extension type ie. 0 == handshake
		msgExtType, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		// // This was set in the handshake and should match
		// if msgExtType != ExtMsgTypeData {
		// 	continue
		// }
		payload, err := io.ReadAll(buf)
		if err != nil {
			return nil, err
		}
		// We are past the protocol and extension type bits
		br := bencode.NewReaderFromBytes(payload)

		if msgExtType == ExtMsgTypeHandshake {
			if pieces != nil {
				// We have already done the handshake!
				return nil, errors.New("duplicate handshake")
			}
			msg := ExtMsgHandshake{}
			if !br.ReadStruct(&msg) {
				return nil, br.Err()
			}
			totalPieces = msg.MetadataSize / blockSize
			if msg.MetadataSize%blockSize != 0 {
				totalPieces += 1
			}
			if totalPieces == 0 {
				return nil, errors.New("no pieces to fetch")
			}
			pieces = make([][]byte, totalPieces)

			// The extenstion type we can handle
			if id, ok := msg.Messages["ut_metadata"]; !ok {
				return nil, errors.New("missing ut_metadata extension")
			} else {
				extMsgType = uint8(id)
			}

			metadataSize = msg.MetadataSize
			// Request all metadata pieces
			if err := requestMetadata(conn, extMsgType, totalPieces); err != nil {
				return nil, err
			}
			continue
		}
		if pieces == nil {
			return nil, errors.New("pieces not created")
		}

		var msg ExtMsg
		if !br.ReadStruct(&msg) {
			return nil, br.Err()
		}
		if msg.Type != ExtMsgTypeData {
			continue
		}
		pieceLen := len(payload) - int(br.Count())
		if msg.Piece == totalPieces-1 {
			// Last piece needs to equal remainder
			if pieceLen != metadataSize%blockSize {
				return nil, fmt.Errorf("incorrect final piece %d", msg.Piece)
			}
		} else {
			// Should be full block
			if pieceLen != blockSize {
				return nil, fmt.Errorf("incomplete piece %d", msg.Piece)
			}
		}

		pieces[msg.Piece] = payload[br.Count():]
		gotPieces += 1
		if gotPieces == totalPieces {
			break
		}
	}
	return bytes.Join(pieces, nil), nil
}

// Our peer name and version TODO
const peerID = "ML-010"

// BTv1 identifier
const protocolString = "BitTorrent protocol"

// The metadata is handled in blocks of 16KiB (16384 Bytes), see BEP9
const blockSize = 16384

const (
	extMsgTypeHandshake = uint8(0)
)

func sendHandshake(w io.Writer, id infohash.ID) (int, error) {
	// 49+len(pstr) bytes long
	// <pstrlen><pstr><reserved><info_hash><peer_id>
	out := make([]byte, 49+len(protocolString))
	out[0] = byte(len(protocolString))
	n := 1
	// pstr
	copy(out[n:n+len(protocolString)], []byte(protocolString))
	n += len([]byte(protocolString))
	// reserved, setting extended protocol bit
	reserved := bytes.Repeat([]byte{0}, 8)
	reserved[5] |= 0x10 // Extension protocol
	reserved[7] |= 0x01 // Distributed Hash Table
	copy(out[n:n+8], reserved)
	n += 8
	// target infohash
	copy(out[n:n+20], id[:])
	n += 20
	// our peer ID
	copy(out[n:], peerID[:])
	return w.Write(out)
}

func sendExtHandshake(rw io.ReadWriter, local, remote netip.AddrPort) error {
	payload := ExtMsgHandshake{
		Messages: map[string]int{
			"ut_metadata": 1,
		},
		IP:   string(remote.Addr().AsSlice()),
		Port: int(local.Port()),
	}
	if local.Addr().Is6() {
		payload.IPv6 = string(local.Addr().AsSlice())
	} else {
		payload.IPv4 = string(local.Addr().AsSlice())
	}
	b, err := bencode.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = sendMessage(rw, ExtMsgTypeHandshake, b)
	return err
}

// Send multiple requests for all the pieces
func requestMetadata(w io.Writer, extMsgType uint8, totalPieces int) error {
	for i := range totalPieces {
		payload := ExtMsg{
			Type:  ExtMsgTypeRequest,
			Piece: i,
		}
		b, err := bencode.Marshal(payload)
		if err != nil {
			return err
		}
		sendMessage(w, extMsgType, b)
	}
	return nil
}

// read reads size-length bytes
func read(r io.Reader, size int, w io.Writer) error {
	n, err := io.CopyN(w, r, int64(size))
	if err != nil {
		return err
	}
	if n != int64(size) {
		return fmt.Errorf("short read, got %d, want %d", n, size)
	}
	return nil
}

// Sends a length prefix, protocol byte, type byte, then data, see BEP10
func sendMessage(w io.Writer, mType uint8, data []byte) (n int, err error) {
	// 4 bytes length prefix. Size of the entire message. (Big endian)
	// 1 byte bt extended message id (20)
	// 1 byte bt message type id (0: handshake, >0: according to handshake)
	out := append([]byte{BTMsgExtended}, mType)
	out = append(out, data...)
	length := int32(len(out))
	if err = binary.Write(w, binary.BigEndian, length); err != nil {
		return n, err
	}
	return w.Write(out)
}

func readMessage(r io.Reader, buf *bytes.Buffer) error {
	var length int32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return err
	}
	if length == 0 {
		return nil
	}
	return read(r, int(length), buf)
}
