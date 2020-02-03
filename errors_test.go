package gotham

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	baseError := errors.New("test error")
	err := &Error{
		Err:  baseError,
		Type: ErrorTypePrivate,
	}
	assert.Equal(t, err.Error(), baseError.Error())

	assert.Equal(t, err.SetType(ErrorTypePublic), err)
	assert.Equal(t, ErrorTypePublic, err.Type)

	assert.Equal(t, err.SetMeta("some data"), err)
	assert.Equal(t, "some data", err.Meta)

	type customError struct {
		status string
		data   string
	}
	err.SetMeta(customError{status: "200", data: "other data"}) // nolint: errcheck
	assert.Equal(t, customError{status: "200", data: "other data"}, err.Meta)
}

func TestErrorSlice(t *testing.T) {
	errs := errorMsgs{
		{Err: errors.New("first"), Type: ErrorTypePrivate},
		{Err: errors.New("second"), Type: ErrorTypePrivate, Meta: "some data"},
		{Err: errors.New("third"), Type: ErrorTypePublic, Meta: "some other data"},
	}

	assert.Equal(t, errs, errs.ByType(ErrorTypeAny))
	assert.Equal(t, "third", errs.Last().Error())
	assert.Equal(t, []string{"first", "second", "third"}, errs.Errors())
	assert.Equal(t, []string{"third"}, errs.ByType(ErrorTypePublic).Errors())
	assert.Equal(t, []string{"first", "second"}, errs.ByType(ErrorTypePrivate).Errors())
	assert.Equal(t, []string{"first", "second", "third"}, errs.ByType(ErrorTypePublic|ErrorTypePrivate).Errors())
	assert.Equal(t, `Error #01: first
Error #02: second
Meta: some data
Error #03: third
Meta: some other data
`, errs.String())

	errs = errorMsgs{}
	assert.Nil(t, errs.Last())
	assert.Empty(t, errs.String())
}
