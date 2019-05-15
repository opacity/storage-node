package routes

import (
	"bytes"
	"mime/multipart"
	"net/http"
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
	strValue      string `form:"str"`
	fileObject    string `formFile:"file"`
	requestObject testRequestObject
}

type testSetRequest struct {
	strValue   string `form:"str"`
	fileObject string `formFile:"file"`
	emptyValue string
}

func (v *testVerifiedRequest) getObjectRef() interface{} {
	return &v.requestObject
}

func Test_verifyAndParseFormRequestWithVerifyRequest(t *testing.T) {
	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("signature", "signature")
	mw.WriteField("publicKey", "public")
	mw.WriteField("requestBody", "body")
	mw.WriteField("str", "strV")
	w, _ := mw.CreateFormFile("file", "test")
	w.Write([]byte("test"))
	mw.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", body)

	request := testVerifiedRequest{}
	err := verifyAndParseFormRequest(&request, c)

	assert.Nil(t, err)
	assert.Equal(t, request.strValue, "strV")
	assert.Equal(t, request.requestObject.data, "body")
}

func Test_verifyAndParseFormRequestWithNormalRequest(t *testing.T) {
	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("str", "strV")
	w, _ := mw.CreateFormFile("file", "test")
	w.Write([]byte("test"))
	mw.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", body)

	request := testSetRequest{}
	err := verifyAndParseFormRequest(&request, c)

	assert.Nil(t, err)
	assert.Equal(t, request.strValue, "strV")
	assert.Equal(t, request.fileObject, "test")
	assert.Equal(t, request.emptyValue, "")
}
