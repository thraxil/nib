package main

import (
	"time"

	"appengine/datastore"
)

type Event struct {
	Action    string
	Author    string
	CreatedAt time.Time
	Post      *datastore.Key
	PreData   string `datastore:"PreData,noindex"`
	PostData  string `datastore:"PostData,noindex"`
}

func (e Event) RenderedCreatedAt() string {
	return e.CreatedAt.UTC().Format(time.UnixDate)
}
