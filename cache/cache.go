/*
Copyright © 2022 Chris Cammack <chris@ccammack.com>

*/

package cache

import (
	"encoding/json"
	"net/http"
)

func init() {
}

// TODO: golang serve file from memory
// TODO: golang lru cache
// https://www.alexedwards.net/blog/golang-response-snippets

func Page(r *http.Request) []byte {
	// TODO: return current page here
	body, _ := json.Marshal(map[string]string{
		"page": "<page goes here>",
	})
	return body
}

func Update(r *http.Request) []byte {
	body, _ := json.Marshal(map[string]string{
		"state": "updated",
	})
	return body
}

func Status(r *http.Request) []byte {
	body, _ := json.Marshal(map[string]string{
		"file": "<file goes here>",
	})
	return body
}
