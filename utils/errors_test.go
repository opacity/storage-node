package utils

import (
	"errors"
	"testing"

	"strings"

	"github.com/stretchr/testify/assert"
)

func Test_ReturnFirstError(t *testing.T) {
	error1 := errors.New("error1")
	error2 := errors.New("error2")
	firstError := ReturnFirstError([]error{error1, error2})
	assert.Equal(t, error1, firstError)
}

func Test_CollectErrors(t *testing.T) {
	error1 := errors.New("error1")
	error2 := errors.New("error2")
	err := CollectErrors([]error{error1, error2})
	assert.Equal(t, true, strings.Contains(err.Error(), error1.Error()))
	assert.Equal(t, true, strings.Contains(err.Error(), error2.Error()))
}

func Test_AppendIfError(t *testing.T) {
	var collectedErrors []error
	var nilError error
	nonNilError := errors.New("error1")
	AppendIfError(nilError, &collectedErrors)
	AppendIfError(nonNilError, &collectedErrors)
	AppendIfError(nilError, &collectedErrors)
	AppendIfError(nonNilError, &collectedErrors)
	assert.Equal(t, 2, len(collectedErrors))
}
