package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/order/internal/repo"
	"github.com/growdu/doctors/shared/contracts"
)

// ---------- fake ----------

type fakeOrderRepo struct {
	orders []*repo.Order
	events []*repo.OrderEvent
	nextID int64
}

func newFakeOrderRepo() *fakeOrderRepo { return &fakeOrderRepo{} }

func (r *fakeOrderRepo) Create(ctx context.Context, o *repo.Order) error {
	r.nextID++
	o.ID = r.nextID
	r.orders = append(r.orders, o)
	return nil
}
func (r *fakeOrderRepo) FindByID(ctx context.Context, id int64) (*repo.Order, error) {
	for _, o := range r.orders {
		if o.ID == id {
			return o, nil
		}
	}
	return nil, repo.ErrOrderNotFound
}
func (r *fakeOrderRepo) ListByPatient(ctx context.Context, patientID int64, limit, offset int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.PatientID == patientID {
			out = append(out, o)
		}
	}
	return out, nil
}
func (r *fakeOrderRepo) UpdateStatus(ctx context.Context, id int64, to string, ver int, escortID *int64) error {
	for _, o := range r.orders {
		if o.ID == id && o.Version == ver {
			o.Status = to
			o.Version = ver + 1
			if escortID != nil {
				o.EscortID = escortID
			}
			return nil
		}
	}
	return errors.New("fake: order not found or version mismatch")
}
func (r *fakeOrderRepo) InsertEvent(ctx context.Context, orderID int64, from *string, to string, actorID *int64, payload []byte) error {
	r.events = append(r.events, &repo.OrderEvent{
		OrderID: orderID, FromStatus: from, ToStatus: to, ActorID: actorID, Payload: payload,
	})
	return nil
}
func (r *fakeOrderRepo) ListEvents(ctx context.Context, orderID int64) ([]*repo.OrderEvent, error) {
	return r.events, nil
}

// ===== v1.1 state machine：fake 实现选人 + 30s 确认窗口四件套 =====
func (r *fakeOrderRepo) SelectForEscort(ctx context.Context, id int64, escortID int64, expireAt time.Time, expectVersion int) error {
	for _, o := range r.orders {
		if o.ID == id && o.Version == expectVersion && o.Status == "selecting_escort" {
			o.SelectedEscortID = &escortID
			t := expireAt
			o.EscortPendingExpireAt = &t
			o.Status = "escort_pending_acceptance"
			o.Version = expectVersion + 1
			return nil
		}
	}
	return repo.ErrVersionConflict
}

func (r *fakeOrderRepo) ConfirmByEscort(ctx context.Context, id int64, escortID int64, now time.Time, expectVersion int) error {
	for _, o := range r.orders {
		if o.ID == id && o.Version == expectVersion &&
			o.Status == "escort_pending_acceptance" &&
			o.SelectedEscortID != nil && *o.SelectedEscortID == escortID &&
			(o.EscortPendingExpireAt == nil || o.EscortPendingExpireAt.After(now)) {
			o.EscortID = &escortID
			o.SelectedEscortID = nil
			o.EscortPendingExpireAt = nil
			o.Status = "accepted"
			o.Version = expectVersion + 1
			return nil
		}
	}
	return repo.ErrInvalidStateForConfirm
}

func (r *fakeOrderRepo) RejectByEscort(ctx context.Context, id int64, escortID int64, expectVersion int) error {
	for _, o := range r.orders {
		if o.ID == id && o.Version == expectVersion && o.Status == "escort_pending_acceptance" {
			o.SelectedEscortID = nil
			o.EscortPendingExpireAt = nil
			o.Status = "selecting_escort"
			o.Version = expectVersion + 1
			return nil
		}
	}
	return repo.ErrVersionConflict
}

func (r *fakeOrderRepo) PendingExpired(ctx context.Context, now time.Time, limit int) ([]*repo.Order, error) {
	out := make([]*repo.Order, 0)
	for _, o := range r.orders {
		if o.Status == "escort_pending_acceptance" && o.SelectedEscortID != nil && o.EscortPendingExpireAt != nil && o.EscortPendingExpireAt.Before(now) {
			out = append(out, o)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

type fakeUserRepo struct {
	users map[int64]*UserSnapshot
}

func (r *fakeUserRepo) FindByID(ctx context.Context, id int64) (*UserSnapshot, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, errors.New("fake: user not found")
	}
	return u, nil
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[int64]*UserSnapshot{}}
}

// ---------- 用 service 包内的接口 ----------

func newService(t *testing.T, oRepo OrderRepo, uRepo UserLookup) *Service {
	return &Service{
		orders:    oRepo,
		users:     uRepo,
		clockNow:  time.Now,
	}
}

func validCreateReq() CreateReq {
	return CreateReq{
		PatientID:      10,
		HospitalID:     100,
		PackageID:      1,
		ServiceStartAt: time.Now().Add(24 * time.Hour),
		Amount:         200.00,
	}
}

// ---------- 测试 ----------

func TestCreate_Success(t *testing.T) {
	uRepo := newFakeUserRepo()
	uRepo.users[10] = &UserSnapshot{ID: 10, Role: "patient", RealNameVerified: true}
	oRepo := newFakeOrderRepo()
	svc := newService(t, oRepo, uRepo)

	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)
	assert.NotZero(t, o.ID)
	assert.Equal(t, "created", o.Status)
	assert.NotEmpty(t, o.OrderNo)
	assert.Len(t, oRepo.events, 1, "必须写一条 order_event")
}

