package ggsel

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestParseAPIError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
		check  func(t *testing.T, err *APIError)
	}{
		{
			name:   "v1 retval",
			status: http.StatusBadRequest,
			body:   `{"retval":7,"retdesc":"bad input"}`,
			check: func(t *testing.T, err *APIError) {
				if err.RetVal == nil || *err.RetVal != 7 || err.RetDesc != "bad input" {
					t.Fatalf("parsed = %+v", err)
				}
				if !strings.Contains(err.Error(), "retval 7") || !strings.Contains(err.Error(), "bad input") {
					t.Fatalf("text = %s", err.Error())
				}
			},
		},
		{
			name:   "apilogin desc",
			status: http.StatusOK,
			body:   `{"retval":-1,"desc":"sign mismatch"}`,
			check: func(t *testing.T, err *APIError) {
				if err.Desc != "sign mismatch" || err.RetDesc != "" {
					t.Fatalf("parsed = %+v", err)
				}
				if !strings.Contains(err.Error(), "sign mismatch") {
					t.Fatalf("text = %s", err.Error())
				}
			},
		},
		{
			name:   "v2 entity",
			status: http.StatusUnprocessableEntity,
			body:   `{"errors":[{"code":"INVALID","entity":{"id":0,"resource":"offer","description":"price"}},{"code":"OTHER","entity":{"id":null,"description":"x"}}]}`,
			check: func(t *testing.T, err *APIError) {
				if len(err.Errors) != 2 || err.Errors[0].Code != "INVALID" || err.Errors[1].Code != "OTHER" {
					t.Fatalf("errors = %+v", err.Errors)
				}
				if err.Errors[0].Entity == nil || err.Errors[0].Entity.ID == nil || *err.Errors[0].Entity.ID != 0 {
					t.Fatalf("entity = %+v", err.Errors[0].Entity)
				}
				if err.Errors[0].Entity.Resource != "offer" || err.Errors[0].Entity.Description != "price" {
					t.Fatalf("entity = %+v", err.Errors[0].Entity)
				}
				if err.Errors[1].Entity == nil || err.Errors[1].Entity.ID != nil {
					t.Fatalf("second entity = %+v", err.Errors[1].Entity)
				}
				if !strings.Contains(err.Error(), "codes INVALID,OTHER") {
					t.Fatalf("text = %s", err.Error())
				}
			},
		},
		{
			name:   "not json",
			status: http.StatusBadGateway,
			body:   `not-json`,
			check: func(t *testing.T, err *APIError) {
				if err.RetVal != nil || err.RetDesc != "" || len(err.Errors) != 0 {
					t.Fatalf("parsed = %+v", err)
				}
				if strings.Contains(err.Error(), "not-json") {
					t.Fatalf("text includes body: %s", err.Error())
				}
			},
		},
		{
			name:   "empty",
			status: http.StatusUnauthorized,
			body:   ``,
			check: func(t *testing.T, err *APIError) {
				if len(err.Body) != 0 || !IsUnauthorized(err) {
					t.Fatalf("parsed = %+v", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseAPIError(http.MethodGet, "/template", tt.status, []byte(tt.body))
			if got.Method != http.MethodGet || got.Path != "/template" || got.StatusCode != tt.status {
				t.Fatalf("meta = %+v", got)
			}
			tt.check(t, got)
		})
	}
}

func TestAPIErrorBodyIsTruncated(t *testing.T) {
	t.Parallel()
	body := bytesRepeat('z', maxErrorBody+25)
	got := parseAPIError(http.MethodGet, "/x", http.StatusInternalServerError, body)
	if len(got.Body) != maxErrorBody {
		t.Fatalf("len = %d", len(got.Body))
	}
	if strings.Contains(got.Error(), "z") {
		t.Fatal("error text includes body")
	}
}

func TestIsUnauthorized(t *testing.T) {
	t.Parallel()
	apiErr := parseAPIError(http.MethodGet, "/x", http.StatusUnauthorized, nil)
	wrapped := fmt.Errorf("ggsel: %w", apiErr)
	if !IsUnauthorized(wrapped) {
		t.Fatal("wrapped 401 was not unauthorized")
	}
	if IsUnauthorized(fmt.Errorf("ggsel: %w", parseAPIError(http.MethodGet, "/x", http.StatusForbidden, nil))) {
		t.Fatal("403 is unauthorized")
	}
	if IsUnauthorized(errors.New("nope")) || IsUnauthorized(nil) {
		t.Fatal("plain error is unauthorized")
	}
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
