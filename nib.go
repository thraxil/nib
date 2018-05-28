package main

import (
	"net/http"

	"google.golang.org/appengine"
)

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc("/all/", all)
	http.HandleFunc("/new/", newPost)
	http.HandleFunc("/post/", post)
	http.HandleFunc("/search/", searchResults)
	appengine.Main()
}
