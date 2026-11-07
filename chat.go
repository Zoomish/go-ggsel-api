package ggsel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Chat struct {
	InvoiceID int64  `json:"id_i"`
	Email     string `json:"email"`
	ProductID int64  `json:"product"`
	// LastMessageAt — метка времени последнего сообщения. В снимке это строка date_time, не текст реплики.
	LastMessageAt string `json:"last_message"`
	MessageCount  *int64 `json:"cnt_msg"`
	NewCount      *int64 `json:"cnt_new"`
}

type ChatPage struct {
	PageCount int64  `json:"cnt_pages"`
	Items     []Chat `json:"items"`
}

// ListChatsParams — фильтр чатов. Нулевые указатели в запрос не попадают.
type ListChatsParams struct {
	FilterNew *int64
	Email     *string
	GoodsID   *string
	PageSize  *int64
	Page      *int64
}

type Message struct {
	ID   int64  `json:"id"`
	Text string `json:"message"`
	// IsFromBuyer и IsFromSeller — флаги «сообщение от покупателя / продавца», не идентификаторы.
	IsFromBuyer  int64   `json:"buyer"`
	IsFromSeller int64   `json:"seller"`
	Deleted      *int64  `json:"deleted"`
	DateWritten  string  `json:"date_written"`
	DateSeen     *string `json:"date_seen"`
	IsFile       *int64  `json:"is_file"`
	Filename     *string `json:"filename"`
	URL          *string `json:"url"`
	IsImage      *int64  `json:"is_img"`
	Preview      *string `json:"preview"`
}

// ListMessagesParams — окно сообщений. id_i задаёт метод.
type ListMessagesParams struct {
	FromID *int64
	ToID   *int64
	Newer  *int64
	Count  *int64
}

func (c *Client) ListChats(ctx context.Context, params ListChatsParams) (ChatPage, error) {
	q := url.Values{}
	setInt(q, "filter_new", params.FilterNew)
	setString(q, "email", params.Email)
	setString(q, "id_ds", params.GoodsID)
	setInt(q, "pagesize", params.PageSize)
	setInt(q, "page", params.Page)
	var page ChatPage
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/debates/v2/chats",
		query:  q,
		dest:   &page,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return ChatPage{}, err
	}
	if page.Items == nil {
		page.Items = []Chat{}
	}
	return page, nil
}

// ListMessages возвращает сообщения заказа. Успешный ответ — JSON-массив.
func (c *Client) ListMessages(ctx context.Context, invoiceID int64, params ListMessagesParams) ([]Message, error) {
	if invoiceID <= 0 {
		return nil, fmt.Errorf("ggsel: list messages: id must be positive")
	}
	q := url.Values{}
	q.Set("id_i", strconv.FormatInt(invoiceID, 10))
	setInt(q, "id_from", params.FromID)
	setInt(q, "id_to", params.ToID)
	setInt(q, "newer", params.Newer)
	setInt(q, "count", params.Count)
	var items []Message
	if err := c.do(ctx, call{
		method: http.MethodGet,
		path:   "/api_sellers/api/debates/v2",
		query:  q,
		dest:   &items,
		auth:   authV1,
		retry:  true,
	}); err != nil {
		return nil, err
	}
	if items == nil {
		items = []Message{}
	}
	return items, nil
}

// SendMessage отправляет текст в чат заказа. Пустой текст в сеть не уходит.
func (c *Client) SendMessage(ctx context.Context, invoiceID int64, text string) error {
	if invoiceID <= 0 {
		return fmt.Errorf("ggsel: send message: id must be positive")
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("ggsel: send message: text is required")
	}
	q := url.Values{}
	q.Set("id_i", strconv.FormatInt(invoiceID, 10))
	return c.do(ctx, call{
		method: http.MethodPost,
		path:   "/api_sellers/api/debates/v2",
		query:  q,
		body: struct {
			Message string `json:"message"`
		}{Message: text},
		auth: authV1,
	})
}
