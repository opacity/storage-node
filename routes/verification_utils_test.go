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
	Data string `json:"data"`
}

type testVerifiedRequest struct {
	verification
	requestBody
	StrValue      string `form:"str"`
	FileObject    string `formFile:"file"`
	requestObject testRequestObject
}

type testSetRequest struct {
	StrValue   string `form:"str"`
	FileObject string `formFile:"file"`
	emptyValue string
}

type testRequireForm struct {
	Object string `formFile:"file"  validate:"required,len=123"`
}

type testNonRequireForm struct {
	Object string `formFile:"file"`
}

func (v *testVerifiedRequest) getObjectRef() interface{} {
	return &v.requestObject
}

func Test_verifyAndParseFormRequestWithVerifyRequest(t *testing.T) {
	obj := testRequestObject{
		Data: "some body message",
	}
	v, b, _ := returnValidVerificationAndRequestBodyWithRandomPrivateKey(t, obj)
	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("signature", v.Signature)
	mw.WriteField("publicKey", v.PublicKey)
	mw.WriteField("requestBody", b.RequestBody)
	mw.WriteField("str", "strV")
	w, _ := mw.CreateFormFile("file", "test")
	w.Write([]byte("test"))
	mw.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())

	request := testVerifiedRequest{}
	err := verifyAndParseFormRequest(&request, c)

	assert.Nil(t, err)
	assert.Equal(t, "strV", request.StrValue)
	assert.Equal(t, "test", request.FileObject)
	assert.Equal(t, obj.Data, request.requestObject.Data)
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
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())

	request := testSetRequest{}
	err := verifyAndParseFormRequest(&request, c)

	assert.Nil(t, err)
	assert.Equal(t, "strV", request.StrValue)
	assert.Equal(t, "test", request.FileObject)
	assert.Equal(t, "", request.emptyValue)
}

func Test_RequiredFileFormExist(t *testing.T) {
	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	w, _ := mw.CreateFormFile("file", "test")
	w.Write([]byte("test"))
	mw.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())

	request := testRequireForm{}
	err := verifyAndParseFormRequest(&request, c)

	assert.Nil(t, err)
	assert.Equal(t, "test", request.Object)
}

func Test_RequiredFileFormNoExist(t *testing.T) {
	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("str", "strV")
	mw.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())

	request := testRequireForm{}
	err := verifyAndParseFormRequest(&request, c)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unable to get file")
}

func Test_NoRequiredFileFormNoExist(t *testing.T) {
	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("str", "strV")
	mw.Close()

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", body)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())

	request := testNonRequireForm{}
	err := verifyAndParseFormRequest(&request, c)

	assert.Nil(t, err)
}
