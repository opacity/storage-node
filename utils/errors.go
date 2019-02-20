package utils

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

/*PanicOnError panics if an error was passed in*/
func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

/*ReturnFirstError accepts an array of errors and returns the first that is not nil*/
func ReturnFirstError(arrayOfErrs []error) error {
	for _, errInArray := range arrayOfErrs {
		if errInArray != nil {
			return errInArray
		}
	}
	return nil
}

/*CollectErrors returns all the errors as one error*/
func CollectErrors(arrayOfErrs []error) error {
	if len(arrayOfErrs) == 0 {
		return nil
	}
	var buffer bytes.Buffer
	var errString string
	i := 0
	for _, errInArray := range arrayOfErrs {
		if errInArray != nil {
			buffer.WriteString("Error ")
			buffer.WriteString(strconv.Itoa(i + 1))
			buffer.WriteString(": ")
			buffer.WriteString(errInArray.Error())
			buffer.WriteString("\n")
			i++
		}
	}
	errString = buffer.String()
	if errString == "" {
		return nil
	}
	return errors.New(errString)
}

/*LogIfError logs any error if it is not nil. Allow caller to provide additional freeform info.*/
func LogIfError(err error) {
	if err == nil {
		return
	}

	// TODO: proper external logging

	fmt.Println(err)
}
