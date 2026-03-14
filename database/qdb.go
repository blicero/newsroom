// /home/krylon/go/src/github.com/blicero/newsroom/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-03-14 13:23:54 krylon>

package database

import "github.com/blicero/newsroom/database/query"

var qdb = map[query.ID]string{
	query.FeedAdd: `
INSERT INTO feed (name, language, url, homepage, refresh_interval)
          VALUES (   ?,        ?,   ?,        ?,                ?)
RETURNING id
`,
	query.FeedDelete: "DELETE FROM feed WHERE id = ?",
	query.FeedGetAll: `
SELECT
    id,
    name,
    language,
    url,
    homepage,
    refresh_interval,
    last_refresh,
    paused
FROM feed
`,
	query.FeedGetDue: `
SELECT
    id,
    name,
    language,
    url,
    homepage,
    refresh_interval,
    last_refresh,
    paused
FROM feed
WHERE paused = 0 AND last_refresh + refresh_interval < unixepoch()
ORDER BY last_refresh
`,
	query.FeedGetByID: `
SELECT
    name,
    language,
    url,
    homepage,
    refresh_interval,
    last_refresh,
    paused
FROM feed
WHERE id = ?
`,
	query.FeedSetInterval:    "UPDATE feed SET refresh_interval = ? WHERE id = ?",
	query.FeedSetLastRefresh: "UPDATE feed SET last_refresh = ? WHERE id = ?",
	query.FeedSetPause:       "UPDATE feed SET paused = ? WHERE id = ?",
	query.ItemAdd: `
INSERT INTO item (feed_id, title, url, timestamp, body)
          VALUES (      ?,     ?,   ?,         ?,    ?)
RETURNING id
`,
	query.ItemGetByID: `
SELECT
    feed_id,
    url,
    title,
    rating,
    timestamp,
    body
FROM item
WHERE id = ?
`,
	query.ItemGetByURL: `
SELECT
    id,
    feed_id,
    title,
    rating,
    timestamp,
    body
FROM item
WHERE url = ?
`,
	query.ItemGetAll: `
SELECT
    id,
    feed_id,
    title,
    url,
    rating,
    timestamp,
    body
FROM item
ORDER BY timestamp DESC
LIMIT ? OFFSET ?
`,
	query.ItemGetByFeed: `
SELECT
    id,
    title,
    url,
    rating,
    timestamp,
    body
FROM item
WHERE feed_id = ?
ORDER BY timestamp DESC
`,
	query.ItemCount:     "SELECT COUNT(id) FROM item",
	query.ItemSetRating: "UPDATE item SET rating = ? WHERE id = ?",
	query.TagAdd: `
INSERT INTO tag (name)
         VALUES (   ?)
RETURNING id
`,
	query.TagGetAll:  "SELECT id, name FROM tag",
	query.TagGetByID: "SELECT name FROM tag WHERE id = ?",
	query.TagDelete:  "DELETE FROM tag WHERE id = ?",
	query.TagLinkAdd: `
INSERT INTO tag_link (tag_id, item_id) VALUES (?, ?) RETURNING id
`,
	query.TagLinkDelete: "DELETE FROM tag_link WHERE id = ?",
	query.TagLinkGetByItem: `
SELECT
    id,
    tag_id
FROM tag_link
WHERE item_id = ?
`,
	query.TagLinkGetByTag: `
SELECT
    id,
    item_id
FROM tag_link
WHERE tag_id = ?
`,
}
