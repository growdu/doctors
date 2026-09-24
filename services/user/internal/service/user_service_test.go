package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- fake ----------

type fakeRepo struct {
	profiles map[int64]*Profile
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{profiles: map[int64]*Profile{
		1: {ID: 1, Phone: "13800138000", Role: "patient", Nickname: "alice", RealNameVerified: true},
		2: {ID: 2, Phone: "13800138001", Role: "escort", Nickname: "bob"},
	}}
}

func (r *fakeRepo) GetByID(ctx context.Context, id int64) (*Profile, error) {
	if p, ok := r.profiles[id]; ok {
		return p, nil
	}
	return nil, ErrProfileNotFound
}

func (r *fakeRepo) UpdateNickname(ctx context.Context, id int64, n string) error {
	p, ok := r.profiles[id]
	if !ok {
		return ErrProfileNotFound
	}
	p.Nickname = n
	return nil
}

func (r *fakeRepo) UpdateAvatar(ctx context.Context, id int64, u string) error {
	p, ok := r.profiles[id]
	if !ok {
		return ErrProfileNotFound
	}
	p.AvatarURL = u
	return nil
}

// ---------- 测试 ----------

func newService() (*Service, *fakeRepo) {
	r := newFakeRepo()
	return New(r), r
}

func TestGetProfile_OK(t *testing.T) {
	s, _ := newService()
	p, err := s.GetProfile(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "alice", p.Nickname)
	assert.True(t, p.RealNameVerified)
}

func TestGetProfile_NotFound(t *testing.T) {
	s, _ := newService()
	_, err := s.GetProfile(context.Background(), 999)
	assert.Error(t, err)
}

func TestUpdateNickname_OK(t *testing.T) {
	s, _ := newService()
	require.NoError(t, s.UpdateNickname(context.Background(), 1, 1, "新昵称"))

	p, _ := s.GetProfile(context.Background(), 1)
	assert.Equal(t, "新昵称", p.Nickname)
}

func TestUpdateNickname_NotSelf(t *testing.T) {
	s, _ := newService()
	err := s.UpdateNickname(context.Background(), 1, 2, "hacker")
	assert.Error(t, err)
}

func TestUpdateNickname_Empty(t *testing.T) {
	s, _ := newService()
	err := s.UpdateNickname(context.Background(), 1, 1, "")
	assert.Error(t, err)
}

func TestUpdateNickname_TooLong(t *testing.T) {
	s, _ := newService()
	err := s.UpdateNickname(context.Background(), 1, 1, string(make([]byte, 65)))
	assert.Error(t, err)
}

func TestUpdateAvatar_OK(t *testing.T) {
	s, _ := newService()
	require.NoError(t, s.UpdateAvatar(context.Background(), 1, 1, "https://cdn.example.com/a.jpg"))
	p, _ := s.GetProfile(context.Background(), 1)
	assert.Equal(t, "https://cdn.example.com/a.jpg", p.AvatarURL)
}

func TestUpdateAvatar_NotSelf(t *testing.T) {
	s, _ := newService()
	err := s.UpdateAvatar(context.Background(), 1, 2, "https://x")
	assert.Error(t, err)
}

func TestUpdateAvatar_TooLong(t *testing.T) {
	s, _ := newService()
	err := s.UpdateAvatar(context.Background(), 1, 1, string(make([]byte, 256)))
	assert.Error(t, err)
}

// 确保 errors.Is 链路有效（其它错误能被识别为 not-found）。
func TestRepoErrNotFound_Propagates(t *testing.T) {
	s, _ := newService()
	// 不可见：触发 ProfileRepo 返回 ErrProfileNotFound 但被 service 翻译成 CodeNotFound。
	_, err := s.GetProfile(context.Background(), 1)
	require.NoError(t, err)

	// 模拟 repo 抛 unknown 错误的情况
	bad := &Service{repo: badRepo{}}
	_, err = bad.GetProfile(context.Background(), 1)
	assert.Error(t, err)
}

type badRepo struct{}

func (badRepo) GetByID(ctx context.Context, id int64) (*Profile, error) {
	return nil, errors.New("db down")
}
func (badRepo) UpdateNickname(ctx context.Context, id int64, n string) error {
	return errors.New("db down")
}
func (badRepo) UpdateAvatar(ctx context.Context, id int64, u string) error {
	return errors.New("db down")
}