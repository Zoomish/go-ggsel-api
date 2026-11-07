package ggsel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const v1TokenSkew = 30 * time.Second

// ErrSellerIDRequired возвращается, если метод v1 вызван без SellerID.
// Префикс ggsel: добавляет обёртка, не сам sentinel.
var ErrSellerIDRequired = errors.New("seller id is required")

// Login получает token v1.
// Кэш пустеет за 30 секунд до valid_thru.
// Если valid_thru не разбирается, token этого вызова возвращается, но в кэш не кладётся.
// Методы v1 вызывают вход сами, если кэш пуст.
func (c *Client) Login(ctx context.Context) error {
	_, err := c.v1Token(ctx)
	return err
}

func (c *Client) v1Token(ctx context.Context) (string, error) {
	if c.sellerID <= 0 {
		return "", fmt.Errorf("ggsel: %w", ErrSellerIDRequired)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if token, ok := c.cachedV1(); ok {
		return token, nil
	}
	ch := c.v1Group.DoChan("v1", func() (any, error) {
		if token, ok := c.cachedV1(); ok {
			return token, nil
		}
		loginCtx, cancel := c.detachLogin(ctx)
		defer cancel()
		return c.apilogin(loginCtx)
	})
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if res.Err != nil {
			return "", res.Err
		}
		token, _ := res.Val.(string)
		if token == "" {
			return "", fmt.Errorf("ggsel: apilogin: empty token")
		}
		return token, nil
	}
}

func (c *Client) cachedV1() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tokenV1 == "" || c.v1ValidThru.IsZero() {
		return "", false
	}
	if !c.now().Add(v1TokenSkew).Before(c.v1ValidThru) {
		return "", false
	}
	return c.tokenV1, true
}

func (c *Client) storeV1(token string, until time.Time) {
	if token == "" || until.IsZero() {
		return
	}
	c.mu.Lock()
	c.tokenV1 = token
	c.v1ValidThru = until
	c.mu.Unlock()
}

func (c *Client) invalidateV1(used string) {
	if used == "" {
		return
	}
	c.mu.Lock()
	if c.tokenV1 == used {
		c.tokenV1 = ""
		c.v1ValidThru = time.Time{}
	}
	c.mu.Unlock()
}

func (c *Client) apilogin(ctx context.Context) (string, error) {
	timestamp := strconv.FormatInt(c.now().Unix(), 10)
	body := struct {
		SellerID  int64  `json:"seller_id"`
		Timestamp string `json:"timestamp"`
		Sign      string `json:"sign"`
	}{
		SellerID:  c.sellerID,
		Timestamp: timestamp,
		Sign:      signV1(c.apiKey, timestamp),
	}
	var raw json.RawMessage
	if err := c.do(ctx, call{
		method: http.MethodPost,
		path:   "/api_sellers/api/apilogin",
		body:   body,
		dest:   &raw,
		auth:   authNone,
	}); err != nil {
		return "", err
	}
	var resp struct {
		RetVal    *int   `json:"retval"`
		Token     string `json:"token"`
		ValidThru string `json:"valid_thru"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("ggsel: apilogin: %w", err)
	}
	if resp.RetVal == nil || *resp.RetVal != 0 {
		return "", fmt.Errorf("ggsel: %w", parseAPIError(http.MethodPost, "/api_sellers/api/apilogin", http.StatusOK, raw))
	}
	if resp.Token == "" {
		return "", fmt.Errorf("ggsel: apilogin: empty token")
	}
	if until, ok := parseValidThru(resp.ValidThru); ok {
		c.storeV1(resp.Token, until)
	}
	return resp.Token, nil
}

func (c *Client) detachLogin(parent context.Context) (context.Context, context.CancelFunc) {
	base := context.WithoutCancel(parent)
	deadline, ok := parent.Deadline()
	timeout := defaultTimeout
	if c.http != nil && c.http.Timeout > 0 {
		timeout = c.http.Timeout
	}
	clientDeadline := time.Now().Add(timeout)
	if !ok || clientDeadline.Before(deadline) {
		deadline = clientDeadline
	}
	return context.WithDeadline(base, deadline)
}

func signV1(apiKey, timestamp string) string {
	sum := sha256.Sum256([]byte(apiKey + timestamp))
	return hex.EncodeToString(sum[:])
}

func parseValidThru(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	until, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, false
	}
	return until, true
}
