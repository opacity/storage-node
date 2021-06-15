package utils

import (
	"github.com/go-playground/validator/v10"
)

// Use for validate any struct
var Validator *validator.Validate

func init() {
	Validator = validator.New()
}
