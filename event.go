package main

import (
	"encoding/json"
	"time"

	"google.golang.org/appengine/datastore"
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

func (e Event) PrePost() *Post {
	if e.PreData == "" {
		return nil
	}
	p := &Post{}
	err := json.Unmarshal([]byte(e.PreData), p)
	if err != nil {
		return nil
	}
	return p
}

func (e Event) PostPost() *Post {
	if e.PostData == "" {
		return nil
	}
	p := &Post{}
	err := json.Unmarshal([]byte(e.PostData), p)
	if err != nil {
		return nil
	}
	return p
}
