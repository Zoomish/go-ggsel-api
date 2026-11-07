package main

import "testing"

func TestJSUnescape(t *testing.T) {
	got := jsUnescape(`line\nquote\'slash\/\u0041`)
	if got != "line\nquote'slash/A" {
		t.Fatalf("%q", got)
	}
}
