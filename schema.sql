create table if not exists torrents (
    id serial not null primary key,
    infohash character varying(40) unique,
    size bigint,
    name text,
    seen timestamp with time zone
);
create table if not exists files (
    id serial not null primary key,
    torrent_id integer not null references torrents,
    path text,
    size bigint
);
create table if not exists tags (
    id serial not null primary key,
    name text
);
create table if not exists tags_torrents (
    tag_id integer not null references tags,
    torrent_id integer not null references torrents,
    primary key (tag_id, torrent_id)
);
