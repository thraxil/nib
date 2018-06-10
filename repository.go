package main

import (
	"fmt"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/search"
	"appengine/user"
)

type Repository struct {
	ctx appengine.Context
}

func NewRepository(ctx appengine.Context) *Repository {
	return &Repository{ctx}
}

func (r *Repository) User() *user.User {
	return user.Current(r.ctx)
}

// Commands

func (r *Repository) NewPost(title, slug, body string, author *user.User) (*Post, error) {
	now := time.Now()
	post := &Post{
		Title:      title,
		Slug:       slug,
		Body:       body,
		Author:     author.String(),
		CreatedAt:  now,
		ModifiedAt: now,
	}

	key := datastore.NewIncompleteKey(r.ctx, "Post", nil)

	nkey, err := datastore.Put(r.ctx, key, post)
	if err != nil {
		r.ctx.Errorf("datastore put failed")
		return nil, err
	}
	ekey := datastore.NewIncompleteKey(r.ctx, "Event", nil)
	event := &Event{
		Action:    "CreatePost",
		Author:    author.String(),
		CreatedAt: now,
		Post:      nkey,
		PreData:   "",
		PostData:  post.AsJSON(),
	}
	_, err = datastore.Put(r.ctx, ekey, event)
	if err != nil {
		r.ctx.Errorf("datastore event put failed")
		return nil, err
	}

	index, err := search.Open("posts")
	if err != nil {
		r.ctx.Errorf("search open failed")
		return nil, err
	}
	_, err = index.Put(r.ctx, fmt.Sprintf("%d", nkey.IntID()), post)
	if err != nil {
		r.ctx.Errorf("search put failed")
		return nil, err
	}
	return post, nil
}

func (r *Repository) DeletePost(post *Post, key *datastore.Key) error {
	now := time.Now()
	ekey := datastore.NewIncompleteKey(r.ctx, "Event", nil)
	event := &Event{
		Action:    "DeletePost",
		Author:    r.User().String(),
		CreatedAt: now,
		Post:      key,
		PreData:   post.AsJSON(),
		PostData:  "",
	}
	_, err := datastore.Put(r.ctx, ekey, event)
	if err != nil {
		r.ctx.Errorf("datastore event put failed")
		return err
	}

	datastore.Delete(r.ctx, key)

	index, err := search.Open("posts")
	if err != nil {
		return err
	}

	err = index.Delete(r.ctx, fmt.Sprintf("%d", key.IntID()))
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) EditPost(post *Post, key *datastore.Key, title, body string) error {
	now := time.Now()
	ekey := datastore.NewIncompleteKey(r.ctx, "Event", nil)
	preData := post.AsJSON()
	post.Title = title
	post.Body = body
	post.ModifiedAt = now
	postData := post.AsJSON()
	event := &Event{
		Action:    "EditPost",
		Author:    r.User().String(),
		CreatedAt: now,
		Post:      key,
		PreData:   preData,
		PostData:  postData,
	}
	_, err := datastore.Put(r.ctx, ekey, event)
	if err != nil {
		r.ctx.Errorf("datastore event put failed")
		return err
	}

	_, err = datastore.Put(r.ctx, key, post)
	if err != nil {
		r.ctx.Errorf("datastore put failed")
		return err
	}

	index, err := search.Open("posts")
	if err != nil {
		r.ctx.Errorf("search open failed")
		return err
	}
	_, err = index.Put(r.ctx, fmt.Sprintf("%d", key.IntID()), post)
	if err != nil {
		r.ctx.Errorf("search put failed")
		return err
	}
	return nil
}

// Queries

func (r *Repository) PostFromSlug(slug string) (*Post, *datastore.Key, error) {
	var post Post
	q := datastore.NewQuery("Post").Filter("Slug =", slug)
	var posts []Post
	var key *datastore.Key
	keys, err := q.GetAll(r.ctx, &posts)
	if err != nil {
		return nil, nil, err
	}
	if len(posts) < 1 {
		return nil, nil, nil
	}
	post = posts[0]
	key = keys[0]

	return &post, key, nil
}

type KeyedEvent struct {
	Key   *datastore.Key
	Event Event
}

func (ke KeyedEvent) URL() string {
	return fmt.Sprintf("/event/%d/", ke.Key.IntID())
}

func (r *Repository) PostEvents(key *datastore.Key) ([]KeyedEvent, error) {
	q := datastore.NewQuery("Event").Filter("Post =", key).Order("-CreatedAt")
	events := make([]Event, 0, 1)
	kevents := make([]KeyedEvent, 0, 1)

	keys, err := q.GetAll(r.ctx, &events)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(events); i++ {
		kevents = append(kevents, KeyedEvent{keys[i], events[i]})
	}
	return kevents, nil
}

func (r *Repository) SearchPosts(q string) ([]Post, error) {
	index, err := search.Open("posts")
	if err != nil {
		return nil, err
	}
	posts := make([]Post, 0, 1)

	for t := index.Search(r.ctx, q, nil); ; {
		var post Post
		_, err := t.Next(&post)
		if err == search.Done {
			break
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (r *Repository) RecentPosts() ([]Post, error) {
	q := datastore.NewQuery("Post").Order("-CreatedAt").Limit(20)
	posts := make([]Post, 0, 1)
	if _, err := q.GetAll(r.ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *Repository) AllPosts() ([]Post, error) {
	q := datastore.NewQuery("Post").Order("Title")
	posts := make([]Post, 0, 1)
	if _, err := q.GetAll(r.ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *Repository) EventFromID(id int64) (*Event, error) {
	var event Event
	key := datastore.NewKey(r.ctx, "Event", "", id, nil)
	err := datastore.Get(r.ctx, key, &event)
	return &event, err
}
