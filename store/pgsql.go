//go:build ignore
// +build ignore

package store

import (
	"fmt"
	"net"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
	"src.userspace.com.au/dhtsearch/models"
)

// Store is a store
type Store struct {
	*pgx.ConnPool
}

// NewStore connects and initializes a new store
func NewStore(dsn string) (*Store, error) {
	cfg, err := pgx.ParseURI(dsn)
	if err != nil {
		return nil, err
	}
	c, err := pgx.NewConnPool(pgx.ConnPoolConfig{ConnConfig: cfg, MaxConnections: 10})
	if err != nil {
		return nil, err
	}

	s := &Store{c}

	err = s.migrate()
	if err != nil {
		return nil, err
	}

	err = s.prepareStatements()
	if err != nil {
		return nil, err
	}

	return s, err
}

// PendingInfohashes gets the next pending infohash from the store
func (s *Store) PendingInfohashes(n int) (peers []*models.Peer, err error) {

	rows, err := s.Query("selectPendingInfohashes", n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p models.Peer
		var ih pgtype.Bytea
		var addr string
		err = rows.Scan(&addr, &ih)
		if err != nil {
			return nil, err
		}
		// TODO save peer network?
		p.Addr, err = net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return nil, err
		}
		ih.AssignTo(&p.Infohash)
		peers = append(peers, &p)
	}
	return peers, nil
}

// SaveTorrent implements torrentStore
func (s *Store) SaveTorrent(t *models.Torrent) error {
	tx, err := s.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var torrentID int
	err = tx.QueryRow("insertTorrent", t.Name, t.Infohash, t.Size).Scan(&torrentID)
	if err != nil {
		return fmt.Errorf("insertTorrent: %s", err)
	}

	// Write tags
	for _, tag := range t.Tags {
		tagID, err := s.SaveTag(tag)
		if err != nil {
			return fmt.Errorf("saveTag: %s", err)
		}
		_, err = tx.Exec("insertTagTorrent", tagID, torrentID)
		if err != nil {
			return fmt.Errorf("insertTagTorrent: %s", err)
		}
	}

	// Write files
	for _, f := range t.Files {
		_, err := tx.Exec("insertFile", torrentID, f.Path, f.Size)
		if err != nil {
			return fmt.Errorf("insertFile: %s", err)
		}
	}

	// Should this be outside the transaction?
	_, err = tx.Exec("updateFTSVectors", torrentID)
	if err != nil {
		return fmt.Errorf("updateVectors: %s", err)
	}
	return tx.Commit()
}

func (s *Store) RemoveTorrent(t *models.Torrent) (err error) {
	_, err = s.Exec("removeTorrent", t.Infohash.Bytes())
	return err
}

// SavePeer implements torrentStore
func (s *Store) SavePeer(p *models.Peer) (err error) {
	_, err = s.Exec("insertPeer", p.Addr.String(), p.Infohash.Bytes())
	return err
}

func (s *Store) RemovePeer(p *models.Peer) (err error) {
	_, err = s.Exec("removePeer", p.Addr.String())
	return err
}

// TorrentsByHash implements torrentStore
func (s *Store) TorrentByHash(ih models.Infohash) (*models.Torrent, error) {
	rows, err := s.Query("getTorrent", ih)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	torrents, err := s.fetchTorrents(rows)
	if err != nil {
		return nil, err
	}
	return torrents[0], nil
}

