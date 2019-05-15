package routes

import (
	"testing"
)

type testRequestObject struct {
	parseData string
}

type testRequest struct {
	verification
	requestBody
	fooObject     string `form:"foo"`
	requestObject testRequestObject
}

func (v *testRequest) getObjectRef() interface{} {
	return &v.requestObject
}

func Test_verifyAndParseFormRequest(t *testing.T) {

}
