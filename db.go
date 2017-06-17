package main

import (
	"fmt"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

type database struct {
	*sqlx.DB
	debug bool
}

func newDB(dsn string) (*database, error) {
	d, err := sqlx.Connect("pgx", dsn)
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
	return &database{d, false}, nil
}

func createTag(tag string) (tagId int, err error) {
	if DB.debug {
		fmt.Printf("Writing tag %s\n", tag)
	}

	err = DB.QueryRow("select id from tags where name = $1", tag).Scan(&tagId)
	if err == nil {
		if DB.debug {
			fmt.Printf("Found existing tag %s\n", tag)
		}
	} else {
		err = DB.QueryRow("insert into tags (name) values ($1) returning id", tag).Scan(&tagId)
		if err != nil {
			fmt.Println(err)
			return -1, err
		}
		if DB.debug {
			fmt.Printf("Created new tag %s\n", tag)
		}
	}
	return tagId, nil
}
