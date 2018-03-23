package bt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/felix/dhtsearch/bencode"
	"github.com/felix/dhtsearch/krpc"
	"github.com/felix/dhtsearch/models"
	"github.com/felix/logger"
)

const (
	// MsgRequest marks a request message type
	MsgRequest = iota
	// MsgData marks a data message type
	MsgData
	// MsgReject marks a reject message type
	MsgReject
	// MsgExtended marks it as an extended message
	MsgExtended = 20
)

const (
	// BlockSize is 2 ^ 14
	BlockSize = 16384
	// MaxMetadataSize represents the max medata it can accept
	MaxMetadataSize = BlockSize * 1000
	// HandshakeBit represents handshake bit
	HandshakeBit = 0
	// TCPTimeout for BT connections
	TCPTimeout = 5
)

var handshakePrefix = []byte{
	19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114,
	111, 116, 111, 99, 111, 108, 0, 0, 0, 0, 0, 16, 0, 1,
}

type Worker struct {
	pool         chan chan models.Peer
	port         int
	family       string
	OnNewTorrent func(t models.Torrent)
	OnBadPeer    func(p models.Peer)
	log          logger.Logger
}

func NewWorker(pool chan chan models.Peer, opts ...Option) (*Worker, error) {
	var err error
	w := &Worker{
		pool: pool,
	}

	// Set variadic options passed
	for _, option := range opts {
		err = option(w)
		if err != nil {
			return nil, err
		}
	}
	return w, nil
}

func (bt *Worker) Run() error {
	peerCh := make(chan models.Peer)

	for {
		// Signal we are ready for work
		bt.pool <- peerCh

		select {
		case p := <-peerCh:
			// Got work
			bt.log.Debug("worker got work", "peer", p)
			md, err := bt.fetchMetadata(p)
			if err != nil {
				bt.log.Debug("failed to fetch metadata", "error", err)
				if bt.OnBadPeer != nil {
					bt.OnBadPeer(p)
				}
				continue
			}
			t, err := models.TorrentFromMetadata(p.Infohash, md)
			if err != nil {
				bt.log.Warn("failed to load torrent", "error", err)
				continue
			}
			if bt.OnNewTorrent != nil {
				bt.OnNewTorrent(*t)
			}
		}
	}
}

// fetchMetadata fetchs medata info accroding to infohash from dht.
func (bt *Worker) fetchMetadata(p models.Peer) (out []byte, err error) {
	var (
		length       int
		msgType      byte
		totalPieces  int
		pieces       [][]byte
		utMetadata   int
		metadataSize int
	)

	defer func() {
		pieces = nil
		recover()
	}()

	//ll := bt.log.WithFields("address", p.Addr.String())

	//ll.Debug("connecting")
	dial, err := net.DialTimeout("tcp", p.Addr.String(), time.Second*15)
	if err != nil {
		return out, err
	}
	// Cast
	conn := dial.(*net.TCPConn)
	conn.SetLinger(0)
	defer conn.Close()
	//ll.Debug("dialed")

	data := bytes.NewBuffer(nil)
	data.Grow(BlockSize)

	ih := models.GenInfohash()

	// TCP handshake
	//ll.Debug("sending handshake")
	_, err = sendHandshake(conn, p.Infohash, ih)
	if err != nil {
		return nil, err
	}

	// Handle the handshake response
	//ll.Debug("handling handshake response")
	err = read(conn, 68, data)
	if err != nil {
		return nil, err
	}
	next := data.Next(68)
	//ll.Debug("got next data")
	if !(bytes.Equal(handshakePrefix[:20], next[:20]) && next[25]&0x10 != 0) {
		//ll.Debug("next data does not match", "next", next)
		return nil, errors.New("invalid handshake response")
	}

	//ll.Debug("sending ext handshake")
	_, err = sendExtHandshake(conn)
	if err != nil {
		return nil, err
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

				totalPieces = metadataSize / BlockSize
				if metadataSize%BlockSize != 0 {
					totalPieces++
				}

				pieces = make([][]byte, totalPieces)
				go bt.requestPieces(conn, utMetadata, metadataSize, totalPieces)

				continue
			}

			if pieces == nil {
				return out, errors.New("no pieces found")
			}

			dict, index, err := bencode.DecodeDict(payload, 0)
			if err != nil {
				return out, err
			}

			mt, err := krpc.GetInt(dict, "msg_type")
			if err != nil {
				return out, err
			}

			if mt != MsgData {
				continue
			}

			piece, err := krpc.GetInt(dict, "piece")
			if err != nil {
				return out, err
			}

			pieceLen := length - 2 - index

			// Not last piece? should be full block
			if totalPieces > 1 && piece != totalPieces-1 && pieceLen != BlockSize {
				return out, fmt.Errorf("incomplete piece %d", piece)
			}
			// Last piece needs to equal remainder
			if piece == totalPieces-1 && pieceLen != metadataSize%BlockSize {
				return out, fmt.Errorf("incorrect final piece %d", piece)
			}

			pieces[piece] = payload[index:]

			if bt.isDone(pieces) {
				return bytes.Join(pieces, nil), nil
			}
		default:
			data.Reset()
		}
	}
}

