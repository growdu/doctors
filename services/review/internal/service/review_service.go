// Package service 是 review-service 的业务编排层。
//
// 设计要点：
//   - 一笔订单只能评一次（unique on order_id）。
//   - 评分 1-5；comment 长度限制。
//   - CreateReview 触发 contracts.OrderReviewedEvent → order-service 关闭订单。
//   - Reply 由 admin 写入回复内容；不影响评分。
package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/growdu/doctors/shared/contracts"
	"github.com/growdu/doctors/shared/errs"
)

// Review 是评价实体。
type Review struct {
	ID         int64
	OrderID    int64
	ReviewerID int64
	EscortID   int64
	Rating     int
	Comment    string
	Reply      string
	RepliedBy  int64
	RepliedAt  *time.Time
	CreatedAt  time.Time
}

// ReviewRepo 是仓储契约。
type ReviewRepo interface {
	Create(ctx context.Context, r *Review) error
	GetByID(ctx context.Context, id int64) (*Review, error)
	GetByOrderID(ctx context.Context, orderID int64) (*Review, error)
	ListByEscort(ctx context.Context, escortID int64, limit, offset int) ([]*Review, error)
	List(ctx context.Context, f ListFilter) ([]*Review, error)
	UpdateReply(ctx context.Context, id int64, reply string, adminID int64) error
}

// ListFilter 是 List 的过滤参数。
type ListFilter struct {
	EscortID  int64
	OrderID   int64
	MinRating int
	Page      int
	PageSize  int
}

// Publisher 是事件发布抽象。
type Publisher interface {
	PublishOrderReviewed(ctx context.Context, ev contracts.OrderReviewedEvent) error
}

// ErrReviewNotFound 查无结果。
var ErrReviewNotFound = errors.New("review service: not found")
var ErrDuplicateReview = errors.New("review service: duplicate review for order")

// Service 是 review 业务编排器。
type Service struct {
	repo      ReviewRepo
	publisher Publisher
}

// New 装配 Service。
func New(r ReviewRepo, p Publisher) *Service { return &Service{repo: r, publisher: p} }

// CreateReview 提交评价；每笔订单限一次。
func (s *Service) CreateReview(ctx context.Context, reviewerID, orderID, escortID int64, rating int, comment string) (*Review, error) {
	if reviewerID == 0 || orderID == 0 || escortID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "reviewer/order/escort id required")
	}
	if rating < 1 || rating > 5 {
		return nil, errs.New(errs.CodeParamInvalid, "rating must be 1..5")
	}
	if utf8.RuneCountInString(strings.TrimSpace(comment)) > 500 {
		return nil, errs.New(errs.CodeParamInvalid, "comment too long (max 500 chars)")
	}
	if existing, _ := s.repo.GetByOrderID(ctx, orderID); existing != nil {
		return nil, errs.New(errs.CodeConflict, "order already reviewed")
	}

	r := &Review{
		OrderID:    orderID,
		ReviewerID: reviewerID,
		EscortID:   escortID,
		Rating:     rating,
		Comment:    strings.TrimSpace(comment),
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "create review", err)
	}

	// 事件发布
	if s.publisher != nil {
		_ = s.publisher.PublishOrderReviewed(ctx, contracts.OrderReviewedEvent{
			OrderID:    orderID,
			ReviewerID: reviewerID,
			Rating:     rating,
			Comment:    r.Comment,
			ReviewedAt: r.CreatedAt,
		})
	}
	return r, nil
}

// GetByID 取评价详情。
func (s *Service) GetByID(ctx context.Context, id int64) (*Review, error) {
	if id == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "id required")
	}
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrReviewNotFound) {
			return nil, errs.New(errs.CodeNotFound, "review not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find review", err)
	}
	return r, nil
}

// GetByOrder 取订单的评价。
func (s *Service) GetByOrder(ctx context.Context, orderID int64) (*Review, error) {
	r, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrReviewNotFound) {
			return nil, errs.New(errs.CodeNotFound, "review not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find review", err)
	}
	return r, nil
}

// ListByEscort 拉陪诊师的评价列表（用于 escort 详情页）。
func (s *Service) ListByEscort(ctx context.Context, escortID int64, limit, offset int) ([]*Review, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListByEscort(ctx, escortID, limit, offset)
}

// List 综合过滤查询。
func (s *Service) List(ctx context.Context, f ListFilter) ([]*Review, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	if f.MinRating < 0 {
		f.MinRating = 0
	}
	if f.MinRating > 5 {
		f.MinRating = 5
	}
	return s.repo.List(ctx, f)
}

// Reply 由 admin 写入回复。
func (s *Service) Reply(ctx context.Context, id, adminID int64, body string) (*Review, error) {
	if id == 0 || adminID == 0 {
		return nil, errs.New(errs.CodeParamInvalid, "id/admin_id required")
	}
	body = strings.TrimSpace(body)
	if utf8.RuneCountInString(body) > 500 {
		return nil, errs.New(errs.CodeParamInvalid, "reply too long (max 500 chars)")
	}
	r, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrReviewNotFound) {
			return nil, errs.New(errs.CodeNotFound, "review not found")
		}
		return nil, errs.Wrap(errs.CodeInternal, "find review", err)
	}
	if r.Reply != "" {
		return nil, errs.New(errs.CodeConflict, "review already replied")
	}
	if err := s.repo.UpdateReply(ctx, id, body, adminID); err != nil {
		return nil, errs.Wrap(errs.CodeInternal, "update reply", err)
	}
	r.Reply = body
	r.RepliedBy = adminID
	return r, nil
}
