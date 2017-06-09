package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

const (
	sqlInit = `
	create table if not exists torrents (
		id integer not null primary key,
		ih text unique,
		size int,
		name text,
		seen text
	);
	create table if not exists files (
		id integer not null primary key,
		torrent_id integer,
		path text,
		size int,
		foreign key(torrent_id) references torrents(id)
	);
	`
)

type db struct {
	*sql.DB
}

func newDB() (*db, error) {
	d, err := sql.Open("sqlite3", "./torrents.db")
	if err != nil {
		fmt.Printf("Error creating DB %q\n", err)
		return nil, err
	}
	_, err = d.Exec(sqlInit)
	if err != nil {
		fmt.Printf("Failed to init DB %q\n", err)
		return nil, err
	}
	_, err = d.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		fmt.Printf("Failed to configure DB %q\n", err)
		return nil, err
	}
	return &db{d}, nil
}

func (d *db) updateTorrent(t Torrent) error {
	tx, err := d.Begin()
	if err != nil {
		fmt.Printf("Transaction err %q\n", err)
	}
	stmt, err := tx.Prepare(`insert or replace into torrents (
		name, ih, size, seen) values(?, ?, ?, date('now'))`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	r, err := stmt.Exec(t.Name, t.InfoHash, t.Length)
	if err != nil {
		return err
	}

	stmt, err = tx.Prepare(`insert into files (
		torrent_id, path, size) values(?, ?, ?)`)
	if err != nil {
		return err
	}
	lastId, err := r.LastInsertId()
	if err != nil {
		return err
	}

	for _, f := range t.Files {
		_, err = stmt.Exec(lastId, f.Path, f.Length)
		if err != nil {
			return err
		}
	}
	tx.Commit()
	return nil
}

func (d *db) torrentExists(ih string) bool {
	stmt, err := d.Prepare("select seen from torrents where ih = ?")
	if err != nil {
		fmt.Printf("Failed to prepare select: %q\n", err)
		return false
	}
	defer stmt.Close()

	rows, err := stmt.Query(ih)
	if err != nil {
		fmt.Printf("Failed to exec select: %q\n", err)
		return false
	}
	defer rows.Close()
	return rows.Next()
}
