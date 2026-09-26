// Package health —— HTTP /readyz handler。
//
// ReadyzHandler 返回 gin.HandlerFunc：
//   - 调用 Manager.RunAll 收集所有 checker 结果
//   - 全部 OK → 200 + JSON（status=healthy）
//   - 任一失败 → 503 + JSON（status=unhealthy，含每个 checker 的 err 字段）
//
// HTTP 状态码语义：
//   - 200：K8s readinessProbe 通过 → 流量下发
//   - 503：readinessProbe 失败 → 摘流（kubectl rollout / service endpoint 摘除）
//
// 推荐挂载顺序：Metrics → Recovery → /readyz（不挂限流，避免摘流时被 429 误判）。
//
// 与 /healthz 的关系：
//   - /healthz：进程存活（liveness），不依赖外部资源。
//   - /readyz：依赖就绪（readiness），包含 DB / Redis / Kafka 健康。

package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ReadyzHandler 返回挂到 /readyz 的 gin.HandlerFunc。
//
// 设计要点：
//   - 使用 gin.HandlerFunc（项目统一 Gin）；如果后续需要 net/http 形态，
//     另加一个 ReadyzHTTPHandler(m) http.Handler；本版本仅 Gin。
//   - body 是 Report 的 JSON；运维 curl /readyz 即可拿到每项依赖明细。
func ReadyzHandler(m *Manager) gin.HandlerFunc {
	if m == nil {
		// nil manager → fail-closed：永远 503，避免"忘了装配就放行"。
		return func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, Report{
				Healthy: false,
				Checks: []Status{{
					Name:  "manager",
					OK:    false,
					Error: "health manager not initialized",
				}},
			})
		}
	}
	return func(c *gin.Context) {
		report := m.RunAll(c.Request.Context())
		status := http.StatusOK
		if !report.Healthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, report)
	}
}