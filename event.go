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
	PreData   string
	PostData  string
}

func (e Event) RenderedCreatedAt() string {
	return e.CreatedAt.UTC().Format(time.UnixDate)
}
