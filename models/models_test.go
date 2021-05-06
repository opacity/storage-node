package models

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setupModelsTests()
	os.Exit(m.Run())
}
