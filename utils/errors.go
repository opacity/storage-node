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
		SlackLogError(err.Error())
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

/*AppendIfError adds a non-nil error to an array of errors*/
func AppendIfError(err error, collectedErrors *[]error) {
	if err != nil {
		*collectedErrors = append(*(collectedErrors),
			err)
	}
}

/*LogIfError logs any error if it is not nil. Allow caller to provide additional freeform info.*/
func LogIfError(err error, extraInfo map[string]interface{}) {
	if err == nil {
		return
	}
	SlackLogError(err.Error())
	fmt.Println(err)
	fmt.Println(extraInfo)
}
