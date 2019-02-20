package utils

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

/*ParseRequestBody take a request and parses the body to the target interface.*/
func ParseRequestBody(req *http.Request, dest interface{}) error {
	body := req.Body
	defer body.Close()

	return parse(body, dest)
}

/*ParseResponseBody take a response and parses the body to the target interface.*/
func ParseResponseBody(res *http.Response, dest interface{}) error {
	body := res.Body
	defer body.Close()

	return parse(body, dest)
}

func parse(body io.ReadCloser, dest interface{}) error {
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(bodyBytes, dest)
}