func TestCreate_UserNotFound(t *testing.T) {
	uRepo := newFakeUserRepo()
	oRepo := newFakeOrderRepo()
	svc := newService(t, oRepo, uRepo)

	_, err := svc.Create(context.Background(), validCreateReq())
	assert.Error(t, err)
}

func TestCreate_NotRealNameVerified(t *testing.T) {
	uRepo := newFakeUserRepo()
	uRepo.users[10] = &UserSnapshot{ID: 10, Role: "patient", RealNameVerified: false}
	oRepo := newFakeOrderRepo()
	svc := newService(t, oRepo, uRepo)

	_, err := svc.Create(context.Background(), validCreateReq())
	assert.Error(t, err)
}

func TestCreate_NegativeAmount(t *testing.T) {
	uRepo := newFakeUserRepo()
	uRepo.users[10] = &UserSnapshot{ID: 10, RealNameVerified: true}
	oRepo := newFakeOrderRepo()
	svc := newService(t, oRepo, uRepo)

	req := validCreateReq()
	req.Amount = -1
	_, err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

func TestCreate_PastStartTime(t *testing.T) {
	uRepo := newFakeUserRepo()
	uRepo.users[10] = &UserSnapshot{ID: 10, RealNameVerified: true}
	oRepo := newFakeOrderRepo()
	svc := newService(t, oRepo, uRepo)

	req := validCreateReq()
	req.ServiceStartAt = time.Now().Add(-1 * time.Hour)
	_, err := svc.Create(context.Background(), req)
	assert.Error(t, err)
}

func TestList_MyOrders(t *testing.T) {
	uRepo := newFakeUserRepo()
	uRepo.users[10] = &UserSnapshot{ID: 10, RealNameVerified: true}
	oRepo := newFakeOrderRepo()
	svc := newService(t, oRepo, uRepo)

	for i := 0; i < 3; i++ {
		req := validCreateReq()
		req.HospitalID = int64(100 + i)
		_, err := svc.Create(context.Background(), req)
		require.NoError(t, err)
	}
	list, err := svc.List(context.Background(), 10, 10, 0)
	require.NoError(t, err)
	assert.Len(t, list, 3)
}

func TestGet_OK(t *testing.T) {
	uRepo := newFakeUserRepo()
	uRepo.users[10] = &UserSnapshot{ID: 10, RealNameVerified: true}
	oRepo := newFakeOrderRepo()
	svc := newService(t, oRepo, uRepo)
	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)

	got, err := svc.Get(context.Background(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, o.ID, got.ID)
}

func TestGet_NotFound(t *testing.T) {
	svc := newService(t, newFakeOrderRepo(), newFakeUserRepo())
	_, err := svc.Get(context.Background(), 999)
	assert.Error(t, err)
}
// fakePub 验证事件发布被调用（v1.1：4 新事件 + 删 OrderMatching）。
type fakePub struct {
	created            int
	accepted           int
	cancelled          int
	selectingEscort    int
	escortSelected     int
	escortConfirmed    int
	escortRejected     int
}

func (p *fakePub) PublishOrderCreated(ctx context.Context, ev contracts.OrderCreatedEvent) error {
	p.created++
	return nil
}
func (p *fakePub) PublishOrderAccepted(ctx context.Context, ev contracts.OrderAcceptedEvent) error {
	p.accepted++
	return nil
}
func (p *fakePub) PublishOrderCancelled(ctx context.Context, ev contracts.OrderCancelledEvent) error {
	p.cancelled++
	return nil
}
func (p *fakePub) PublishOrderSelectingEscort(ctx context.Context, ev contracts.OrderSelectingEscortEvent) error {
	p.selectingEscort++
	return nil
}
func (p *fakePub) PublishOrderEscortSelected(ctx context.Context, ev contracts.OrderEscortSelectedEvent) error {
	p.escortSelected++
	return nil
}
func (p *fakePub) PublishOrderEscortConfirmed(ctx context.Context, ev contracts.OrderEscortConfirmedEvent) error {
	p.escortConfirmed++
	return nil
}
func (p *fakePub) PublishOrderEscortRejected(ctx context.Context, ev contracts.OrderEscortRejectedEvent) error {
	p.escortRejected++
	return nil
}
func (p *fakePub) Close() error { return nil }

// TestCreate_PublishesOrderCreated 验证 Create 后 OrderCreatedEvent 被发布。
func TestCreate_PublishesOrderCreated(t *testing.T) {
	uRepo := seedVerifiedPatient(newFakeUserRepo())
	svc := newService(t, newFakeOrderRepo(), uRepo)
	pub := &fakePub{}
	svc.WithPublisher(pub)
	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)
	assert.Equal(t, 1, pub.created)
	assert.NotZero(t, o.ID)
}

