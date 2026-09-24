// Package sms 提供短信发送抽象。
//
// 设计要点：
//   - Sender 是接口，业务层只依赖接口，不绑定具体渠道。
//   - 默认实现 LogSender：写 stderr + 内存 map，便于本地开发查验证码。
//   - 真实渠道（阿里云 / 腾讯云 / 华为云）在 v2 接入；本期不做。
//   - ValidatePhone 用 E.164 兼容的国内 11 位手机号校验。
package sms

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
)

// Sender 抽象短信下发；返回 ErrSMS.New。
type Sender interface {
	Send(ctx context.Context, phone, code string) error
}

// phoneRE 匹配 11 位手机号或 +86 前缀的 13 位。
var phoneRE = regexp.MustCompile(`^(\+?86)?1[3-9]\d{9}$`)

// ValidatePhone 校验国内手机号；通过返回 true。
func ValidatePhone(phone string) bool {
	return phoneRE.MatchString(phone)
}

// LogSender 是开发期的默认实现。
type LogSender struct {
	mu    sync.RWMutex
	codes map[string]string
}

// NewLogSender 构造一个 LogSender。
func NewLogSender() *LogSender {
	return &LogSender{codes: make(map[string]string)}
}

// Send 把 phone → code 记入内存，同时写到全局 writer（默认 stderr）。
func (s *LogSender) Send(ctx context.Context, phone, code string) error {
	if !ValidatePhone(phone) {
		return fmt.Errorf("sms: invalid phone %q", phone)
	}
	if code == "" {
		return fmt.Errorf("sms: empty code")
	}

	s.mu.Lock()
	s.codes[phone] = code
	s.mu.Unlock()

	_, _ = fmt.Fprintf(sw, "[sms mock] phone=%s code=%s\n", phone, code)
	return nil
}

// LookupCode 仅供测试 / dev 工具查询最近一次发送的验证码。
func (s *LogSender) LookupCode(phone string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.codes[phone]
	return c, ok
}

// VerifyCode 是 LookupCode 的布尔别名，让 LogSender 同时实现 service.SMSSender。
func (s *LogSender) VerifyCode(phone, code string) bool {
	got, ok := s.LookupCode(phone)
	return ok && got == code
}

// 全局 writer，默认 stderr；测试可替换以捕获输出。
var (
	swMu sync.RWMutex
	sw   io.Writer = os.Stderr
)

func newBuffer() *lockedBuf { return &lockedBuf{} }

type lockedBuf struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *lockedBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *lockedBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}