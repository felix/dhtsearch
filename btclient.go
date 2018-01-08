package dhtsearch

// Lifted and adapted from github.com/shiyanhui/dht

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/felix/logger"
)

const (
	// MsgRequest represents request message type
	MsgRequest = iota
	// MsgData represents data message type
	MsgData
	// MsgReject represents reject message type
	MsgReject
	// MsgExtended represents it is a extended message
	MsgExtended = 20
)

const (
	// BlockSize is 2 ^ 14
	BlockSize = 16384
	// MaxMetadataSize represents the max medata it can accept
	MaxMetadataSize = BlockSize * 1000
	// HandshakeBit represents handshake bit
	HandshakeBit = 0
)

var handshakePrefix = []byte{
	19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114,
	111, 116, 111, 99, 111, 108, 0, 0, 0, 0, 0, 16, 0, 1,
}

type btClient struct {
	pool chan chan peer
	log  logger.Logger
}

func (bt *btClient) run(torrentCh chan<- *Torrent) error {
	peerCh := make(chan peer)

	if bt.log == nil {
		bt.log = logger.New(&logger.Options{
			Name:  "bt",
			Level: logger.Info,
		})
	}

	go func() {
		for {
			// Signal we are ready for work
			bt.pool <- peerCh

			select {
			case p := <-peerCh:
				// Got work
				if len(p.id) != 20 {
					return
				}
				bt.log.Debug("fetching metadata", "peer", p.id)
				md, err := bt.fetchMetadata(p)
				if err != nil {
					bt.log.Error("failed to fetch metadata", "error", err)
				}

				t, err := decodeMetadata(p, md)
				if err != nil {
					bt.log.Error("failed to decode metadata", "error", err)
				}
				torrentCh <- t
			}
		}
	}()
	return nil
}

// fetchMetadata fetchs medata info accroding to infohash from dht.
func (bt *btClient) fetchMetadata(p peer) (out []byte, err error) {
	var (
		length       int
		msgType      byte
		piecesNum    int
		pieces       [][]byte
		utMetadata   int
		metadataSize int
	)

	defer func() {
		pieces = nil
		recover()
	}()

	infoHash := p.id
	address := p.address.String()

	dial, err := net.DialTimeout("tcp", address, time.Second*15)
	if err != nil {
		return out, err
	}
	conn := dial.(*net.TCPConn)
	conn.SetLinger(0)
	defer conn.Close()

	data := bytes.NewBuffer(nil)
	data.Grow(BlockSize)

	// TCP handshake
	if sendHandshake(conn, []byte(infoHash), []byte(genInfoHash())) != nil ||
		read(conn, 68, data) != nil ||
		onHandshake(data.Next(68)) != nil ||
		sendExtHandshake(conn) != nil {
		return
	}

	for {
		length, err = readMessage(conn, data)
		if err != nil {
			return out, err
		}

		if length == 0 {
			continue
		}

		msgType, err = data.ReadByte()
		if err != nil {
			return out, err
		}

		switch msgType {
		case MsgExtended:
			extendedID, err := data.ReadByte()
			if err != nil {
				return out, err
			}

			payload, err := ioutil.ReadAll(data)
			if err != nil {
				return out, err
			}

			if extendedID == 0 {
				if pieces != nil {
					return out, errors.New("invalid extended ID")
				}

				utMetadata, metadataSize, err = getUTMetaSize(payload)
				if err != nil {
					return out, err
				}

				piecesNum = metadataSize / BlockSize
				if metadataSize%BlockSize != 0 {
					piecesNum++
				}

				pieces = make([][]byte, piecesNum)
				go bt.requestPieces(conn, utMetadata, metadataSize, piecesNum)

				continue
			}

			if pieces == nil {
				return out, errors.New("no pieces found")
			}

			d, index, err := DecodeDict(payload, 0)
			if err != nil {
				return out, err
			}
			dict := d.(map[string]interface{})

			err = parseKeys(dict, [][]string{{"msg_type", "int"}, {"piece", "int"}})
			if err != nil {
				return out, err
			}

			if dict["msg_type"].(int) != MsgData {
				continue
			}

			piece := dict["piece"].(int)
			pieceLen := length - 2 - index

			if (piece != piecesNum-1 && pieceLen != BlockSize) ||
				(piece == piecesNum-1 && pieceLen != metadataSize%BlockSize) {
				return out, errors.New("invalid piece count")
			}

			pieces[piece] = payload[index:]

			if bt.isDone(pieces) {
				metadataInfo := bytes.Join(pieces, nil)

				// Check the metadata
				info := sha1.Sum(metadataInfo)
				if !bytes.Equal([]byte(infoHash), info[:]) {
					return out, errors.New("metadata does not match infohash")
				}
				return metadataInfo, nil
			}
		default:
			data.Reset()
		}
	}
}

