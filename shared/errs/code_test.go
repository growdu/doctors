package errs_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/growdu/doctors/shared/errs"
)

func TestCode_HTTPStatus(t *testing.T) {
	cases := []struct {
		name string
		code errs.Code
		want int
	}{
		{"ok", errs.CodeOK, http.StatusOK},
		{"param_invalid", errs.CodeParamInvalid, http.StatusBadRequest},
		{"unauthorized", errs.CodeUnauthorized, http.StatusUnauthorized},
		{"forbidden", errs.CodeForbidden, http.StatusForbidden},
		{"not_found", errs.CodeNotFound, http.StatusNotFound},
		{"conflict", errs.CodeConflict, http.StatusConflict},
		{"rate_limit", errs.CodeRateLimit, http.StatusTooManyRequests},
		{"internal", errs.CodeInternal, http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.code.HTTPStatus())
		})
	}
}

func TestCode_IsSuccess(t *testing.T) {
	assert.True(t, errs.CodeOK.IsSuccess())
	assert.False(t, errs.CodeParamInvalid.IsSuccess())
	assert.False(t, errs.CodeInternal.IsSuccess())
}

func TestError_ErrorMessage(t *testing.T) {
	e := errs.New(errs.CodeParamInvalid, "phone invalid")
	assert.Equal(t, "phone invalid", e.Error())
	assert.Equal(t, errs.CodeParamInvalid, e.Code)
}

func TestError_WithCause(t *testing.T) {
	root := errors.New("db down")
	e := errs.Wrap(errs.CodeInternal, "查询失败", root)
	assert.Equal(t, "查询失败: db down", e.Error())
	assert.ErrorIs(t, e, root)
}

func TestAs(t *testing.T) {
	e := errs.New(errs.CodeNotFound, "user not found")
	var got *errs.Error
	assert.True(t, errors.As(e, &got))
	assert.Equal(t, errs.CodeNotFound, got.Code)
}