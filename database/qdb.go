// /home/krylon/go/src/github.com/blicero/newsroom/database/qdb.go
// -*- mode: go; coding: utf-8; -*-
// Created on 09. 03. 2026 by Benjamin Walkenhorst
// (c) 2026 Benjamin Walkenhorst
// Time-stamp: <2026-04-20 13:46:56 krylon>

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
    active
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
    active
FROM feed
WHERE active = 1 AND last_refresh + refresh_interval < unixepoch()
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
    active
FROM feed
WHERE id = ?
`,
	query.FeedSetInterval:    "UPDATE feed SET refresh_interval = ? WHERE id = ?",
	query.FeedSetLastRefresh: "UPDATE feed SET last_refresh = ? WHERE id = ?",
	query.FeedSetActive:      "UPDATE feed SET active = ? WHERE id = ?",
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
	query.ItemGetRated: `
SELECT
    id,
    feed_id,
    title,
    url,
    rating,
    timestamp,
    body
FROM item
WHERE rating <> 0
`,
	query.ItemCount:     "SELECT COUNT(id) FROM item",
	query.ItemSetRating: "UPDATE item SET rating = ? WHERE id = ?",
	query.ItemSearchPlain: `
SELECT
    id,
    feed_id,
    title,
    url,
    rating,
    timestamp,
    body
FROM item
WHERE (title LIKE ?) OR (body LIKE ?)
ORDER BY timestamp DESC
`,
	query.ItemSearchDate: `
SELECT
    id,
    feed_id,
    title,
    url,
    rating,
    timestamp,
    body
FROM item
WHERE (timestamp BETWEEN ? AND ?)
  AND (title LIKE ? OR body LIKE ?)
ORDER BY timestamp DESC
`,
	query.ItemSearchTag: `
SELECT DISTINCT
    i.id,
    i.feed_id,
    i.title,
    i.url,
    i.rating,
    i.timestamp,
    i.body
FROM tag_link l
INNER JOIN item i ON l.item_id = i.id
WHERE (l.tag_id IN (?))
  AND (title LIKE ? OR body LIKE ?)
ORDER BY timestamp DESC
`,
	query.ItemSearchDateTag: `
SELECT DISTINCT
    i.id,
    i.feed_id,
    i.title,
    i.url,
    i.rating,
    i.timestamp,
    i.body
FROM tag_link l
INNER JOIN item i ON l.item_id = i.id
WHERE (i.timestamp BETWEEN ? AND ?)
  AND (l.tag_id IN (?))
  AND (title LIKE ? OR body LIKE ?)
ORDER BY timestamp DESC
`,
	query.TagAdd: `
INSERT INTO tag (name, parent)
         VALUES (   ?,      ?)
RETURNING id
`,
	query.TagGetAll:  "SELECT id, name, COALESCE(parent, 0) FROM tag",
	query.TagGetByID: "SELECT name, COALESCE(parent, 0) FROM tag WHERE id = ?",
	query.TagGetSorted: `
WITH RECURSIVE children(id, name, lvl, root, parent, full_name) AS (
    SELECT
        id,
        name,
        0 AS lvl,
        id AS root,
        COALESCE(parent, 0) AS parent,
        name AS full_name
    FROM tag
    WHERE COALESCE(parent, 0) = 0
    UNION ALL
    SELECT
        tag.id,
        tag.name,
        lvl + 1 AS lvl,
        children.root,
        tag.parent,
        full_name || '/' || tag.name AS full_name
    FROM tag, children
    WHERE tag.parent = children.id
)

SELECT
        id,
        name,
        COALESCE(parent, 0),
        lvl,
        full_name
FROM children
ORDER BY full_name
`,
	query.TagDelete:    "DELETE FROM tag WHERE id = ?",
	query.TagSetParent: "UPDATE tag SET parent = ? WHERE id = ?",
	query.TagLinkAdd: `
INSERT INTO tag_link (tag_id, item_id) VALUES (?, ?) RETURNING id
`,
	query.TagLinkDelete: "DELETE FROM tag_link WHERE tag_id = ? AND item_id = ?",
	query.TagLinkGetByItem: `
SELECT
    t.id,
    COALESCE(t.parent, 0),
    t.name
FROM tag_link l
INNER JOIN tag t ON l.tag_id = t.id
WHERE l.item_id = ?
`,
	query.TagLinkGetByTag: `
SELECT
    i.id,
    i.feed_id,
    i.title,
    i.url,
    i.rating,
    i.timestamp,
    i.body
FROM tag_link l
INNER JOIN item i ON l.item_id = i.id
WHERE l.tag_id = ?
`,
	query.TagLinkGetMap: `
SELECT
    DISTINCT l.tag_id,
    t.name,
    COALESCE(t.parent, 0)
FROM tag_link l
INNER JOIN tag t ON l.tag_id = t.id
`,
}
