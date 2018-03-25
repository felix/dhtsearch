package db

import (
	"database/sql"
	"fmt"
	"net"
	"sync"

	"github.com/felix/dhtsearch/models"
	_ "github.com/mattn/go-sqlite3"
)

// Store is a store
type Store struct {
	stmts map[string]*sql.Stmt
	conn  *sql.DB
	lock  sync.RWMutex
}

// NewStore connects and initializes a new store
func NewStore(dsn string) (*Store, error) {
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open store: %s", err)
	}

	s := &Store{conn: conn, stmts: make(map[string]*sql.Stmt)}

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

func (s *Store) Close() error {
	return s.conn.Close()
}

// PendingInfohashes gets the next pending infohash from the store
func (s *Store) PendingInfohashes(n int) (peers []*models.Peer, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	rows, err := s.stmts["selectPendingInfohashes"].Query(n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p models.Peer
		var ih models.Infohash
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
		p.Infohash = ih
		peers = append(peers, &p)
	}
	return peers, nil
}

// SaveTorrent implements torrentStore
func (s *Store) SaveTorrent(t *models.Torrent) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	tx, err := s.conn.Begin()
	if err != nil {
		return fmt.Errorf("saveTorrent: %s", err)
	}
	defer tx.Rollback()

	var torrentID int64
	var res sql.Result
	res, err = tx.Stmt(s.stmts["insertTorrent"]).Exec(t.Name, t.Infohash.Bytes(), t.Size)
	if err != nil {
		return fmt.Errorf("insertTorrent: %s", err)
	}
	if torrentID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("insertTorrent: %s", err)
	}

	// Write tags
	for _, tag := range t.Tags {
		var tagID int64

		res, err = tx.Stmt(s.stmts["insertTag"]).Exec(tag)
		if err != nil {
			return fmt.Errorf("saveTag: %s", err)
		}
		tagID, err = res.LastInsertId()
		if err != nil {
			return fmt.Errorf("saveTag: %s", err)
		}
		_, err = tx.Stmt(s.stmts["insertTagTorrent"]).Exec(tagID, torrentID)
		if err != nil {
			return fmt.Errorf("insertTagTorrent: %s", err)
		}
	}

	// Write files
	for _, f := range t.Files {
		_, err := tx.Stmt(s.stmts["insertFile"]).Exec(torrentID, f.Path, f.Size)
		if err != nil {
			return fmt.Errorf("insertFile: %s", err)
		}
	}

	return tx.Commit()
}

func (s *Store) RemoveTorrent(t *models.Torrent) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, err = s.stmts["removeTorrent"].Exec(t.Infohash)
	return fmt.Errorf("removeTorrent: %s", err)
}

// SavePeer implements torrentStore
func (s *Store) SavePeer(p *models.Peer) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var peerID int64
	var torrentID int64
	var res sql.Result

	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if res, err = tx.Stmt(s.stmts["insertPeer"]).Exec(p.Addr.String()); err != nil {
		return fmt.Errorf("savePeer: %s", err)
	}
	if peerID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("savePeer: %s", err)
	}

	if res, err = tx.Stmt(s.stmts["insertTorrent"]).Exec(nil, p.Infohash, 0); err != nil {
		return fmt.Errorf("savePeer: %s", err)
	}
	if torrentID, err = res.LastInsertId(); err != nil {
		return fmt.Errorf("savePeer: %s", err)
	}

	if _, err = tx.Stmt(s.stmts["insertPeerTorrent"]).Exec(peerID, torrentID); err != nil {
		return fmt.Errorf("savePeer: %s", err)
	}
	return tx.Commit()
}

func (s *Store) RemovePeer(p *models.Peer) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, err = s.stmts["removePeer"].Exec(p.Addr.String())
	return err
}

// TorrentsByHash implements torrentStore
func (s *Store) TorrentByHash(ih models.Infohash) (*models.Torrent, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	rows, err := s.stmts["getTorrent"].Query(ih)
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
	s.lock.RLock()
	defer s.lock.RUnlock()

	rows, err := s.stmts["searchTorrents"].Query(fmt.Sprintf("%%%s%%", query), offset)
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
	s.lock.RLock()
	defer s.lock.RUnlock()

	rows, err := s.stmts["torrentsByTag"].Query(tag, offset)
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
func (s *Store) SaveTag(tag string) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	res, err := s.stmts["insertTag"].Exec(tag)
	if err != nil {
		return 0, fmt.Errorf("saveTag: %s", err)
	}
	tagID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("saveTag: %s", err)
	}
	return int(tagID), nil
}

func (s *Store) fetchTorrents(rows *sql.Rows) (torrents []*models.Torrent, err error) {
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
			rowsf, err := s.stmts["selectFiles"].Query(t.ID)
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
			rowst, err := s.stmts["selectTags"].Query(t.ID)
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
	_, err := s.conn.Exec(`
	pragma journal_mode=wal;
	pragma temp_store=1;
	pragma foreign_keys=on;
	pragma encoding='utf-8';
	`)
	if err != nil {
		return err
	}

	tx, err := s.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var version int
	err = tx.QueryRow("pragma user_version;").Scan(&version)
	if err != nil {
		return err
	}

	if version == 0 {
		_, err = tx.Exec(sqliteSchema)
		if err != nil {
			return err
		}
	}
	tx.Commit()

	return nil
}

