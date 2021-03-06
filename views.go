package main

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"google.golang.org/appengine/user"
)

var indexTemplate = template.Must(template.ParseFiles("templates/base.html", "templates/index.html"))

func index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rep := NewRepository(ctx)

	spage := r.FormValue("page")
	page, err := strconv.Atoi(spage)
	if err != nil {
		page = 0
	}
	limit := 20
	offset := limit * page

	posts, cnt, err := rep.RecentPosts(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	next_page := fmt.Sprintf("%d", (offset+limit)/limit)
	if offset+limit > cnt {
		next_page = ""
	}
	prev_page := ""
	if page > 0 {
		prev_page = fmt.Sprintf("%d", page-1)
	}
	tc := make(map[string]interface{})
	tc["recent_posts"] = posts
	tc["next_page"] = next_page
	tc["prev_page"] = prev_page
	tc["page"] = fmt.Sprintf("%d", page)
	tc["cnt"] = fmt.Sprintf("%d", cnt)

	if err := indexTemplate.Execute(w, tc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var searchTemplate = template.Must(template.ParseFiles("templates/base.html", "templates/search.html"))

func searchResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
		ctx := r.Context()
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

func generateSlug(slug, title string, ctx context.Context) string {
	if slug == "" {
		slug = title
	}
	slug = slugify(slug)
	// TODO: ensure slug is unique
	return slug
}

func unSlug(slug string) string {
	// convert a slug to a likely title string
	slug = strings.Replace(slug, "-", " ", -1)
	return strings.Title(slug)
}

var post404Template = template.Must(template.ParseFiles("templates/base.html", "templates/post404.html"))

func post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
		// give them a form to add a post with that slug
		tc := make(map[string]interface{})
		tc["slug"] = slug
		tc["title"] = unSlug(slug)
		if err := post404Template.Execute(w, tc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

		funcMap := template.FuncMap{
			"htmlescaper": html.EscapeString,
		}

		postTemplate, err := template.New("base.html").Funcs(funcMap).ParseFiles("templates/base.html", "templates/post.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		if err = postTemplate.Execute(w, tc); err != nil {
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
	if len(parts) == 5 && parts[3] == "append" && r.Method == "POST" {
		append_body := r.FormValue("body")
		now := time.Now()

		body := post.Body + "\n\n<hr />\n" + "<p class=\"byline\">" + now.UTC().Format(time.UnixDate) + "</p>\n\n" + append_body
		err = rep.EditPost(post, key, post.Title, body)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, post.URL(), http.StatusFound)
		return
	}
	if len(parts) == 5 && parts[3] == "comment" && r.Method == "POST" {
		comment_body := r.FormValue("body")
		now := time.Now()
		author := user.Current(ctx)

		body := post.Body + "\n\n<hr class='comment'>\n\n" + comment_body + "\n\n<p class=\"byline\"><span class=\"comment-author\">" +
			username(author) + "</span> - " + now.UTC().Format(time.UnixDate) + "</p>\n\n"

		err = rep.EditPost(post, key, post.Title, body)

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
	ctx := r.Context()
	rep := NewRepository(ctx)

	spage := r.FormValue("page")
	page, err := strconv.Atoi(spage)
	if err != nil {
		page = 0
	}
	limit := 50
	offset := limit * page

	posts, cnt, err := rep.AllPosts(limit, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	next_page := fmt.Sprintf("%d", (offset+limit)/limit)
	if offset+limit > cnt {
		next_page = ""
	}
	prev_page := ""
	if page > 0 {
		prev_page = fmt.Sprintf("%d", page-1)
	}
	tc := make(map[string]interface{})
	tc["posts"] = posts
	tc["next_page"] = next_page
	tc["prev_page"] = prev_page
	tc["page"] = fmt.Sprintf("%d", page)
	tc["cnt"] = fmt.Sprintf("%d", cnt)
	if err := allTemplate.Execute(w, tc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var eventTemplate = template.Must(template.ParseFiles("templates/base.html", "templates/event.html"))

func event(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "bad request", 404)
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rep := NewRepository(ctx)
	event, err := rep.EventFromID(int64(id))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if event == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if len(parts) == 4 {
		tc := make(map[string]interface{})
		tc["event"] = event

		if err := eventTemplate.Execute(w, tc); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}