func decodeMetadata(p peer, md []byte) (*Torrent, error) {
	metadata, err := Decode(md)
	if err != nil {
		return nil, err
	}
	info := metadata.(map[string]interface{})

	if _, ok := info["name"]; !ok {
		return nil, errors.New("Metadata missing name")
	}

	bt := Torrent{
		InfoHash: hex.EncodeToString([]byte(p.id)),
		Name:     info["name"].(string),
	}

	if v, ok := info["files"]; ok {
		files := v.([]interface{})
		bt.Files = make([]File, len(files))

		for i, item := range files {
			f := item.(map[string]interface{})
			paths := f["path"].([]interface{})
			path := make([]string, len(paths))
			for j, p := range paths {
				path[j] = p.(string)
			}
			fSize := f["length"].(int)
			bt.Files[i] = File{
				// Assume Unix path sep
				Path: strings.Join(path[:], "/"),
				Size: fSize,
			}
			// Ensure the torrent size totals all files'
			bt.Size = bt.Size + fSize
		}
	} else if _, ok := info["length"]; ok {
		bt.Size = info["length"].(int)
	}
	return &bt, nil
}

// isDone checks if all pieces are complete
func (bt *btClient) isDone(pieces [][]byte) bool {
	for _, piece := range pieces {
		if len(piece) == 0 {
			return false
		}
	}
	return true
}

// read reads size-length bytes from conn to data.
func read(conn *net.TCPConn, size int, data *bytes.Buffer) error {
	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(TCPTimeout)))

	n, err := io.CopyN(data, conn, int64(size))
	if err != nil || n != int64(size) {
		return errors.New("read error")
	}
	return nil
}

// readMessage gets a message from the tcp connection.
func readMessage(conn *net.TCPConn, data *bytes.Buffer) (length int, err error) {
	if err = read(conn, 4, data); err != nil {
		return length, err
	}

	length, err = bytes2int(data.Next(4))
	if err != nil {
		return length, err
	}

	if length == 0 {
		return length, nil
	}

	err = read(conn, length, data)
	return length, err
}

// sendMessage sends data to the connection.
func sendMessage(conn *net.TCPConn, data []byte) error {
	length := int32(len(data))

	buffer := bytes.NewBuffer(nil)
	binary.Write(buffer, binary.BigEndian, length)

	conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(TCPTimeout)))
	b, err := conn.Write(append(buffer.Bytes(), data...))
	return err
}

// sendHandshake sends handshake message to conn.
func sendHandshake(conn *net.TCPConn, infoHash, peerID []byte) error {
	data := make([]byte, 68)
	copy(data[:28], handshakePrefix)
	copy(data[28:48], infoHash)
	copy(data[48:], peerID)

	conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(TCPTimeout)))
	b, err := conn.Write(data)
	return err
}

// onHandshake handles the handshake response.
func onHandshake(data []byte) (err error) {
	if !(bytes.Equal(handshakePrefix[:20], data[:20]) && data[25]&0x10 != 0) {
		err = errors.New("invalid handshake response")
	}
	return err
}

// sendExtHandshake requests for the ut_metadata and metadata_size.
func sendExtHandshake(conn *net.TCPConn) error {
	data := append(
		[]byte{MsgExtended, HandshakeBit},
		Encode(map[string]interface{}{
			"m": map[string]interface{}{"ut_metadata": 1},
		})...,
	)

	return sendMessage(conn, data)
}

// getUTMetaSize returns the ut_metadata and metadata_size.
func getUTMetaSize(data []byte) (utMetadata int, metadataSize int, err error) {
	v, err := Decode(data)
	if err != nil {
		return utMetadata, metadataSize, err
	}

	dict, ok := v.(map[string]interface{})
	if !ok {
		return utMetadata, metadataSize, errors.New("invalid dict")
	}

	err = parseKeys(dict, [][]string{{"metadata_size", "int"}, {"m", "map"}})
	if err != nil {
		return utMetadata, metadataSize, err
	}

	m := dict["m"].(map[string]interface{})
	err = parseKey(m, "ut_metadata", "int")
	if err != nil {
		return utMetadata, metadataSize, err
	}

	utMetadata = m["ut_metadata"].(int)
	metadataSize = dict["metadata_size"].(int)

	if metadataSize > MaxMetadataSize {
		err = errors.New("metadata_size too long")
	}
	return utMetadata, metadataSize, err
}

// Request more pieces
func (bt *btClient) requestPieces(conn *net.TCPConn, utMetadata int, metadataSize int, piecesNum int) {
	buffer := make([]byte, 1024)
	for i := 0; i < piecesNum; i++ {
		buffer[0] = MsgExtended
		buffer[1] = byte(utMetadata)

		msg := Encode(map[string]interface{}{
			"msg_type": MsgRequest,
			"piece":    i,
		})

		length := len(msg) + 2
		copy(buffer[2:length], msg)

		sendMessage(conn, buffer[:length])
	}
	buffer = nil
}

// bytes2int returns the int value it represents.
func bytes2int(data []byte) (int, error) {
	n := len(data)
	if n > 8 {
		return 0, errors.New("data too long")
	}

	val := uint64(0)

	for i, b := range data {
		val += uint64(b) << uint64((n-i-1)*8)
	}
	return int(val), nil
}
