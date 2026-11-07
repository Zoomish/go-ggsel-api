package ggsel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

var errRequestFailed = errors.New("request failed")

type authMode int

const (
	authNone authMode = iota
	authV2
	authV1
)

type call struct {
	method    string
	path      string
	query     url.Values
	header    http.Header
	body      any
	dest      any
	auth      authMode
	retry     bool
	errorPath string
}

func (c *Client) do(ctx context.Context, call call) error {
	if ctx == nil {
		return fmt.Errorf("ggsel: nil context")
	}
	if call.method == "" {
		return fmt.Errorf("ggsel: method is required")
	}
	if call.path == "" || !strings.HasPrefix(call.path, "/") || strings.Contains(call.path, "?") {
		return fmt.Errorf("ggsel: path must be absolute and must not include a query")
	}
	var token string
	switch call.auth {
	case authNone, authV2:
	case authV1:
		var err error
		token, err = c.v1Token(ctx)
		if err != nil {
			return err
		}
		call.query = withToken(call.query, token)
	default:
		return fmt.Errorf("ggsel: unknown auth mode %d", call.auth)
	}
	rawURL, err := c.buildURL(call.path, call.query)
	if err != nil {
		return err
	}
	var body io.Reader
	if hasBody(call.body) {
		buf, err := json.Marshal(call.body)
		if err != nil {
			return fmt.Errorf("ggsel: encode body: %w", err)
		}
		body = bytes.NewReader(buf)
	}
	reqCtx := ctx
	if call.retry && call.method == http.MethodGet {
		reqCtx = withSafeRetry(ctx)
	}
	req, err := http.NewRequestWithContext(reqCtx, call.method, rawURL, body)
	if err != nil {
		return fmt.Errorf("ggsel: build request: %w", err)
	}
	for key, values := range call.header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Accept", "application/json")
	switch call.auth {
	case authNone, authV1:
		req.Header.Del("Authorization")
	case authV2:
		req.Header.Set("Authorization", c.apiKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return c.failTransport(call.method, call.publicPath(), err)
	}
	defer resp.Body.Close()
	payload, err := readLimited(resp.Body, maxResponseBody)
	if err != nil {
		return c.failTransport(call.method, call.publicPath(), err)
	}
	if len(payload) > maxResponseBody {
		return fmt.Errorf("ggsel: %s %s: response exceeds %d bytes", call.method, call.publicPath(), maxResponseBody)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
		if call.auth == authV1 && resp.StatusCode == http.StatusUnauthorized {
			c.invalidateV1(token)
		}
		apiErr := parseAPIError(call.method, call.publicPath(), resp.StatusCode, payload)
		return fmt.Errorf("ggsel: %w", apiErr)
	}
	if call.auth == authV1 {
		if err := v1ResultError(call, payload); err != nil {
			return err
		}
	}
	if call.dest == nil {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	if err := dec.Decode(call.dest); err != nil {
		return fmt.Errorf("ggsel: %s %s: decode: %w", call.method, call.publicPath(), err)
	}
	return nil
}

func (c call) publicPath() string {
	if c.errorPath != "" {
		return c.errorPath
	}
	return c.path
}

func withToken(query url.Values, token string) url.Values {
	q := make(url.Values, len(query)+1)
	for key, values := range query {
		if strings.EqualFold(key, "token") {
			continue
		}
		q[key] = append([]string(nil), values...)
	}
	q.Set("token", token)
	return q
}

func v1ResultError(call call, payload []byte) error {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || trimmed[0] == '[' {
		return nil
	}
	if trimmed[0] != '{' {
		return nil
	}
	var wire struct {
		RetVal *int `json:"retval"`
	}
	if err := json.Unmarshal(trimmed, &wire); err != nil || wire.RetVal == nil || *wire.RetVal == 0 {
		return nil
	}
	return fmt.Errorf("ggsel: %w", parseAPIError(call.method, call.publicPath(), http.StatusOK, payload))
}

func (c *Client) buildURL(path string, query url.Values) (string, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", fmt.Errorf("ggsel: build url: %w", err)
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	return u.String(), nil
}

func (c *Client) failTransport(method, path string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("ggsel: %s %s: %w", method, path, context.Canceled)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("ggsel: %s %s: %w", method, path, context.DeadlineExceeded)
	}
	cause := redactTransport(c.apiKey, err)
	if cause == "" {
		return fmt.Errorf("ggsel: %s %s: %w", method, path, errRequestFailed)
	}
	return fmt.Errorf("ggsel: %s %s: %w: %s", method, path, errRequestFailed, cause)
}

func redactTransport(apiKey string, err error) string {
	text := err.Error()
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		text = urlErr.Err.Error()
	}
	text = cutURLs(text)
	if apiKey != "" {
		text = strings.ReplaceAll(text, apiKey, "")
	}
	return strings.Join(strings.Fields(text), " ")
}

func cutURLs(s string) string {
	for _, scheme := range []string{"https://", "http://"} {
		for {
			i := strings.Index(s, scheme)
			if i < 0 {
				break
			}
			j := i + len(scheme)
			for j < len(s) && s[j] != ' ' && s[j] != '"' && s[j] != '\'' {
				j++
			}
			s = s[:i] + s[j:]
		}
	}
	return s
}

func readLimited(r io.Reader, limit int) ([]byte, error) {
	if limit < 0 {
		limit = 0
	}
	buf, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func hasBody(v any) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func, reflect.Chan:
		return !rv.IsNil()
	default:
		return true
	}
}
