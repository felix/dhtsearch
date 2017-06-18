create table if not exists torrents (
    id serial not null primary key,
    infohash character varying(40) unique,
    size bigint,
    name text,
    seen timestamp with time zone,
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
    id serial not null primary key,
    name text
);
create table if not exists tags_torrents (
    tag_id integer not null references tags on delete cascade,
    torrent_id integer not null references torrents on delete cascade,
    primary key (tag_id, torrent_id)
);
