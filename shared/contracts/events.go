// Package contracts 定义跨服务共享的事件类型与常量。
//
// 设计要点：
//   - 各服务通过 Kafka 通信时，事件 schema 必须保持一致；本包就是 schema 的单一来源。
//   - 所有事件类型是 value struct，便于序列化 / 反序列化。
//   - 事件名（topic）用常量集中管理，避免拼写漂移。
//   - 字段尽量原子（int64 / 字符串 / 时间）；复杂对象（如 Order）走 id 引用 + 拉取，
//     避免事件膨胀导致 schema 演化困难。
//   - 本包不引入任何项目内 import，零依赖、纯数据。
package contracts

import "time"

// ---------- Kafka topic 常量 ----------

const (
	TopicOrderCreated   = "order.created"
	TopicOrderAccepted  = "order.accepted"
	TopicOrderCancelled = "order.cancelled"
	TopicOrderReviewed  = "order.reviewed"
	TopicOrderMatching  = "order.matching" // 锁单超时 / 拒接 → 回退 matching，重新进入匹配池

	TopicUserRegistered   = "user.registered"
	TopicUserRealNameDone = "user.real_name.done"

	TopicEscortRegistered = "escort.registered"
	TopicEscortAvailable  = "escort.available"
	TopicEscortUnavailable = "escort.unavailable"

	TopicPaymentCreated   = "payment.created"
	TopicPaymentCompleted = "payment.completed"
	TopicPaymentRefunded  = "payment.refunded"
	TopicRefundCompleted  = "refund.completed"

	TopicSOSRaised = "sos.raised"
	TopicMessageSent = "message.sent"
)

// ---------- Order 事件 ----------

// OrderCreatedEvent 订单创建事件（order-service → match-service / notification）。
type OrderCreatedEvent struct {
	OrderID        int64     `json:"order_id"`
	PatientID      int64     `json:"patient_id"`
	HospitalID     int64     `json:"hospital_id"`
	PackageID      int64     `json:"package_id"`
	ServiceStartAt time.Time `json:"service_start_at"`
	City           string    `json:"city"`
	HospitalLat    float64   `json:"hospital_lat"`
	HospitalLng    float64   `json:"hospital_lng"`
	Amount         float64   `json:"amount"`
}

// OrderAcceptedEvent 订单被陪诊师接单（order-service → notification / billing）。
type OrderAcceptedEvent struct {
	OrderID    int64     `json:"order_id"`
	EscortID   int64     `json:"escort_id"`
	AcceptedAt time.Time `json:"accepted_at"`
}

// OrderCancelledEvent 订单取消（order-service → billing / notification）。
type OrderCancelledEvent struct {
	OrderID     int64     `json:"order_id"`
	CancelledBy int64     `json:"cancelled_by"`
	Reason      string    `json:"reason,omitempty"`
	CancelledAt time.Time `json:"cancelled_at"`
}

