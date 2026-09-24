package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authpkg "github.com/growdu/doctors/shared/auth"
)

// ---------- fake 实现（满足 UserRepo / SMSSender / WXLogin / RealNameVerifier） ----------

type userRec struct {
	id        int64
	phone     string
	role      string
	unionid   *string
	realDone  bool
	idHash    *string
	idTail    *string
}

type fakeRepo struct {
	users    map[int64]*userRec
	byPhone  map[string]int64
	byUnion  map[string]int64
	next     int64
	notFound bool // toggle：用于触发 not-found 路径
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users:   map[int64]*userRec{},
		byPhone: map[string]int64{},
		byUnion: map[string]int64{},
	}
}

func (r *fakeRepo) Create(ctx context.Context, phone, role string, unionid *string) (int64, error) {
	r.next++
	u := &userRec{id: r.next, phone: phone, role: role, unionid: unionid}
	r.users[u.id] = u
	if phone != "" {
		r.byPhone[phone] = u.id
	}
	if unionid != nil {
		r.byUnion[*unionid] = u.id
	}
	return u.id, nil
}

func (r *fakeRepo) FindByPhone(ctx context.Context, phone string) (*User, error) {
	if r.notFound {
		return nil, errFake
	}
	id, ok := r.byPhone[phone]
	if !ok {
		return nil, errFake
	}
	u := r.users[id]
	uid := u.id
	return &User{ID: uid, Phone: u.phone, Role: u.role, UnionID: u.unionid, RealNameVerified: u.realDone}, nil
}

func (r *fakeRepo) FindByUnionID(ctx context.Context, unionid string) (*User, error) {
	if r.notFound {
		return nil, errFake
	}
	id, ok := r.byUnion[unionid]
	if !ok {
		return nil, errFake
	}
	u := r.users[id]
	return &User{ID: u.id, Phone: u.phone, Role: u.role, UnionID: u.unionid, RealNameVerified: u.realDone}, nil
}

func (r *fakeRepo) FindByID(ctx context.Context, id int64) (*User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errFake
	}
	return &User{ID: u.id, Phone: u.phone, Role: u.role, UnionID: u.unionid, RealNameVerified: u.realDone}, nil
}

func (r *fakeRepo) UpdateRealName(ctx context.Context, id int64, hash, tail string) error {
	u, ok := r.users[id]
	if !ok {
		return errFake
	}
	h, t := hash, tail
	u.idHash = &h
	u.idTail = &t
	u.realDone = true
	return nil
}

var errFake = errors.New("fake: not found")

// userRec → 测试断言用
func (r *fakeRepo) raw(id int64) *userRec { return r.users[id] }

type fakeSMS struct {
	codes map[string]string
}

func (s *fakeSMS) Send(ctx context.Context, phone, code string) error {
	s.codes[phone] = code
	return nil
}
func (s *fakeSMS) VerifyCode(phone, code string) bool {
	return s.codes[phone] == code
}

type fakeWX struct{}

func (w *fakeWX) Code2Session(ctx context.Context, code string) (string, string, error) {
	if code == "" {
		return "", "", errors.New("wx: empty code")
	}
	return "unionid-" + code, "openid-" + code, nil
}

type fakeRealName struct{}

func (r *fakeRealName) Verify(ctx context.Context, name, idCard string) (bool, string, string, error) {
	return true, "hash-" + idCard, idCard[len(idCard)-4:], nil
}

// ---------- 装配 helpers ----------

func newService() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	svc := &Service{
		repo:      repo,
		sms:       &fakeSMS{codes: map[string]string{}},
		wx:        &fakeWX{},
		realName:  &fakeRealName{},
		jwtSecret: "test-secret",
		jwtTTL:    60_000_000_000, // 1min
	}
	return svc, repo
}

// ---------- 行为用例 ----------

func TestSendSMS_StoresCode(t *testing.T) {
	svc, _ := newService()
	require.NoError(t, svc.SendSMS(context.Background(), "13800138000"))
}

func TestLoginBySMS_CreatesUserIfNotExist(t *testing.T) {
	svc, r := newService()
	require.NoError(t, svc.SendSMS(context.Background(), "13800138000"))
	sender := svc.sms.(*fakeSMS)
	code := sender.codes["13800138000"]
	require.NotEmpty(t, code)

	tok, uid, err := svc.LoginBySMS(context.Background(), "13800138000", code)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
	assert.NotZero(t, uid)
	assert.Equal(t, "13800138000", r.raw(uid).phone)
	assert.Equal(t, "patient", r.raw(uid).role)
}

