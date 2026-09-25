package clients

import (
	"context"
	"net/http"
	"time"
)

// EscortClient 调 escort-service 内部接口。
//
// baseURL 例如 "http://127.0.0.1:8083"。
type EscortClient struct {
	baseURL string
	timeout time.Duration
}

// NewEscortClient 构造。
func NewEscortClient(baseURL string, timeout time.Duration) *EscortClient {
	return &EscortClient{baseURL: baseURL, timeout: timeout}
}

// ListPendingAudit 调 GET /internal/escorts/pending-audit。
func (c *EscortClient) ListPendingAudit(ctx context.Context) ([]map[string]any, error) {
	var out []map[string]any
	err := doJSON(ctx, http.MethodGet,
		c.baseURL+"/internal/escorts/pending-audit", nil, &out, c.timeout)
	return out, err
}

// Approve 调 POST /internal/escorts/:id/approve。
func (c *EscortClient) Approve(ctx context.Context, id, adminID int64, note string) error {
	body := map[string]any{"admin_id": adminID, "note": note}
	return doJSON(ctx, http.MethodPost,
		c.baseURL+"/internal/escorts/"+fmtInt(id)+"/approve", body, nil, c.timeout)
}

// Reject 调 POST /internal/escorts/:id/reject。
func (c *EscortClient) Reject(ctx context.Context, id, adminID int64, note string) error {
	body := map[string]any{"admin_id": adminID, "note": note}
	return doJSON(ctx, http.MethodPost,
		c.baseURL+"/internal/escorts/"+fmtInt(id)+"/reject", body, nil, c.timeout)
}