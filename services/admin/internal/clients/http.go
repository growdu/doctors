// Package clients 是 admin-service 调其他服务内部接口的 HTTP client 集合。
//
// 设计要点：
//   - 每个 client 一个上游服务（order / refund / escort / user）。
//   - 调用失败统一翻译为 ErrUpstreamUnavailable，handler 层再映射到 15003。
//   - 上游业务码（如 退款已处理 12002）透传为 errs.Error；handler 层照常翻译。
//   - 不引入任何项目内 import（除 shared/errs），保持 client 纯净。
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/growdu/doctors/shared/errs"
)

// ErrUpstreamUnavailable 上游服务不可用（DNS / connect / 5xx / bad envelope）。
var ErrUpstreamUnavailable = errors.New("clients: upstream unavailable")

// doJSON 通用 HTTP 调用 + JSON 响应解析。
//   - method / url / body：请求；body=nil 表示 GET。
//   - out：响应 data 反序列化目标；nil 表示丢弃。
//   - timeout：HTTP 客户端超时。
//
// 返回：
//   - 上游 HTTP 非 200 / connect 失败 → ErrUpstreamUnavailable（含原因）。
//   - 上游 body.code != 0 → errs.Error（含上游业务码）。
//   - 上游 body.code == 0 → nil，并把 env.Data unmarshal 到 out。
func doJSON(ctx context.Context, method, url string, body, out any, timeout time.Duration) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status=%d body=%s", ErrUpstreamUnavailable, resp.StatusCode, string(respBody))
	}

	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &env); err != nil {
		return fmt.Errorf("%w: bad envelope: %s", ErrUpstreamUnavailable, string(respBody))
	}

	if env.Code != 0 {
		// 上游业务错误，透传给上游错误码（handler 层统一翻译）。
		return errs.New(errs.Code(env.Code), env.Message)
	}
	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("unmarshal data: %w", err)
		}
	}
	return nil
}