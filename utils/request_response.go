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

	if err := parse(body, dest); err != nil {
		return err
	}

	return Validator.Struct(dest)
}

/*ParseStringifiedRequest takes a stringified request and parses the body to the target interface.*/
func ParseStringifiedRequest(body string, dest interface{}) error {
	if err := json.Unmarshal([]byte(body), dest); err != nil {
		return err
	}

	return Validator.Struct(dest)
}

/*ParseResponseBody take a response and parses the body to the target interface.*/
func ParseResponseBody(res *http.Response, dest interface{}) error {
	body := res.Body
	defer body.Close()

	if err := parse(body, dest); err != nil {
		return err
	}
	return Validator.Struct(dest)
}

func parse(body io.ReadCloser, dest interface{}) error {
	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(bodyBytes, dest)
}
