package dhtsearch

import (
	"fmt"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

type database struct {
	*sqlx.DB
}

// Global
var DB *database

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
	return &database{d}, nil
}