func TestLoginBySMS_RejectsWrongCode(t *testing.T) {
	svc, _ := newService()
	require.NoError(t, svc.SendSMS(context.Background(), "13800138000"))
	_, _, err := svc.LoginBySMS(context.Background(), "13800138000", "wrong")
	assert.Error(t, err)
}

func TestLoginBySMS_RejectsInvalidPhone(t *testing.T) {
	svc, _ := newService()
	_, _, err := svc.LoginBySMS(context.Background(), "abc", "123456")
	assert.Error(t, err)
}

func TestLoginBySMS_ExistingUser(t *testing.T) {
	svc, r := newService()
	require.NoError(t, svc.SendSMS(context.Background(), "13800138000"))
	sender := svc.sms.(*fakeSMS)
	_, uid1, err := svc.LoginBySMS(context.Background(), "13800138000", sender.codes["13800138000"])
	require.NoError(t, err)

	require.NoError(t, svc.SendSMS(context.Background(), "13800138000"))
	_, uid2, err := svc.LoginBySMS(context.Background(), "13800138000", sender.codes["13800138000"])
	require.NoError(t, err)
	assert.Equal(t, uid1, uid2)
	assert.Len(t, r.users, 1)
}

func TestLoginByWX_BindsToExistingUserByUnionid(t *testing.T) {
	svc, r := newService()

	_, uid1, err := svc.LoginByWX(context.Background(), "user-1")
	require.NoError(t, err)

	_, uid2, err := svc.LoginByWX(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, uid1, uid2)
	assert.Len(t, r.users, 1)
}

func TestLoginByWX_CreatesUserOnFirstSeen(t *testing.T) {
	svc, r := newService()
	_, uid, err := svc.LoginByWX(context.Background(), "user-X")
	require.NoError(t, err)
	assert.NotZero(t, uid)
	assert.Len(t, r.users, 1)
}

func TestRealNameAuth_StoresHashAndTail(t *testing.T) {
	svc, r := newService()
	require.NoError(t, svc.SendSMS(context.Background(), "13800138000"))
	sender := svc.sms.(*fakeSMS)
	_, uid, err := svc.LoginBySMS(context.Background(), "13800138000", sender.codes["13800138000"])
	require.NoError(t, err)

	err = svc.RealNameAuth(context.Background(), uid, "张三", "110101199001011234")
	require.NoError(t, err)

	u := r.raw(uid)
	assert.True(t, u.realDone)
	require.NotNil(t, u.idHash)
	assert.Equal(t, "hash-110101199001011234", *u.idHash)
	require.NotNil(t, u.idTail)
	assert.Equal(t, "1234", *u.idTail)
}

func TestRefreshToken_RoundTrip(t *testing.T) {
	svc, _ := newService()
	require.NoError(t, svc.SendSMS(context.Background(), "13800138000"))
	sender := svc.sms.(*fakeSMS)
	tok1, _, err := svc.LoginBySMS(context.Background(), "13800138000", sender.codes["13800138000"])
	require.NoError(t, err)

	tok2, err := svc.Refresh(context.Background(), tok1)
	require.NoError(t, err)
	assert.NotEmpty(t, tok2)

	claims, err := authpkg.Parse("test-secret", tok2)
	require.NoError(t, err)
	assert.Equal(t, "patient", claims.Role)
	assert.NotZero(t, claims.UserID)
}

func TestMe_ReturnsUser(t *testing.T) {
	svc, _ := newService()
	require.NoError(t, svc.SendSMS(context.Background(), "13800138000"))
	sender := svc.sms.(*fakeSMS)
	_, uid, err := svc.LoginBySMS(context.Background(), "13800138000", sender.codes["13800138000"])
	require.NoError(t, err)

	u, err := svc.Me(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, uid, u.ID)
	assert.Equal(t, "13800138000", u.Phone)
}

func TestMe_NotFound(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Me(context.Background(), 999)
	assert.Error(t, err)
}

// ensure jwt import is used.
var _ = jwt.NewNumericDate