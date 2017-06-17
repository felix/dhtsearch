package main

import (
	"fmt"
	"time"
)

// Data for persistent storage
type Torrent struct {
	Id       int       `json:"-"`
	InfoHash string    `json:"infohash"`
	Name     string    `json:"name"`
	Files    []File    `json:"files" db:"-"`
	Size     int       `json:"size"`
	Seen     time.Time `json:"seen"`
	Tags     []string  `json:"tags" db:"-"`
}

type File struct {
	Id        int    `json:"-"`
	Path      string `json:"path"`
	Size      int    `json:"size"`
	TorrentId int    `json:"torrent_id" db:"torrent_id"`
}

func torrentExists(ih string) bool {
	rows, err := DB.Query(sqlGetTorrent, fmt.Sprintf("%s", ih))
	defer rows.Close()
	if err != nil {
		fmt.Printf("Failed to exec SQL: %q\n", err)
		return false
	}
	return rows.Next()
}

func (t *Torrent) save() error {
	tx, err := DB.Begin()
	if err != nil {
		fmt.Printf("Transaction err %q\n", err)
	}
	defer tx.Commit()

	var lastId int

	// Need to turn infohash into string here
	err = tx.QueryRow(sqlInsertTorrent, t.Name, fmt.Sprintf("%s", t.InfoHash), t.Size).Scan(&lastId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Write tags
	for _, tag := range t.Tags {
		tagId, err := createTag(tag)
		if err != nil {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec(sqlInsertTagTorrent, tagId, lastId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Write files
	for _, f := range t.Files {
		_, err := tx.Exec(sqlInsertFile, lastId, f.Path, f.Size)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return nil
}

// Fill in a torrents dependant data
func (t *Torrent) load() (err error) {
	// Files
	t.Files = []File{}
	err = DB.Select(&t.Files, sqlSelectFiles, t.Id)
	if err != nil {
		fmt.Printf("Error selecting files %s\n", err)
	}
	// t.Files = files

	// Tags
	t.Tags = []string{}
	err = DB.Select(&t.Tags, sqlSelectTags, t.Id)
	if err != nil {
		fmt.Printf("Error selecting tags %s\n", err)
	}
	return
}

func torrentsByName(query string) ([]Torrent, error) {
	torrents := []Torrent{}
	err := DB.Select(&torrents, sqlSearchTorrents, fmt.Sprintf("%%%s%%", query))
	if err != nil {
		return nil, err
	}
	fmt.Printf("Search for %q returned %d torrents\n", query, len(torrents))

	for idx, _ := range torrents {
		torrents[idx].load()
	}
	return torrents, nil
}

func torrentsByTag(tag string) ([]Torrent, error) {
	torrents := []Torrent{}
	err := DB.Select(&torrents, sqlTorrentsByTag, tag)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Search for tag %q returned %d torrents\n", tag, len(torrents))

	for idx, _ := range torrents {
		torrents[idx].load()
	}
	return torrents, nil
}

const (
	sqlGetTorrent = `update torrents
	set seen = now()
	where infohash = $1
	returning id`

	sqlInsertTorrent = `insert into torrents (
		name, infohash, size, seen
	) values (
		$1, $2, $3, now()
	) on conflict (infohash) do
	update set seen = now()
	returning id`

	sqlSearchTorrents = `select t.*
	from torrents t
	inner join files f on t.id = f.torrent_id
	where t.name ilike $1 or f.path ilike $1 group by t.id
	order by seen desc
	limit 50`

	sqlTorrentsByTag = `select t.*
	from torrents t
	inner join tags_torrents tt on t.id = tt.torrent_id
	inner join tags ta on tt.tag_id = ta.id
	where ta.name = $1 group by t.id
	order by seen desc
	limit 50`

	sqlSelectFiles = `select * from files
	where torrent_id = $1
	order by path asc`

	sqlInsertFile = `insert into files (
		torrent_id, path, size
	) values($1, $2, $3)`

	sqlSelectTags = `select name
	from tags t
	inner join tags_torrents tt on t.id = tt.tag_id
	where tt.torrent_id = $1`

	sqlInsertTagTorrent = `insert into tags_torrents (
		tag_id, torrent_id
	) values ($1, $2)
	on conflict do nothing`
)
