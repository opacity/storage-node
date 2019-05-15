package routes

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type testRequestObject struct {
	data string
}

type testVerifiedRequest struct {
	verification
	requestBody
	fooValue      string `form:"foo"`
	fileObject    string `formFile:"file"`
	requestObject testRequestObject
}

type testSetRequest struct {
	fooValue    string `form:"foo"`
	fileObject  string `formFile:"file"`
	randomValue string
}

func (v *testRequest) getObjectRef() interface{} {
	return &v.requestObject
}

func Test_verifyAndParseFormRequestWithVerifyRequest(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("signature", "signature")
	mw.WriteField("publicKey", "public")
	mw.WriteField("requestBody", "body")
	mw.WriteField("foo", "foo")
	w, err := mw.CreateFormFile("file", "test")
	w.Write([]byte("test"))

	c.Request, _ = http.NewRequest("POST", "/", body)

	request := testVerifiedRequest{}
	err := verifyAndParseFormRequest(&request, c)

	assert.Nil(t, err)
	assert.Equal(t, request.fooValue, "foo")
	assert.Equal(t, request.requestObject.data, "body")
}

func Test_verifyAndParseFormRequestWithNormalRequest(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("foo", "foo")
	w, err := mw.CreateFormFile("file", "test")
	w.Write([]byte("test"))

	c.Request, _ = http.NewRequest("POST", "/", body)

	request := testSetRequest{}
	err := verifyAndParseFormRequest(&request, c)

	assert.Nil(t, err)
	assert.Equal(t, request.fooValue, "foo")
	assert.Equal(t, request.fileObject, "test")
	assert.Equal(t, request.randomValue, "")
}
