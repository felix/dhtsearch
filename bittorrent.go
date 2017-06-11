package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"
)

const (
	// REQUEST represents request message type
	REQUEST = iota
	// DATA represents data message type
	DATA
	// REJECT represents reject message type
	REJECT
)

const (
	// BLOCK is 2 ^ 14
	BLOCK = 16384
	// MaxMetadataSize represents the max medata it can accept
	MaxMetadataSize = BLOCK * 1000
	// EXTENDED represents it is a extended message
	EXTENDED = 20
	// HANDSHAKE represents handshake bit
	HANDSHAKE = 0
)

var handshakePrefix = []byte{
	19, 66, 105, 116, 84, 111, 114, 114, 101, 110, 116, 32, 112, 114,
	111, 116, 111, 99, 111, 108, 0, 0, 0, 0, 0, 16, 0, 1,
}

// Annouced peer
type peer struct {
	address net.UDPAddr
	id      string
}

type File struct {
	Path   string
	Length int
}

// Data for persistent storage
type Torrent struct {
	InfoHash string
	Name     string
	Files    []File
	Length   int
	Tags     []string
}

type btClient struct {
	debug        bool
	peersIn      <-chan peer
	torrentsOut  chan<- Torrent
	workerTokens chan struct{}
}

func newBTClient(r <-chan peer, t chan<- Torrent) *btClient {
	return &btClient{
		peersIn:      r,
		torrentsOut:  t,
		workerTokens: make(chan struct{}, 256),
	}
}

func (bt *btClient) run(done <-chan struct{}) error {
	var p peer
	go func() {
		for {
			select {
			case <-done:
				return
			case p = <-bt.peersIn:
				bt.workerTokens <- struct{}{}

				go func(p peer) {
					defer func() {
						<-bt.workerTokens
					}()

					if len(p.id) != 20 {
						return
					}

					if bt.debug {
						fmt.Printf("Fetching metadata for %x\n", p.id)
					}
					bt.fetchMetadata(p)
				}(p)
			}
		}
	}()
	return nil
}

// isDone returns whether the wire get all pieces of the metadata info.
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
	conn.SetReadDeadline(time.Now().Add(time.Second * 15))

	n, err := io.CopyN(data, conn, int64(size))
	if err != nil || n != int64(size) {
		return errors.New("read error")
	}
	return nil
}

// readMessage gets a message from the tcp connection.
func readMessage(conn *net.TCPConn, data *bytes.Buffer) (
	length int, err error) {

	if err = read(conn, 4, data); err != nil {
		return
	}

	length = int(bytes2int(data.Next(4)))
	if length == 0 {
		return
	}

	if err = read(conn, length, data); err != nil {
		return
	}
	return
}

// sendMessage sends data to the connection.
func sendMessage(conn *net.TCPConn, data []byte) error {
	length := int32(len(data))

	buffer := bytes.NewBuffer(nil)
	binary.Write(buffer, binary.BigEndian, length)

	conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	_, err := conn.Write(append(buffer.Bytes(), data...))
	return err
}

// sendHandshake sends handshake message to conn.
func sendHandshake(conn *net.TCPConn, infoHash, peerID []byte) error {
	data := make([]byte, 68)
	copy(data[:28], handshakePrefix)
	copy(data[28:48], infoHash)
	copy(data[48:], peerID)

	conn.SetWriteDeadline(time.Now().Add(time.Second * 10))
	_, err := conn.Write(data)
	return err
}

// onHandshake handles the handshake response.
func onHandshake(data []byte) (err error) {
	if !(bytes.Equal(handshakePrefix[:20], data[:20]) && data[25]&0x10 != 0) {
		err = errors.New("invalid handshake response")
	}
	return
}

// sendExtHandshake requests for the ut_metadata and metadata_size.
func sendExtHandshake(conn *net.TCPConn) error {
	data := append(
		[]byte{EXTENDED, HANDSHAKE},
		Encode(map[string]interface{}{
			"m": map[string]interface{}{"ut_metadata": 1},
		})...,
	)

	return sendMessage(conn, data)
}

