package utils

import (
	"testing"

	"strings"

	"github.com/stretchr/testify/assert"
)

func Test_AppendLookupErrors(t *testing.T) {
	var errorsArray []error

	property1 := "NOT_A_REAL_PROPERTY"
	property2 := "ALSO_NOT_A_REAL_PROPERTY"

	AppendLookupErrors(property1, &errorsArray)
	AppendLookupErrors(property2, &errorsArray)

	collectedErrors := CollectErrors(errorsArray)

	assert.True(t, strings.Contains(collectedErrors.Error(), property1))
	assert.True(t, strings.Contains(collectedErrors.Error(), property2))
}

func Test_SetTesting(t *testing.T) {
	assert.True(t, Env.AccountRetentionDays > 0)
}
