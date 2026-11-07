package main

import (
	"strings"
	"testing"
)

func TestJSUnescape(t *testing.T) {
	got := jsUnescape(`line\nquote\'slash\/\u0041`)
	if got != "line\nquote'slash/A" {
		t.Fatalf("%q", got)
	}
	if got := jsUnescape(`\uD83D\uDE00`); got != "😀" {
		t.Fatalf("pair = %q", got)
	}
	if got := jsUnescape(`\uD83D`); got != "\uFFFD" {
		t.Fatalf("lone = %q", got)
	}
}

func TestUnifiedInsertionKeepsTail(t *testing.T) {
	got := unified("doc", []byte("a\nb\nc"), []byte("a\nX\nb\nc"))
	if strings.Contains(got, "-b") || strings.Contains(got, "-c") || !strings.Contains(got, "+X") {
		t.Fatalf("%s", got)
	}
	gone := unified("doc", []byte("a\nX\nb"), []byte("a\nb"))
	if strings.Contains(gone, "-b") || !strings.Contains(gone, "-X") || strings.Contains(gone, "+b") {
		t.Fatalf("%s", gone)
	}
}
