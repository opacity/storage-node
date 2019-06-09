package routes

import (
	"testing"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

)

func Test_Init_Upload_Status(t *testing.T) {
	setupTests(t)
	cleanUpBeforeTest(t)
}

func Test_CheckWithAccountNoExist(t *testing.T) {
}

func Test_CheckFileNotCompleted(t *testing.T) {
}

func Test_CheckFileIsCompleted(t *testing.T) {
}

func Test_MissingIndexes(t *testing.T) {
}