// isDone checks if all pieces are complete
func (bt *Worker) isDone(pieces [][]byte) bool {
	for _, piece := range pieces {
		if len(piece) == 0 {
			return false
		}
	}
	return true
}

// read reads size-length bytes from conn to data.
func read(conn net.Conn, size int, data io.Writer) error {
	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(TCPTimeout)))
	n, err := io.CopyN(data, conn, int64(size))
	if err != nil || n != int64(size) {
		return errors.New("read error")
	}
	return nil
}

// readMessage gets a message from the tcp connection.
func readMessage(conn net.Conn, data *bytes.Buffer) (length int, err error) {
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
func sendMessage(conn net.Conn, data []byte) (int, error) {
	length := int32(len(data))

	buffer := bytes.NewBuffer(nil)
	binary.Write(buffer, binary.BigEndian, length)

	conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(TCPTimeout)))
	return conn.Write(append(buffer.Bytes(), data...))
}

// sendHandshake sends handshake message to conn.
func sendHandshake(conn net.Conn, ih, id models.Infohash) (int, error) {
	data := make([]byte, 68)
	copy(data[:28], handshakePrefix)
	copy(data[28:48], []byte(ih))
	copy(data[48:], []byte(id))

	conn.SetWriteDeadline(time.Now().Add(time.Second * time.Duration(TCPTimeout)))
	return conn.Write(data)
}

// onHandshake handles the handshake response.
func onHandshake(data []byte) (err error) {
	if !(bytes.Equal(handshakePrefix[:20], data[:20]) && data[25]&0x10 != 0) {
		err = errors.New("invalid handshake response")
	}
	return err
}

// sendExtHandshake requests for the ut_metadata and metadata_size.
func sendExtHandshake(conn net.Conn) (int, error) {
	m, err := bencode.EncodeDict(map[string]interface{}{
		"m": map[string]interface{}{"ut_metadata": 1},
	})
	if err != nil {
		return 0, err
	}
	data := append([]byte{MsgExtended, HandshakeBit}, m...)

	return sendMessage(conn, data)
}

// getUTMetaSize returns the ut_metadata and metadata_size.
func getUTMetaSize(data []byte) (utMetadata int, metadataSize int, err error) {
	dict, _, err := bencode.DecodeDict(data, 0)
	if err != nil {
		return utMetadata, metadataSize, err
	}

	m, err := krpc.GetMap(dict, "m")
	if err != nil {
		return utMetadata, metadataSize, err
	}

	utMetadata, err = krpc.GetInt(m, "ut_metadata")
	if err != nil {
		return utMetadata, metadataSize, err
	}

	metadataSize, err = krpc.GetInt(dict, "metadata_size")
	if err != nil {
		return utMetadata, metadataSize, err
	}

	if metadataSize > MaxMetadataSize {
		err = errors.New("metadata_size too long")
	}
	return utMetadata, metadataSize, err
}

// Request more pieces
func (bt *Worker) requestPieces(conn net.Conn, utMetadata int, metadataSize int, totalPieces int) {
	buffer := make([]byte, 1024)
	for i := 0; i < totalPieces; i++ {
		buffer[0] = MsgExtended
		buffer[1] = byte(utMetadata)

		msg, _ := bencode.EncodeDict(map[string]interface{}{
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
