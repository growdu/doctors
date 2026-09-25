package clients

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// RefundClient 调 refund-service 内部接口。
//
// baseURL 例如 "http://127.0.0.1:8086"。
type RefundClient struct {
	baseURL string
	timeout time.Duration
}

// NewRefundClient 构造。
func NewRefundClient(baseURL string, timeout time.Duration) *RefundClient {
	return &RefundClient{baseURL: baseURL, timeout: timeout}
}

// List 调 GET /internal/refunds（管理员视角）。
func (c *RefundClient) List(ctx context.Context, status string, page, pageSize int) ([]map[string]any, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	if page > 0 {
		q.Set("page", fmtInt(int64(page)))
	}
	if pageSize > 0 {
		q.Set("page_size", fmtInt(int64(pageSize)))
	}
	var out []map[string]any
	err := doJSON(ctx, http.MethodGet, c.baseURL+"/internal/refunds?"+q.Encode(), nil, &out, c.timeout)
	return out, err
}

// Approve 调 POST /internal/refunds/:id/approve。
func (c *RefundClient) Approve(ctx context.Context, id, adminID int64, note string) error {
	body := map[string]any{"admin_id": adminID, "note": note}
	return doJSON(ctx, http.MethodPost,
		c.baseURL+"/internal/refunds/"+fmtInt(id)+"/approve", body, nil, c.timeout)
}

// Reject 调 POST /internal/refunds/:id/reject。
func (c *RefundClient) Reject(ctx context.Context, id, adminID int64, note string) error {
	body := map[string]any{"admin_id": adminID, "note": note}
	return doJSON(ctx, http.MethodPost,
		c.baseURL+"/internal/refunds/"+fmtInt(id)+"/reject", body, nil, c.timeout)
}