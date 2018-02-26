package models

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/felix/dhtsearch/bencode"
	"github.com/felix/dhtsearch/krpc"
)

// Data for persistent storage
type Torrent struct {
	ID       int       `json:"-"`
	Infohash string    `json:"infohash"`
	Name     string    `json:"name"`
	Files    []File    `json:"files" db:"-"`
	Size     int       `json:"size"`
	Seen     time.Time `json:"seen"`
	Tags     []string  `json:"tags" db:"-"`
}

type File struct {
	ID        int    `json:"-"`
	Path      string `json:"path"`
	Size      int    `json:"size"`
	TorrentID int    `json:"torrent_id" db:"torrent_id"`
}

func InfohashMatchesMetadata(ih Infohash, md []byte) bool {
	info := sha1.Sum(md)
	return bytes.Equal([]byte(ih), info[:])
}

func TorrentFromMetadata(ih Infohash, md []byte) (*Torrent, error) {
	if !InfohashMatchesMetadata(ih, md) {
		return nil, fmt.Errorf("infohash does not match metadata")
	}
	info, _, err := bencode.DecodeDict(md, 0)
	if err != nil {
		return nil, err
	}

	// Get the directory or advisory filename
	name, err := krpc.GetString(info, "name")
	if err != nil {
		return nil, err
	}

	bt := Torrent{
		Infohash: hex.EncodeToString([]byte(ih)),
		Name:     name,
	}

	if files, err := krpc.GetList(info, "files"); err == nil {
		// Multiple file mode
		bt.Files = make([]File, len(files))

		// Files is a list of dicts
		for i, item := range files {
			file := item.(map[string]interface{})

			// Paths is a list of strings
			paths := file["path"].([]interface{})
			path := make([]string, len(paths))
			for j, p := range paths {
				path[j] = p.(string)
			}

			fSize, err := krpc.GetInt(file, "length")
			if err != nil {
				return nil, err
			}
			bt.Files[i] = File{
				// Assume Unix path sep?
				Path: strings.Join(path[:], string(os.PathSeparator)),
				Size: fSize,
			}
			// Ensure the torrent size totals all files'
			bt.Size = bt.Size + fSize
		}
	} else if length, err := krpc.GetInt(info, "length"); err == nil {
		// Single file mode
		bt.Size = length
	} else {
		return nil, fmt.Errorf("found neither length or files")
	}
	return &bt, nil
}
