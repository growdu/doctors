package clients

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// ListOrdersParams 是 order ListAll 的参数。
type ListOrdersParams struct {
	Status     string
	HospitalID int64
	PatientID  int64
	Page       int
	PageSize   int
}

// OrderClient 调 order-service 内部接口。
//
// baseURL 例如 "http://127.0.0.1:8082"。
type OrderClient struct {
	baseURL string
	timeout time.Duration
}

// NewOrderClient 构造。
func NewOrderClient(baseURL string, timeout time.Duration) *OrderClient {
	return &OrderClient{baseURL: baseURL, timeout: timeout}
}

// ListAll 调 GET /internal/orders（全量 + 筛选）。
func (c *OrderClient) ListAll(ctx context.Context, p ListOrdersParams) ([]map[string]any, error) {
	q := url.Values{}
	if p.Status != "" {
		q.Set("status", p.Status)
	}
	if p.HospitalID > 0 {
		q.Set("hospital_id", fmtInt(p.HospitalID))
	}
	if p.PatientID > 0 {
		q.Set("patient_id", fmtInt(p.PatientID))
	}
	if p.Page > 0 {
		q.Set("page", fmtInt(int64(p.Page)))
	}
	if p.PageSize > 0 {
		q.Set("page_size", fmtInt(int64(p.PageSize)))
	}
	var out []map[string]any
	err := doJSON(ctx, http.MethodGet, c.baseURL+"/internal/orders?"+q.Encode(), nil, &out, c.timeout)
	return out, err
}

// Get 调 GET /internal/orders/:id。
func (c *OrderClient) Get(ctx context.Context, id int64) (map[string]any, error) {
	var out map[string]any
	err := doJSON(ctx, http.MethodGet, c.baseURL+"/internal/orders/"+fmtInt(id), nil, &out, c.timeout)
	return out, err
}

// ForceCancel 调 POST /internal/orders/:id/force-cancel。
func (c *OrderClient) ForceCancel(ctx context.Context, orderID, adminID int64, reason string) error {
	body := map[string]any{"admin_id": adminID, "reason": reason}
	return doJSON(ctx, http.MethodPost,
		c.baseURL+"/internal/orders/"+fmtInt(orderID)+"/force-cancel",
		body, nil, c.timeout)
}