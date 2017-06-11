package main

import (
	//"database/sql"
	"fmt"
	//_ "github.com/lib/pq"
	"github.com/jackc/pgx"
)

type db struct {
	*pgx.ConnPool
	debug bool
}

func newDB(dsn string) (*db, error) {
	pgxConfig, err := pgx.ParseConnectionString(dsn)
	if err != nil {
		fmt.Printf("Error creating DB config %q\n", err)
		return nil, err
	}
	d, err := pgx.NewConnPool(pgx.ConnPoolConfig{pgxConfig, 3, nil, 0})
	if err != nil {
		fmt.Printf("Error creating DB %q\n", err)
		return nil, err
	}
	var count int
	err = d.QueryRow("select count(*) from torrents").Scan(&count)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Found %d existing torrents\n", count)
	return &db{d, false}, nil
}

func (d *db) saveTorrent(t Torrent) error {
	tx, err := d.Begin()
	if err != nil {
		fmt.Printf("Transaction err %q\n", err)
	}
	defer tx.Commit()

	var lastId int

	err = tx.QueryRow(`insert into torrents (name, infohash, size, seen) values($1, $2, $3, now()) on conflict (infohash) do update set seen = now() returning id`, t.Name, fmt.Sprintf("%s", t.InfoHash), t.Length).Scan(&lastId)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Write tags
	for _, tag := range t.Tags {
		tagId, err := d.createTag(tag)
		if err != nil {
			tx.Rollback()
			return err
		}
		_, err = tx.Exec("insert into tags_torrents (tag_id, torrent_id) values ($1, $2) on conflict do nothing", tagId, lastId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	// Write files
	for _, f := range t.Files {
		_, err := tx.Exec(`insert into files (torrent_id, path, size) values($1, $2, $3)`, lastId, f.Path, f.Length)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return nil
}

func (d *db) torrentExists(ih string) bool {
	rows, err := d.Query("select seen from torrents where infohash = $1", fmt.Sprintf("%s", ih))
	defer rows.Close()
	if err != nil {
		fmt.Printf("Failed to exec SQL: %q\n", err)
		return false
	}
	return rows.Next()
}

func (d *db) createTag(tag string) (tagId int, err error) {
	if d.debug {
		fmt.Printf("Writing tag %s\n", tag)
	}

	err = d.QueryRow("select id from tags where name = $1", tag).Scan(&tagId)
	if err == nil {
		if d.debug {
			fmt.Printf("Found existing tag %s\n", tag)
		}
	} else {
		err = d.QueryRow("insert into tags (name) values ($1) returning id", tag).Scan(&tagId)
		if err != nil {
			fmt.Println(err)
			return -1, err
		}
		if d.debug {
			fmt.Printf("Created new tag %s\n", tag)
		}
	}
	return tagId, nil
}
