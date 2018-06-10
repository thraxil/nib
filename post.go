package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/russross/blackfriday"
)

type Post struct {
	Slug       string    `json:"slug"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	ModifiedAt time.Time `json:"modified_at"`
	Body       string    `datastore:"Body,noindex"`
	Author     string    `json:"author"`
}

func (p Post) URL() string {
	return "/post/" + p.Slug + "/"
}

func makeLink(s string) string {
	// s should look like '[[Page Title]]'
	// or [[Page Title|link text]]
	// we turn those into
	// [Page Title](/post/page-title/)
	// or
	// [link text](/post/page-title/)
	// respectively
	s = strings.Trim(s, "[]- ") // get rid of the delimiters
	title := s
	link := "/post/" + slugify(s) + "/"
	if strings.Index(s, "|") != -1 {
		parts := strings.SplitN(s, "|", 2)
		page_title := strings.Trim(parts[0], " ")
		link_text := strings.Trim(parts[1], " ")
		title = link_text
		link = "/post/" + slugify(page_title) + "/"
	}
	return "[" + title + "](" + link + ")"
}

func (p Post) LinkText() string {
	pattern, _ := regexp.Compile(`(\[\[\s*[^\|\]]+\s*\|?\s*[^\]]*\s*\]\])`)
	return pattern.ReplaceAllStringFunc(p.Body, makeLink)
}

func (p Post) RenderedBody() template.HTML {
	return template.HTML(string(blackfriday.MarkdownCommon([]byte(p.LinkText()))))
}

func truncateString(s string, size int) string {
	ns := s
	if len(s) > size {
		if size > 3 {
			size -= 3
		}
		ns = s[0:size] + "..."
	}
	return ns
}

func (p Post) RenderedPreview() template.HTML {
	return template.HTML(truncateString(string(blackfriday.MarkdownCommon([]byte(p.LinkText()))), 512))
}

func (p Post) AsJSON() string {
	b, _ := json.Marshal(p)
	return string(b)
}

func (p Post) RenderedCreatedAt() string {
	return p.CreatedAt.UTC().Format(time.UnixDate)
}

func (p Post) RenderedModifiedAt() string {
	return p.ModifiedAt.UTC().Format(time.UnixDate)
}

func AddComment(body, comment, author string, now time.Time) (string, error) {
	var commentTemplate = template.Must(template.ParseFiles("templates/comment.html"))

	tc := make(map[string]interface{})
	tc["comment_body"] = comment
	tc["author"] = author
	tc["timestamp"] = now.UTC().Format(time.UnixDate)

	var b bytes.Buffer
	if err := commentTemplate.Execute(&b, tc); err != nil {
		return body, err
	}

	commentsBody := b.String()
	body = body + commentsBody + "\n"
	return body, nil
}