// OrderReviewedEvent 评价完成（review-service → order-service 关闭订单 + 通知 escort）。
type OrderReviewedEvent struct {
	OrderID   int64     `json:"order_id"`
	ReviewerID int64    `json:"reviewer_id"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	ReviewedAt time.Time `json:"reviewed_at"`
}

// OrderMatchingEvent 订单重新进入匹配池（锁单超时回退 或 陪诊师拒接）。
// order-service → match-service：重新推送候选陪诊师。
type OrderMatchingEvent struct {
	OrderID   int64     `json:"order_id"`
	EscortID  int64     `json:"escort_id"`           // 拒接的 escort（超时则为 0）
	Reason    string    `json:"reason"`              // "lock_expired" | "escort_declined"
	RetriedAt time.Time `json:"retried_at"`
}

// ---------- User 事件 ----------

// UserRegisteredEvent 用户注册成功（auth-service → welcome notification）。
type UserRegisteredEvent struct {
	UserID    int64     `json:"user_id"`
	Phone     string    `json:"phone,omitempty"`
	Role      string    `json:"role"`
	Channel   string    `json:"channel"` // "sms" | "wx"
	CreatedAt time.Time `json:"created_at"`
}

// UserRealNameDoneEvent 实名完成（auth-service → order-service 解锁下单）。
type UserRealNameDoneEvent struct {
	UserID    int64     `json:"user_id"`
	CompletedAt time.Time `json:"completed_at"`
}

// ---------- Escort 事件 ----------

// EscortRegisteredEvent 陪诊师注册（auth-service / escort-service → 审核流程）。
type EscortRegisteredEvent struct {
	EscortID  int64     `json:"escort_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// EscortAvailableEvent 陪诊师上线（escort-service → match-service 写入候选池）。
type EscortAvailableEvent struct {
	EscortID int64     `json:"escort_id"`
	City     string    `json:"city"`
	Lat      float64   `json:"lat"`
	Lng      float64   `json:"lng"`
	ValidUntil time.Time `json:"valid_until"`
}

// EscortUnavailableEvent 陪诊师下线（escort-service → match-service 移除候选）。
type EscortUnavailableEvent struct {
	EscortID int64     `json:"escort_id"`
	OfflineAt time.Time `json:"offline_at"`
}

// ---------- Payment 事件 ----------

// PaymentCreatedEvent 支付单创建（payment-service → billing / notification）。
type PaymentCreatedEvent struct {
	PaymentID int64     `json:"payment_id"`
	OrderID   int64     `json:"order_id"`
	Amount    float64   `json:"amount"`
	Channel   string    `json:"channel"` // "wx" | "alipay" | "mock"
	CreatedAt time.Time `json:"created_at"`
}

// PaymentCompletedEvent 支付成功（payment-service → order-service 推进状态）。
type PaymentCompletedEvent struct {
	PaymentID  int64     `json:"payment_id"`
	OrderID    int64     `json:"order_id"`
	Amount     float64   `json:"amount"`
	CompletedAt time.Time `json:"completed_at"`
	ExternalTxID string  `json:"external_tx_id"`
}

// PaymentRefundedEvent 退款完成（payment-service → order-service + notification）。
type PaymentRefundedEvent struct {
	RefundID int64     `json:"refund_id"`
	PaymentID int64    `json:"payment_id"`
	OrderID   int64    `json:"order_id"`
	Amount    float64  `json:"amount"`
	RefundedAt time.Time `json:"refunded_at"`
}

// RefundResult 是退款单据的简化视图（refund-service → order-service）。
// 放在 contracts 是为了避免 order / payment 包互相引用。
type RefundResult struct {
	ID     int64   `json:"id"`
	Amount float64 `json:"amount"`
	Status string  `json:"status"` // created/processing/completed/failed/rejected
}

// RefundCompletedEvent 退款完成（refund-service → notification + audit）。
type RefundCompletedEvent struct {
	RefundID     int64     `json:"refund_id"`
	OrderID      int64     `json:"order_id"`
	Amount       float64   `json:"amount"`
	RefundedAt   time.Time `json:"refunded_at"`
	ExternalTxID string    `json:"external_tx_id,omitempty"`
}

// ---------- SOS / 消息事件 ----------

// SOSRaisedEvent 紧急信号（sos-service → notification + admin + escort）。
type SOSRaisedEvent struct {
	SOSID    int64     `json:"sos_id"`
	OrderID  int64     `json:"order_id"`
	UserID   int64     `json:"user_id"`
	Lat      float64   `json:"lat"`
	Lng      float64   `json:"lng"`
	Note     string    `json:"note,omitempty"`
	RaisedAt time.Time `json:"raised_at"`
}

// MessageSentEvent 消息已发送（message-service → notification 推送）。
type MessageSentEvent struct {
	MessageID int64     `json:"message_id"`
	OrderID   int64     `json:"order_id"`
	FromID    int64     `json:"from_id"`
	ToID      int64     `json:"to_id"`
	Body      string    `json:"body"`
	SentAt    time.Time `json:"sent_at"`
}