// /home/krylon/go/src/github.com/blicero/newsroom/database/qinit.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-02 14:03:43 krylon>

package database

var qInit = []string{
	`
CREATE TABLE feed (
    id			INTEGER PRIMARY KEY,
    name		TEXT UNIQUE NOT NULL,
    language		TEXT NOT NULL,
    url			TEXT UNIQUE NOT NULL,
    homepage		TEXT NOT NULL,
    refresh_interval	INTEGER NOT NULL,
    last_refresh	INTEGER NOT NULL DEFAULT 0,
    paused		INTEGER NOT NULL DEFAULT 0
) STRICT
`,
	"CREATE INDEX feed_ref_idx ON feed (last_refresh)",
	"CREATE INDEX feed_pause_idx ON feed (paused)",
	"CREATE INDEX feed_due_idx ON feed (last_refresh + refresh_interval)",
	`
CREATE TABLE item (
    id			INTEGER PRIMARY KEY,
    feed_id		INTEGER NOT NULL,
    title		TEXT NOT NULL,
    url			TEXT UNIQUE NOT NULL,
    rating		INTEGER NOT NULL DEFAULT 0,
    timestamp		INTEGER NOT NULL,
    body		TEXT NOT NULL,
    FOREIGN KEY (feed_id) REFERENCES feed (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE
) STRICT
`,
	"CREATE INDEX item_feed_idx ON item (feed_id)",
	"CREATE INDEX item_time_idx ON item (timestamp)",
	`
CREATE TABLE tag (
    id			INTEGER PRIMARY KEY,
    parent              INTEGER,
    name		TEXT UNIQUE NOT NULL,
    FOREIGN KEY (parent) REFERENCES tag (id)
        ON UPDATE RESTRICT
        ON DELETE RESTRICT,
    CHECK (parent <> id)
) STRICT`,
	`
CREATE TABLE tag_link (
    id			INTEGER PRIMARY KEY,
    tag_id		INTEGER NOT NULL,
    item_id		INTEGER NOT NULL,
    UNIQUE (tag_id, item_id),
    FOREIGN KEY (tag_id) REFERENCES tag (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE,
    FOREIGN KEY (item_id) REFERENCES item (id)
        ON UPDATE RESTRICT
        ON DELETE CASCADE
) STRICT
`,
}
