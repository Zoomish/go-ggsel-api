package ggsel

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// APIError — ответ Seller API, который не разобран как успех.
// Error не содержит тело, ключ и token. Тело лежит в Body.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Body       []byte
	RetVal     *int
	RetDesc    string
	Desc       string
	Errors     []APIErrorItem
}

// APIErrorItem — элемент errors[] контура v2.
type APIErrorItem struct {
	Code   string
	Entity *APIErrorEntity
}

// APIErrorEntity — объект entity у ошибки v2.
type APIErrorEntity struct {
	ID          *int64
	Resource    string
	Description string
}

func (e *APIError) Error() string {
	if e == nil {
		return "ggsel: <nil>"
	}
	var b strings.Builder
	if e.Method != "" || e.Path != "" {
		b.WriteString(e.Method)
		if e.Method != "" && e.Path != "" {
			b.WriteByte(' ')
		}
		b.WriteString(e.Path)
	}
	if e.StatusCode != 0 {
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		b.WriteString("status ")
		b.WriteString(strconv.Itoa(e.StatusCode))
	}
	if e.RetVal != nil {
		b.WriteString(" retval ")
		b.WriteString(strconv.Itoa(*e.RetVal))
	}
	switch {
	case e.RetDesc != "":
		b.WriteString(": ")
		b.WriteString(e.RetDesc)
	case e.Desc != "":
		b.WriteString(": ")
		b.WriteString(e.Desc)
	}
	if len(e.Errors) > 0 {
		b.WriteString(" codes ")
		for i, item := range e.Errors {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(item.Code)
		}
	}
	if b.Len() == 0 {
		return "ggsel: api error"
	}
	return b.String()
}

// IsUnauthorized сообщает, что API ответил HTTP 401.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr != nil && apiErr.StatusCode == http.StatusUnauthorized
}

func parseAPIError(method, path string, status int, body []byte) *APIError {
	out := &APIError{
		StatusCode: status,
		Method:     method,
		Path:       path,
		Body:       truncate(body),
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return out
	}
	var wire wireError
	if err := json.Unmarshal(trimmed, &wire); err != nil {
		return out
	}
	out.RetVal = wire.RetVal
	out.RetDesc = wire.RetDesc
	out.Desc = wire.Desc
	if len(wire.Errors) == 0 {
		return out
	}
	out.Errors = make([]APIErrorItem, 0, len(wire.Errors))
	for _, item := range wire.Errors {
		next := APIErrorItem{Code: item.Code}
		if item.Entity != nil {
			next.Entity = &APIErrorEntity{
				ID:          item.Entity.ID,
				Resource:    item.Entity.Resource,
				Description: item.Entity.Description,
			}
		}
		out.Errors = append(out.Errors, next)
	}
	return out
}

type wireError struct {
	RetVal  *int   `json:"retval"`
	RetDesc string `json:"retdesc"`
	Desc    string `json:"desc"`
	Errors  []struct {
		Code   string `json:"code"`
		Entity *struct {
			ID          *int64 `json:"id"`
			Resource    string `json:"resource"`
			Description string `json:"description"`
		} `json:"entity"`
	} `json:"errors"`
}

func truncate(b []byte) []byte {
	if len(b) <= maxErrorBody {
		return append([]byte(nil), b...)
	}
	return append([]byte(nil), b[:maxErrorBody]...)
}
