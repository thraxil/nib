package main

import (
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	"appengine"
	"appengine/user"
)

var indexTemplate = template.Must(template.ParseFiles("templates/base.html", "templates/index.html"))

func index(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	rep := NewRepository(ctx)

	posts, err := rep.RecentPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tc := make(map[string]interface{})
	tc["recent_posts"] = posts
	if err := indexTemplate.Execute(w, tc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var searchTemplate = template.Must(template.ParseFiles("templates/base.html", "templates/search.html"))

func searchResults(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	rep := NewRepository(ctx)
	q := r.FormValue("q")
	posts, err := rep.SearchPosts(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tc := make(map[string]interface{})
	tc["q"] = q
	tc["posts"] = posts
	if err := searchTemplate.Execute(w, tc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var newPostTemplate = template.Must(template.ParseFiles("templates/base.html", "templates/newPost.html"))

func newPost(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		ctx := appengine.NewContext(r)
		author := user.Current(ctx)
		title := generateTitle(r.FormValue("title"), author)
		slug := generateSlug(r.FormValue("slug"), title, ctx)
		body := r.FormValue("body")

		rep := NewRepository(ctx)
		post, err := rep.NewPost(title, slug, body, author)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, post.URL(), http.StatusFound)
	} else {
		tc := make(map[string]interface{})
		if err := newPostTemplate.Execute(w, tc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func username(u *user.User) string {
	parts := strings.Split(u.String(), "@")
	return parts[0]
}

func generateTitle(title string, author *user.User) string {
	if title == "" {
		return username(author) + ": " + time.Now().UTC().Format("2006-01-02 15:04:05 MST")
	}
	return title
}

func slugify(s string) string {
	pattern, _ := regexp.Compile("\\W+")
	s = pattern.ReplaceAllString(s, "-")
	//	s = strings.Trim(s, " \t\n\r-")
	//	s = strings.Replace(s, " ", "-", -1)
	s = strings.ToLower(s)
	return s
}

func generateSlug(slug, title string, ctx appengine.Context) string {
	if slug == "" {
		slug = title
	}
	slug = slugify(slug)
	// TODO: ensure slug is unique
	return slug
}

var postTemplate = template.Must(template.ParseFiles("templates/base.html", "templates/post.html"))

func post(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "bad request", 404)
		return
	}
	slug := parts[2]

	rep := NewRepository(ctx)
	post, key, err := rep.PostFromSlug(slug)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if post == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	events, err := rep.PostEvents(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(parts) == 4 {
		tc := make(map[string]interface{})
		tc["post"] = post
		tc["events"] = events

		if err := postTemplate.Execute(w, tc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if len(parts) == 5 && parts[3] == "delete" {
		if r.Method == "POST" {
			err = rep.DeletePost(post, key)

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		var confirmTemplate = template.Must(template.ParseFiles("templates/confirm.html"))
		tc := make(map[string]interface{})
		if err := confirmTemplate.Execute(w, tc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	if len(parts) == 5 && parts[3] == "edit" && r.Method == "POST" {
		title := r.FormValue("title")
		body := r.FormValue("body")
		err = rep.EditPost(post, key, title, body)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, post.URL(), http.StatusFound)
		return
	}
	http.Error(w, "Bad Request", http.StatusInternalServerError)
}

var allTemplate = template.Must(template.ParseFiles("templates/base.html", "templates/all.html"))

func all(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	rep := NewRepository(ctx)

	posts, err := rep.AllPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tc := make(map[string]interface{})
	tc["posts"] = posts
	if err := allTemplate.Execute(w, tc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
