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
//
// 指标埋点：
//   - 每次 /readyz 调用入口 → ReadyzCheckTotal{check="_request",status="request"} +1
//     （用于 QPS 类告警 / 容量规划）。
//   - 每个 checker 调用后按结果 Inc：status="ok" 或 "fail"。
//     （用于 ReadyzCheckFailure 告警：持续 fail 1m 即触发 critical）。
//
// 失败注入：仅依赖失败计入 status="fail"；handler 自身 panic 由 recovery middleware
// 接管（与 /healthz 一致），不进本计数器。

package health

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/growdu/doctors/shared/metrics"
)

// ReadyzHandler 返回挂到 /readyz 的 gin.HandlerFunc。
//
// 设计要点：
//   - 使用 gin.HandlerFunc（项目统一 Gin）；如果后续需要 net/http 形态，
//     另加一个 ReadyzHTTPHandler(m) http.Handler；本版本仅 Gin。
//   - body 是 Report 的 JSON；运维 curl /readyz 即可拿到每项依赖明细。
//   - nil manager → fail-closed：永远 503，避免"忘了装配就放行"；
//     此时 ReadyzCheckTotal{check="_manager",status="fail"} +1，便于告警识别。
func ReadyzHandler(m *Manager) gin.HandlerFunc {
	if m == nil {
		// nil manager → fail-closed：永远 503 + 计数器 Inc。
		return func(c *gin.Context) {
			metrics.ReadyzCheckTotal.WithLabelValues("_request", "request").Inc()
			metrics.ReadyzCheckTotal.WithLabelValues("_manager", "fail").Inc()
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
		// 请求级埋点：每次 /readyz 调用都 +1，与 checker 数无关，便于 QPS 监控。
		metrics.ReadyzCheckTotal.WithLabelValues("_request", "request").Inc()

		report := m.RunAll(c.Request.Context())
		// 每个 checker 按结果 Inc（ok / fail）。
		for _, s := range report.Checks {
			if s.OK {
				metrics.ReadyzCheckTotal.WithLabelValues(s.Name, "ok").Inc()
			} else {
				metrics.ReadyzCheckTotal.WithLabelValues(s.Name, "fail").Inc()
			}
		}
		status := http.StatusOK
		if !report.Healthy {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, report)
	}
}