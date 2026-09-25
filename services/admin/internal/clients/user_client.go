package clients

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// UserClient 调 user-service 内部接口。
//
// baseURL 例如 "http://127.0.0.1:8081"。
type UserClient struct {
	baseURL string
	timeout time.Duration
}

// NewUserClient 构造。
func NewUserClient(baseURL string, timeout time.Duration) *UserClient {
	return &UserClient{baseURL: baseURL, timeout: timeout}
}

// List 调 GET /internal/users?role=patient&keyword=xxx&page=1&page_size=20。
func (c *UserClient) List(ctx context.Context, role, keyword string, page, pageSize int) ([]map[string]any, error) {
	q := url.Values{}
	if role != "" {
		q.Set("role", role)
	}
	if keyword != "" {
		q.Set("keyword", keyword)
	}
	if page > 0 {
		q.Set("page", fmtInt(int64(page)))
	}
	if pageSize > 0 {
		q.Set("page_size", fmtInt(int64(pageSize)))
	}
	var out []map[string]any
	err := doJSON(ctx, http.MethodGet,
		c.baseURL+"/internal/users?"+q.Encode(), nil, &out, c.timeout)
	return out, err
}