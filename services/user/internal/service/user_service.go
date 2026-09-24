// Package service 是 user-service 的业务编排层。
//
// 设计要点：
//   - UserProfile 是给其他服务查的最小视图（id + 是否实名 + 角色）；
//     与 auth-service 的 repo.User 解耦。
//   - UpdateProfile / ChangeAvatar / ChangeNickname 都需要权限校验（userID == self）；
//     由 handler 层从 JWT 取 userID 后传入。
//   - GetProfile 走 cache（v1 不接，v2 引入 Redis 后加）；保持 pure function 易测。
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/growdu/doctors/shared/errs"
)

// Profile 是用户对外暴露的最小子集（供其他服务查）。
type Profile struct {
	ID               int64
	Phone            string
	Role             string
	Nickname         string
	AvatarURL        string
	RealNameVerified bool
	CreatedAt        time.Time
}

// ProfileRepo 是仓储契约（auth-service 的 UserRepo 子集）。
type ProfileRepo interface {
	GetByID(ctx context.Context, id int64) (*Profile, error)
	UpdateNickname(ctx context.Context, id int64, nickname string) error
	UpdateAvatar(ctx context.Context, id int64, avatarURL string) error
}

// ErrProfileNotFound 是查询无结果。
var ErrProfileNotFound = errors.New("user service: profile not found")

// Service 是 user 业务编排器。
type Service struct {
	repo ProfileRepo
}

// New 装配 Service。
func New(r ProfileRepo) *Service { return &Service{repo: r} }

// GetProfile 取一个用户的资料。
func (s *Service) GetProfile(ctx context.Context, id int64) (*Profile, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			return nil, errs.New(errs.CodeNotFound, "user not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "get profile", err)
	}
	return p, nil
}

// UpdateNickname 修改昵称；仅限本人。
func (s *Service) UpdateNickname(ctx context.Context, callerID, targetID int64, nickname string) error {
	if callerID != targetID {
		return errs.New(errs.CodeForbidden, "can only update own nickname")
	}
	if l := len(strings.TrimSpace(nickname)); l == 0 || l > 64 {
		return errs.New(errs.CodeParamInvalid, "nickname length must be 1..64")
	}
	if err := s.repo.UpdateNickname(ctx, targetID, nickname); err != nil {
		return errs.Wrap(errs.CodeInternal, "update nickname", err)
	}
	return nil
}

// UpdateAvatar 修改头像 URL；仅限本人。
func (s *Service) UpdateAvatar(ctx context.Context, callerID, targetID int64, avatarURL string) error {
	if callerID != targetID {
		return errs.New(errs.CodeForbidden, "can only update own avatar")
	}
	if avatarURL == "" {
		return errs.New(errs.CodeParamInvalid, "avatar_url required")
	}
	if l := len(avatarURL); l > 255 {
		return errs.New(errs.CodeParamInvalid, "avatar_url too long")
	}
	if err := s.repo.UpdateAvatar(ctx, targetID, avatarURL); err != nil {
		return errs.Wrap(errs.CodeInternal, "update avatar", err)
	}
	return nil
}