package utils

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	SetTesting("../.env")
	os.Exit(m.Run())
}
