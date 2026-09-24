// Package service 是 auth-service 的业务编排层。
//
// 设计要点：
//   - 业务规则（参数校验、用户查找、JWT 签发）放这里；
//     数据访问（SQL）放 repo；渠道适配（短信 / 微信 / 实名）放子包。
//   - 所有依赖都是接口；fakeRepo / fakeSMS 等替身写在 _test.go。
//   - SMS 验证码：Send 写入渠道 → Login 时再 VerifyCode 比对；
//     默认 LogSender 同时支持 send + lookup，正好充当 verifier。
//   - 微信登录：以 wxunionid 为唯一标识；新 unionid 自动创建 patient 用户。
//   - JWT 用 HS256（shared/auth.Sign），claim 含 user_id / role / unionid。
package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	authpkg "github.com/growdu/doctors/shared/auth"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/services/auth/internal/sms"
)

// User 是业务层对外暴露的用户视图；与 repo.User 解耦。
type User struct {
	ID               int64
	Phone            string
	Role             string
	UnionID          *string
	RealNameVerified bool
}

// UserRepo 是仓储的最小契约；具体实现可指向真实 PG 或 fake。
type UserRepo interface {
	Create(ctx context.Context, phone, role string, unionid *string) (int64, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
	FindByUnionID(ctx context.Context, unionid string) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	UpdateRealName(ctx context.Context, id int64, hash, tail string) error
}

// SMSSender 同时承担下发与核验（默认实现是 *sms.LogSender）。
type SMSSender interface {
	Send(ctx context.Context, phone, code string) error
	VerifyCode(phone, code string) bool
}

// WXLogin 抽象微信 code2session。
type WXLogin interface {
	Code2Session(ctx context.Context, code string) (unionid, openid string, err error)
}

// RealNameVerifier 抽象实名核验。
type RealNameVerifier interface {
	Verify(ctx context.Context, name, idCard string) (verified bool, hash, tail string, err error)
}

// Service 是业务编排器。
type Service struct {
	repo      UserRepo
	sms       SMSSender
	wx        WXLogin
	realName  RealNameVerifier
	jwtSecret string
	jwtTTL    time.Duration
}

// New 装配一个 Service。
func New(repo UserRepo, sms SMSSender, wx WXLogin, realName RealNameVerifier, jwtSecret string, jwtTTL time.Duration) *Service {
	return &Service{
		repo:      repo,
		sms:       sms,
		wx:        wx,
		realName:  realName,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

// SMS 验证码长度。
const smsCodeLen = 6

// SendSMS 生成 6 位数字验证码并下发。
func (s *Service) SendSMS(ctx context.Context, phone string) error {
	if !sms.ValidatePhone(phone) {
		return errs.New(errs.CodeParamInvalid, "invalid phone")
	}
	code, err := randomDigits(smsCodeLen)
	if err != nil {
		return errs.Wrap(errs.CodeInternal, "generate sms code", err)
	}
	if err := s.sms.Send(ctx, phone, code); err != nil {
		return errs.Wrap(errs.CodeInternal, "send sms", err)
	}
	return nil
}

// LoginBySMS 用手机号 + 验证码登录；首次登录自动创建 patient 用户。
func (s *Service) LoginBySMS(ctx context.Context, phone, code string) (string, int64, error) {
	if !sms.ValidatePhone(phone) {
		return "", 0, errs.New(errs.CodeParamInvalid, "invalid phone")
	}
	if code == "" {
		return "", 0, errs.New(errs.CodeParamInvalid, "empty code")
	}
	if !s.sms.VerifyCode(phone, code) {
		return "", 0, errs.New(errs.CodeParamInvalid, "wrong code")
	}

	u, err := s.repo.FindByPhone(ctx, phone)
	if err != nil {
		if !isNotFound(err) {
			return "", 0, errs.Wrap(errs.CodeInternal, "find user", err)
		}
		uid, err := s.repo.Create(ctx, phone, "patient", nil)
		if err != nil {
			return "", 0, errs.Wrap(errs.CodeInternal, "create user", err)
		}
		tok, err := s.signToken(uid, "patient", nil)
		return tok, uid, err
	}
	tok, err := s.signToken(u.ID, u.Role, u.UnionID)
	return tok, u.ID, err
}

// LoginByWX 用微信 code 登录；首次见 unionid 自动创建 patient 用户。
func (s *Service) LoginByWX(ctx context.Context, code string) (string, int64, error) {
	if strings.TrimSpace(code) == "" {
		return "", 0, errs.New(errs.CodeParamInvalid, "empty code")
	}
	unionid, _, err := s.wx.Code2Session(ctx, code)
	if err != nil {
		return "", 0, errs.Wrap(errs.CodeUnauthorized, "wx code2session", err)
	}

	u, err := s.repo.FindByUnionID(ctx, unionid)
	if err != nil {
		if !isNotFound(err) {
			return "", 0, errs.Wrap(errs.CodeInternal, "find user by unionid", err)
		}
		uid, err := s.repo.Create(ctx, "", "patient", &unionid)
		if err != nil {
			return "", 0, errs.Wrap(errs.CodeInternal, "create wx user", err)
		}
		tok, err := s.signToken(uid, "patient", &unionid)
		return tok, uid, err
	}
	tok, err := s.signToken(u.ID, u.Role, u.UnionID)
	return tok, u.ID, err
}

// Refresh 用旧 token 换新 token。
func (s *Service) Refresh(ctx context.Context, token string) (string, error) {
	claims, err := authpkg.Parse(s.jwtSecret, token)
	if err != nil {
		return "", errs.Wrap(errs.CodeUnauthorized, "parse token", err)
	}
	return s.signToken(claims.UserID, claims.Role, &claims.UnionID)
}

// RealNameAuth 实名认证；写入哈希 + 末四位，不存明文。
func (s *Service) RealNameAuth(ctx context.Context, userID int64, name, idCard string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(idCard) == "" {
		return errs.New(errs.CodeParamInvalid, "empty name or idCard")
	}
	if len(idCard) < 4 {
		return errs.New(errs.CodeParamInvalid, "idCard too short")
	}
	ok, hash, tail, err := s.realName.Verify(ctx, name, idCard)
	if err != nil {
		return errs.Wrap(errs.CodeInternal, "verify real name", err)
	}
	if !ok {
		return errs.New(errs.CodeParamInvalid, "verify failed")
	}
	if err := s.repo.UpdateRealName(ctx, userID, hash, tail); err != nil {
		return errs.Wrap(errs.CodeInternal, "update real name", err)
	}
	return nil
}

// Me 返回当前用户。
func (s *Service) Me(ctx context.Context, userID int64) (*User, error) {
	u, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if isNotFound(err) {
			return nil, errs.New(errs.CodeNotFound, "user not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find user", err)
	}
	return u, nil
}

// signToken 用 HS256 签发 token。
func (s *Service) signToken(userID int64, role string, unionid *string) (string, error) {
	now := time.Now()
	claims := authpkg.Claims{
		UserID:  userID,
		Role:    role,
		UnionID: deref(unionid),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtTTL)),
		},
	}
	return authpkg.Sign(s.jwtSecret, claims)
}

// isNotFound 通过错误文本识别 "not found"（生产 Repo 替换 §2.3 时统一改为 errors.Is）。
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "not found")
}

// randomDigits 生成 n 位十进制字符串。
func randomDigits(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("random: n must be > 0")
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		r, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteByte(byte('0' + r.Int64()))
	}
	return b.String(), nil
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// 保留 errors 引用，避免 lint 报错。
var _ = errors.New