// TorrentsByName implements torrentStore
func (s *Store) TorrentsByName(query string, offset int) ([]*models.Torrent, error) {
	rows, err := s.Query("searchTorrents", fmt.Sprintf("%%%s%%", query), offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	torrents, err := s.fetchTorrents(rows)
	if err != nil {
		return nil, err
	}
	return torrents, nil
}

// TorrentsByTag implements torrentStore
func (s *Store) TorrentsByTag(tag string, offset int) ([]*models.Torrent, error) {
	rows, err := s.Query("torrentsByTag", tag, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	torrents, err := s.fetchTorrents(rows)
	if err != nil {
		return nil, err
	}
	return torrents, nil
}

// SaveTag implements tagStore interface
func (s *Store) SaveTag(tag string) (tagID int, err error) {
	err = s.QueryRow("insertTag", tag).Scan(&tagID)
	return tagID, err
}

func (s *Store) fetchTorrents(rows *pgx.Rows) (torrents []*models.Torrent, err error) {
	for rows.Next() {
		var t models.Torrent
		/*
			t := &models.Torrent{
				Files: []models.File{},
				Tags:  []string{},
			}
		*/
		err = rows.Scan(
			&t.ID, &t.Infohash, &t.Name, &t.Size, &t.Created, &t.Updated,
		)
		if err != nil {
			return nil, err
		}

		err = func() error {
			rowsf, err := s.Query("selectFiles", t.ID)
			defer rowsf.Close()
			if err != nil {
				return fmt.Errorf("failed to select files: %s", err)
			}
			for rowsf.Next() {
				var f models.File
				err = rowsf.Scan(&f.ID, &f.TorrentID, &f.Path, &f.Size)
				if err != nil {
					return fmt.Errorf("failed to build file: %s", err)
				}
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}

		err = func() error {
			rowst, err := s.Query("selectTags", t.ID)
			defer rowst.Close()
			if err != nil {
				return fmt.Errorf("failed to select tags: %s", err)
			}
			for rowst.Next() {
				var tg string
				err = rowst.Scan(&tg)
				if err != nil {
					return fmt.Errorf("failed to build tag: %s", err)
				}
				t.Tags = append(t.Tags, tg)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
		torrents = append(torrents, &t)
	}
	return torrents, err
}

func (s *Store) migrate() error {
	tx, err := s.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var initialized bool
	err = tx.QueryRow(`select exists (
		select 1 from pg_tables
		where schemaname = 'public'
		and tablename = 'settings'
	)`).Scan(&initialized)
	if err != nil {
		return err
	}

	if !initialized {
		_, err = tx.Exec("baseSchema")
	}

	// Start migrations
	var currentVersion int
	err = tx.QueryRow("select schema_version from settings").Scan(currentVersion)
	if err != nil {
		return err
	}

	switch currentVersion {
	case 1:
	default:
	}

	return nil
}

func (s *Store) prepareStatements() error {
	if _, err := s.Prepare(
		"removeTorrent",
		`delete from torrents
		where infohash = $1`,
	); err != nil {
		return err
	}
	if _, err := s.Prepare(
		"selectPendingInfohashes",
		`with get_order as (
			select t.id as torrent_id, min(pt.peer_id) as peer_id, count(pt.peer_id) as c
			from torrents t
			join peers_torrents pt on pt.torrent_id = t.id
			where t.name is null
			group by t.id
			-- order by c desc
			order by t.updated desc
			limit $1
		) select p.address, t.infohash
		from get_order go
		join torrents t on t.id = go.torrent_id
		join peers p on p.id = go.peer_id`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"selectFiles",
		`select * from files
		where torrent_id = $1
		order by path asc`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"insertPeer",
		`with save_peer as (
			insert into peers
			(address, created, updated) values ($1, now(), now())
			returning id
		), save_torrent as (
			insert into torrents (infohash, created, updated)
			values ($2, now(), now())
			on conflict (infohash) do update set
			updated = now()
			returning id
		) insert into peers_torrents
		(peer_id, torrent_id)
		select
		sp.id, st.id
		from save_peer sp, save_torrent st
		on conflict do nothing`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"getTorrent",
		`select * from torrents where infohash = $1 limit 1`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"insertFile",
		`insert into files
		(torrent_id, path, size)
		values
		($1, $2, $3)`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"selectTags",
		`select name
		from tags t
		inner join tags_torrents tt on t.id = tt.tag_id
		where tt.torrent_id = $1`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"removePeer",
		`delete from peers where address = $1`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"insertTagTorrent",
		`insert into tags_torrents
		(tag_id, torrent_id) values ($1, $2)
		on conflict do nothing`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"insertTag",
		`insert into tags (name) values ($1)
		on conflict (name) do update set name = excluded.name returning id`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"torrentsByTag",
		`select t.id, t.infohash, t.name, t.size, t.created, t.updated
		from torrents t
		inner join tags_torrents tt on t.id = tt.torrent_id
		inner join tags ta on tt.tag_id = ta.id
		where ta.name = $1 group by t.id
		order by updated asc
		limit 50 offset $2`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"searchTorrents",
		`select t.id, t.infohash, t.name, t.size, t.updated
		from torrents t
		where t.tsv @@ plainto_tsquery($1)
		order by ts_rank(tsv, plainto_tsquery($1)) desc, t.updated desc
		limit 50 offset $2`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"insertTorrent",
		`insert into torrents (
			name, infohash, size, created, updated
		) values (
			$1, $2, $3, now(), now()
		) on conflict (infohash) do
		update set
		name = $1,
		size = $3,
		updated = now()
		returning id`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"updateFTSVectors",
		`update torrents set
		tsv = sub.tsv from (
			select t.id,
			setweight(to_tsvector(
				translate(t.name, '._-', ' ')
			), 'A')
			|| setweight(to_tsvector(
				translate(string_agg(coalesce(f.path, ''), ' '), './_-', ' ')
			), 'B') as tsv
			from torrents t
			left join files f on t.id = f.torrent_id
			where t.id = $1
			group by t.id
		) as sub
		where sub.id = torrents.id`,
	); err != nil {
		return err
	}

	if _, err := s.Prepare(
		"baseSchema",
		`create table if not exists torrents (
			id serial primary key,
			infohash bytea not null unique,
			size bigint,
			name text,
			created timestamp with time zone,
			updated timestamp with time zone,
			tsv tsvector
		);
		create index tsv_idx on torrents using gin(tsv);
		create table if not exists files (
			id serial not null primary key,
			torrent_id integer not null references torrents on delete cascade,
			path text,
			size bigint
		);
		create table if not exists tags (
			id serial primary key,
			name character varying(50) unique
		);
		create table if not exists tags_torrents (
			tag_id integer not null references tags (id) on delete cascade,
			torrent_id integer not null references torrents (id) on delete cascade,
			primary key (tag_id, torrent_id)
		);
		create table if not exists peers (
			id serial primary key,
			address character varying(50) not null,
			created timestamp with time zone,
			updated timestamp with time zone
		);
		create table if not exists peers_torrents (
			peer_id integer not null references peers (id) on delete cascade,
			torrent_id integer not null references torrents (id) on delete cascade,
			primary key (peer_id, torrent_id)
		);
		create table if not exists settings (
			schema_version integer not null
		);
		insert into settings (schema_version) values (1);`,
	); err != nil {
		return err
	}
	return nil
}