func (s *Store) prepareStatements() error {
	var err error
	if s.stmts["removeTorrent"], err = s.conn.Prepare(
		`delete from torrents
		where infohash = ?`,
	); err != nil {
		return err
	}

	if s.stmts["selectPendingInfohashes"], err = s.conn.Prepare(
		`select max(p.address) as address, t.infohash
		from torrents t
		join peers_torrents pt on pt.torrent_id = t.id
		join peers p on p.id = pt.peer_id
		where t.name is null
		group by t.infohash`,
	); err != nil {
		return err
	}

	if s.stmts["selectFiles"], err = s.conn.Prepare(
		`select * from files
		where torrent_id = ?
		order by path asc`,
	); err != nil {
		return err
	}

	if s.stmts["insertPeer"], err = s.conn.Prepare(
		`insert or ignore into peers
		(address, created, updated)
		values
		(?, date('now'), date('now'))`,
	); err != nil {
		return err
	}

	if s.stmts["insertPeerTorrent"], err = s.conn.Prepare(
		`insert or ignore into peers_torrents
		(peer_id, torrent_id)
		values
		(?, ?)`,
	); err != nil {
		return err
	}

	if s.stmts["insertTorrent"], err = s.conn.Prepare(
		`insert or replace into torrents (
			name, infohash, size, created, updated
		) values (
			?, ?, ?, date('now'), date('now')
		)`,
	); err != nil {
		return err
	}

	if s.stmts["getTorrent"], err = s.conn.Prepare(
		`select * from torrents where infohash = ? limit 1`,
	); err != nil {
		return err
	}

	if s.stmts["insertFile"], err = s.conn.Prepare(
		`insert into files
		(torrent_id, path, size)
		values
		(?, ?, ?)`,
	); err != nil {
		return err
	}

	if s.stmts["selectTags"], err = s.conn.Prepare(
		`select name
		from tags t
		inner join tags_torrents tt on t.id = tt.tag_id
		where tt.torrent_id = ?`,
	); err != nil {
		return err
	}

	if s.stmts["removePeer"], err = s.conn.Prepare(
		`delete from peers where address = ?`,
	); err != nil {
		return err
	}

	if s.stmts["insertTagTorrent"], err = s.conn.Prepare(
		`insert or ignore into tags_torrents
		(tag_id, torrent_id) values (?, ?)`,
	); err != nil {
		return err
	}

	if s.stmts["insertTag"], err = s.conn.Prepare(
		`insert or replace into tags (name) values (?)`,
	); err != nil {
		return err
	}

	if s.stmts["torrentsByTag"], err = s.conn.Prepare(
		`select t.id, t.infohash, t.name, t.size, t.created, t.updated
		from torrents t
		inner join tags_torrents tt on t.id = tt.torrent_id
		inner join tags ta on tt.tag_id = ta.id
		where ta.name = ? group by t.id
		order by updated asc
		limit 50 offset ?`,
	); err != nil {
		return err
	}

	if s.stmts["searchTorrents"], err = s.conn.Prepare(
		`select id, infohash, name, size, updated
		from torrents
		where id in (
			select * from torrents_fts
			where torrents_fts match ?
			order by rank desc
		)
		order by updated desc
		limit 50 offset ?`,
	); err != nil {
		return err
	}

	return nil
}

const sqliteSchema = `create table if not exists torrents (
	id integer primary key,
	infohash blob not null unique,
	size bigint,
	name text,
	created timestamp with time zone,
	updated timestamp with time zone,
	tsv tsvector
);
create virtual table torrents_fts using fts5(
	name, content='torrents', content_rowid='id',
	tokenize="porter unicode61 separators ' !""#$%&''()*+,-./:;<=>?@[\]^_` + "`" + `{|}~'"
);
create trigger torrents_after_insert after insert on torrents begin
insert into torrents_fts(rowid, name) values (new.id, new.name);
end;
create trigger torrents_ad after delete on torrents begin
insert into torrents_fts(torrents_fts, rowid, name) values('delete', old.id, old.name);
end;
create trigger torrents_au after update on torrents begin
insert into torrents_fts(torrents_fts, rowid, name) values('delete', old.id, old.name);
insert into torrents_fts(rowid, name) values (new.id, new.name);
end;
create table if not exists files (
	id integer primary key,
	torrent_id integer not null references torrents on delete cascade,
	path text,
	size bigint
);
create index files_torrent_idx on files (torrent_id);
create table if not exists tags (
	id integer primary key,
	name character varying(50) unique
);
create unique index tags_name_idx on tags (name);
create table if not exists tags_torrents (
	tag_id integer not null references tags on delete cascade,
	torrent_id integer not null references torrents on delete cascade,
	primary key (tag_id, torrent_id)
);
create index tags_torrents_tag_idx on tags_torrents (tag_id);
create index tags_torrents_torrent_idx on tags_torrents (torrent_id);
create table if not exists peers (
	id integer primary key,
	address character varying(50) not null unique,
	created timestamp with time zone,
	updated timestamp with time zone
);
create table if not exists peers_torrents (
	peer_id integer not null references peers on delete cascade,
	torrent_id integer not null references torrents on delete cascade,
	primary key (peer_id, torrent_id)
);
create index peers_torrents_peer_idx on peers_torrents (peer_id);
create index peers_torrents_torrent_idx on peers_torrents (torrent_id);
pragma user_version = 1;`
