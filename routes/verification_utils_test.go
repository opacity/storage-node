package routes

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type testRequestObject struct {
	data string `json:"data"`
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

func (v *testVerifiedRequest) getObjectRef() interface{} {
	return &v.requestObject
}

func Test_verifyAndParseFormRequestWithVerifyRequest(t *testing.T) {
	obj := testRequestObject{
		data: "some body message",
	}
	reqJSON, _ := json.Marshal(obj)
	reqBody := bytes.NewBuffer(reqJSON)
	fmt.Printf("RequestBody %v\n", reqBody)
	verification := returnSuccessVerificationForTest(t, reqBody.String())
	body := new(bytes.Buffer)
	mw := multipart.NewWriter(body)
	mw.WriteField("signature", verification.Signature)
	mw.WriteField("publicKey", verification.PublicKey)
	mw.WriteField("requestBody", reqBody.String())
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
	assert.Equal(t, obj.data, request.requestObject.data)
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
