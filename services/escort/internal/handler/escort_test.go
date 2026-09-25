package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/growdu/doctors/services/escort/internal/service"
	"github.com/growdu/doctors/shared/errs"
	"github.com/growdu/doctors/shared/httpx"
)

// stubSvc 是 handler.Service 的最小 fake。
type stubSvc struct {
	regEsc  *service.Escort
	regErr  error
	getEsc  *service.Escort
	getErr  error
	setErr  error
	locErr  error
	cityErr error

	createQual  *service.Qualification
	createQErr  error
	listQuals   []*service.Qualification
	listQErr    error
	updateQual  *service.Qualification
	updateQErr  error
	deleteQErr  error

	createTr  *service.Training
	createTrErr error
	listTrs   []*service.Training
	listTrErr  error
}

func (s *stubSvc) Register(ctx context.Context, userID int64) (*service.Escort, error) {
	if s.regErr != nil {
		return nil, s.regErr
	}
	return s.regEsc, nil
}
func (s *stubSvc) Get(ctx context.Context, id int64) (*service.Escort, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getEsc, nil
}
func (s *stubSvc) SetAvailability(ctx context.Context, id int64, available bool, validUntil time.Time) error {
	return s.setErr
}
func (s *stubSvc) UpdateLocation(ctx context.Context, callerID int64, lat, lng float64) error {
	return s.locErr
}
func (s *stubSvc) UpdateCity(ctx context.Context, callerID int64, city string) error {
	return s.cityErr
}

func (s *stubSvc) CreateQualification(ctx context.Context, callerID int64, q *service.Qualification) (*service.Qualification, error) {
	if s.createQErr != nil {
		return nil, s.createQErr
	}
	return s.createQual, nil
}
func (s *stubSvc) ListQualifications(ctx context.Context, callerID int64) ([]*service.Qualification, error) {
	if s.listQErr != nil {
		return nil, s.listQErr
	}
	return s.listQuals, nil
}
func (s *stubSvc) UpdateQualification(ctx context.Context, callerID, qid int64, patch map[string]any) (*service.Qualification, error) {
	if s.updateQErr != nil {
		return nil, s.updateQErr
	}
	return s.updateQual, nil
}
func (s *stubSvc) DeleteQualification(ctx context.Context, callerID, qid int64) error {
	return s.deleteQErr
}

func (s *stubSvc) CreateTraining(ctx context.Context, callerID int64, t *service.Training) (*service.Training, error) {
	if s.createTrErr != nil {
		return nil, s.createTrErr
	}
	return s.createTr, nil
}
func (s *stubSvc) ListTrainings(ctx context.Context, callerID int64) ([]*service.Training, error) {
	if s.listTrErr != nil {
		return nil, s.listTrErr
	}
	return s.listTrs, nil
}

func setupRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(func(c *gin.Context) {
		c.Set("escort_user_id", int64(42))
		c.Set("escort_role", "escort")
		c.Next()
	})
	h.RegisterRoutes(v1)
	return r
}

func doJSON(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// ---------- /me/location ----------

func TestUpdateLocation_OK(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPatch, "/api/v1/escorts/me/location",
		`{"lat":39.9,"lng":116.4}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestUpdateLocation_BadJSON(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPatch, "/api/v1/escorts/me/location", `{}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

func TestUpdateLocation_InternalError(t *testing.T) {
	stub := &stubSvc{locErr: errors.New("db")}
	w := doJSON(setupRouter(New(stub)), http.MethodPatch, "/api/v1/escorts/me/location",
		`{"lat":39.9,"lng":116.4}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeInternal), resp.Code)
}

// ---------- /me/city ----------

func TestUpdateCity_OK(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPatch, "/api/v1/escorts/me/city",
		`{"city":"北京"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestUpdateCity_BadJSON(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPatch, "/api/v1/escorts/me/city", `{}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

// ---------- /me/qualifications ----------

func TestCreateQualification_OK(t *testing.T) {
	stub := &stubSvc{createQual: &service.Qualification{ID: 7, Type: "doctor_license", Number: "DOC-1"}}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/escorts/me/qualifications",
		`{"type":"doctor_license","number":"DOC-1"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, int64(7), int64(resp.Data["id"].(float64)))
}

func TestCreateQualification_BadJSON(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/escorts/me/qualifications", `{}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

func TestCreateQualification_NotFound(t *testing.T) {
	stub := &stubSvc{createQErr: errs.New(errs.CodeNotFound, "escort not registered")}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/escorts/me/qualifications",
		`{"type":"doctor_license","number":"DOC-1"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeNotFound), resp.Code)
}

func TestListQualifications_OK(t *testing.T) {
	stub := &stubSvc{listQuals: []*service.Qualification{{ID: 1}, {ID: 2}}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/escorts/me/qualifications", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestUpdateQualification_OK(t *testing.T) {
	stub := &stubSvc{updateQual: &service.Qualification{ID: 7, Number: "NEW"}}
	w := doJSON(setupRouter(New(stub)), http.MethodPatch, "/api/v1/escorts/me/qualifications/7",
		`{"number":"NEW"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestUpdateQualification_InvalidID(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPatch, "/api/v1/escorts/me/qualifications/abc",
		`{"number":"X"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

func TestUpdateQualification_NotFound(t *testing.T) {
	stub := &stubSvc{updateQErr: errs.New(errs.CodeNotFound, "missing")}
	w := doJSON(setupRouter(New(stub)), http.MethodPatch, "/api/v1/escorts/me/qualifications/99",
		`{"number":"X"}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeNotFound), resp.Code)
}

func TestDeleteQualification_OK(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodDelete, "/api/v1/escorts/me/qualifications/7", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestDeleteQualification_InvalidID(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodDelete, "/api/v1/escorts/me/qualifications/abc", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

func TestDeleteQualification_NotFound(t *testing.T) {
	stub := &stubSvc{deleteQErr: errs.New(errs.CodeNotFound, "missing")}
	w := doJSON(setupRouter(New(stub)), http.MethodDelete, "/api/v1/escorts/me/qualifications/99", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeNotFound), resp.Code)
}

// ---------- /me/trainings ----------

func TestCreateTraining_OK(t *testing.T) {
	stub := &stubSvc{createTr: &service.Training{ID: 5, Title: "急救"}}
	w := doJSON(setupRouter(New(stub)), http.MethodPost, "/api/v1/escorts/me/trainings",
		`{"title":"急救"}`)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestCreateTraining_BadJSON(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodPost, "/api/v1/escorts/me/trainings", `{}`)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}

func TestListTrainings_OK(t *testing.T) {
	stub := &stubSvc{listTrs: []*service.Training{{ID: 1}}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/escorts/me/trainings", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestListTrainings_ServiceError(t *testing.T) {
	stub := &stubSvc{listTrErr: errors.New("db")}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/escorts/me/trainings", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeInternal), resp.Code)
}

// ---------- /escorts/:id ----------

func TestGet_OK(t *testing.T) {
	stub := &stubSvc{getEsc: &service.Escort{ID: 7, UserID: 100, Status: "available"}}
	w := doJSON(setupRouter(New(stub)), http.MethodGet, "/api/v1/escorts/7", "")
	assert.Equal(t, http.StatusOK, w.Code)
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
}

func TestGet_InvalidID(t *testing.T) {
	w := doJSON(setupRouter(New(&stubSvc{})), http.MethodGet, "/api/v1/escorts/abc", "")
	var resp httpx.Resp[map[string]any]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int(errs.CodeParamInvalid), resp.Code)
}