// TestCancel_PublishesOrderCancelled 验证 Cancel 后 OrderCancelledEvent 被发布。
func TestCancel_PublishesOrderCancelled(t *testing.T) {
	uRepo := seedVerifiedPatient(newFakeUserRepo())
	svc := newService(t, newFakeOrderRepo(), uRepo)
	pub := &fakePub{}
	svc.WithPublisher(pub)
	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)
	require.NoError(t, svc.Cancel(context.Background(), o.ID, o.PatientID, "patient changed mind"))
	assert.Equal(t, 1, pub.cancelled)
}

// fakeRefundSvc 满足 service.RefundService 接口。
type fakeRefundSvc struct {
	called   int
	orderID  int64
	reason   string
	res      *contracts.RefundResult
	err      error
}

func (f *fakeRefundSvc) Refund(ctx context.Context, orderID int64, reason string) (*contracts.RefundResult, error) {
	f.called++
	f.orderID = orderID
	f.reason = reason
	if f.err != nil {
		return nil, f.err
	}
	if f.res != nil {
		return f.res, nil
	}
	return &contracts.RefundResult{ID: 1, Amount: 200, Status: "completed"}, nil
}

// TestCancel_TriggersRefund 验证 Cancel 后调 RefundService.Refund（user_cancel）。
func TestCancel_TriggersRefund(t *testing.T) {
	uRepo := seedVerifiedPatient(newFakeUserRepo())
	svc := newService(t, newFakeOrderRepo(), uRepo)
	rf := &fakeRefundSvc{}
	svc.WithRefundService(rf)

	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)
	require.NoError(t, svc.Cancel(context.Background(), o.ID, o.PatientID, "changed mind"))

	assert.Equal(t, 1, rf.called)
	assert.Equal(t, o.ID, rf.orderID)
	assert.Equal(t, "user_cancel", rf.reason, "患者主动取消 → user_cancel")
}

// TestCancel_AdminCancel_TriggersRefund 验证非患者取消 → admin_cancel。
func TestCancel_AdminCancel_TriggersRefund(t *testing.T) {
	uRepo := seedVerifiedPatient(newFakeUserRepo())
	svc := newService(t, newFakeOrderRepo(), uRepo)
	rf := &fakeRefundSvc{}
	svc.WithRefundService(rf)

	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)
	// actorID != PatientID → admin_cancel
	require.NoError(t, svc.Cancel(context.Background(), o.ID, 99999, "admin force"))
	assert.Equal(t, "admin_cancel", rf.reason)
}

// TestCancel_RefundSvcFailure_DoesNotBlockCancel 验证 refund 失败不影响 Cancel 成功。
func TestCancel_RefundSvcFailure_DoesNotBlockCancel(t *testing.T) {
	uRepo := seedVerifiedPatient(newFakeUserRepo())
	svc := newService(t, newFakeOrderRepo(), uRepo)
	rf := &fakeRefundSvc{err: errors.New("refund service down")}
	svc.WithRefundService(rf)

	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)
	// Cancel 仍应成功（refund 失败仅 log）
	require.NoError(t, svc.Cancel(context.Background(), o.ID, o.PatientID, "x"))
	assert.Equal(t, 1, rf.called, "即使 refund 失败也应被调用一次")
}

// TestCancel_NoRefundService_NilSafe 验证 refund=nil 时 Cancel 不 panic。
func TestCancel_NoRefundService_NilSafe(t *testing.T) {
	uRepo := seedVerifiedPatient(newFakeUserRepo())
	svc := newService(t, newFakeOrderRepo(), uRepo)
	// 不调 WithRefundService
	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)
	require.NoError(t, svc.Cancel(context.Background(), o.ID, o.PatientID, "x"))
}

// TestNoPublisher_NilSafe 验证 publisher=nil 时业务能跑。
func TestNoPublisher_NilSafe(t *testing.T) {
	uRepo := seedVerifiedPatient(newFakeUserRepo())
	svc := newService(t, newFakeOrderRepo(), uRepo)
	o, err := svc.Create(context.Background(), validCreateReq())
	require.NoError(t, err)
	require.NoError(t, svc.Cancel(context.Background(), o.ID, o.PatientID, "x"))
}

// ensure contracts imported (avoid unused).
var _ = contracts.OrderCreatedEvent{}

// helper to seed a verified patient at id=10.
func seedVerifiedPatient(uRepo *fakeUserRepo) *fakeUserRepo {
	uRepo.users[10] = &UserSnapshot{ID: 10, Role: "patient", RealNameVerified: true}
	return uRepo
}