// getUTMetaSize returns the ut_metadata and metadata_size.
func getUTMetaSize(data []byte) (
	utMetadata int, metadataSize int, err error) {

	v, err := Decode(data)
	if err != nil {
		return
	}

	dict, ok := v.(map[string]interface{})
	if !ok {
		err = errors.New("invalid dict")
		return
	}

	if err = parseKeys(
		dict, [][]string{{"metadata_size", "int"}, {"m", "map"}}); err != nil {
		return
	}

	m := dict["m"].(map[string]interface{})
	if err = parseKey(m, "ut_metadata", "int"); err != nil {
		return
	}

	utMetadata = m["ut_metadata"].(int)
	metadataSize = dict["metadata_size"].(int)

	if metadataSize > MaxMetadataSize {
		err = errors.New("metadata_size too long")
	}
	return
}

func (bt *btClient) requestPieces(
	conn *net.TCPConn, utMetadata int, metadataSize int, piecesNum int) {

	buffer := make([]byte, 1024)
	for i := 0; i < piecesNum; i++ {
		buffer[0] = EXTENDED
		buffer[1] = byte(utMetadata)

		msg := Encode(map[string]interface{}{
			"msg_type": REQUEST,
			"piece":    i,
		})

		length := len(msg) + 2
		copy(buffer[2:length], msg)

		sendMessage(conn, buffer[:length])
	}
	buffer = nil
}

// fetchMetadata fetchs medata info accroding to infohash from dht.
func (bt *btClient) fetchMetadata(p peer) {
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
		return
	}
	conn := dial.(*net.TCPConn)
	conn.SetLinger(0)
	defer conn.Close()

	data := bytes.NewBuffer(nil)
	data.Grow(BLOCK)

	if sendHandshake(conn, []byte(infoHash), []byte(genInfoHash())) != nil ||
		read(conn, 68, data) != nil ||
		onHandshake(data.Next(68)) != nil ||
		sendExtHandshake(conn) != nil {
		return
	}

	for {
		length, err = readMessage(conn, data)
		if err != nil {
			return
		}

		if length == 0 {
			continue
		}

		msgType, err = data.ReadByte()
		if err != nil {
			return
		}

		switch msgType {
		case EXTENDED:
			extendedID, err := data.ReadByte()
			if err != nil {
				return
			}

			payload, err := ioutil.ReadAll(data)
			if err != nil {
				return
			}

			if extendedID == 0 {
				if pieces != nil {
					return
				}

				utMetadata, metadataSize, err = getUTMetaSize(payload)
				if err != nil {
					return
				}

				piecesNum = metadataSize / BLOCK
				if metadataSize%BLOCK != 0 {
					piecesNum++
				}

				pieces = make([][]byte, piecesNum)
				go bt.requestPieces(conn, utMetadata, metadataSize, piecesNum)

				continue
			}

			if pieces == nil {
				return
			}

			d, index, err := DecodeDict(payload, 0)
			if err != nil {
				return
			}
			dict := d.(map[string]interface{})

			if err = parseKeys(dict, [][]string{
				{"msg_type", "int"},
				{"piece", "int"}}); err != nil {
				return
			}

			if dict["msg_type"].(int) != DATA {
				continue
			}

			piece := dict["piece"].(int)
			pieceLen := length - 2 - index

			if (piece != piecesNum-1 && pieceLen != BLOCK) ||
				(piece == piecesNum-1 && pieceLen != metadataSize%BLOCK) {
				return
			}

			pieces[piece] = payload[index:]

			if bt.isDone(pieces) {
				metadataInfo := bytes.Join(pieces, nil)

				// Check the metadata
				info := sha1.Sum(metadataInfo)
				if !bytes.Equal([]byte(infoHash), info[:]) {
					fmt.Println("Metadata does not match infohash")
					return
				}

				torrent, err := decodeMetadata(p, metadataInfo)
				if err != nil {
					return
				}
				bt.torrentsOut <- *torrent
				return
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
			paths := f["path"].([]string)
			path := strings.Join(paths[:], "/")
			bt.Files[i] = File{
				Path:   path,
				Length: f["length"].(int),
			}
			fmt.Printf("got file %z\n", bt.Files[i])
		}
	} else if _, ok := info["length"]; ok {
		bt.Length = info["length"].(int)
	}
	return &bt, nil
}

// bytes2int returns the int value it represents.
func bytes2int(data []byte) uint64 {
	n, val := len(data), uint64(0)
	if n > 8 {
		panic("data too long")
	}

	for i, b := range data {
		val += uint64(b) << uint64((n-i-1)*8)
	}
	return val
}
