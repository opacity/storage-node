package utils

import (
	"gopkg.in/go-playground/validator.v8"
)

// Use for validate any struct
var Validator *validator.Validate

func init() {
	config := &validator.Config{TagName: "binding"}
	Validator = validator.New(config)
}
