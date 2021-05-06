package utils

import (
	"testing"
)

/*AssertTrue fails if v is false.*/
func AssertTrue(v bool, t *testing.T, desc string) {
	if !v {
		t.Error(desc)
	}
}
