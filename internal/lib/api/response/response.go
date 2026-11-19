// Package response builds the JSON bodies shared by all handlers.
package response

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Response is the common part of every JSON response.
type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Values of Response.Status.
const (
	StatusOK    = "OK"
	StatusError = "Error"
)

// OK returns a success response.
func OK() Response {
	return Response{
		Status: StatusOK,
	}
}

// Error returns an error response with the given message.
func Error(msg string) Response {
	return Response{
		Status: StatusError,
		Error:  msg,
	}
}

// ValidationError turns validator errors into one readable error response.
func ValidationError(errs validator.ValidationErrors) Response {
	var errMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is a required field", err.Field()))
		case "http_url":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not a valid http(s) URL", err.Field()))
		case "alphanum":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s may only contain letters and digits", err.Field()))
		case "max":
			errMsgs = append(errMsgs, fmt.Sprintf("field %s must be at most %s characters long", err.Field(), err.Param()))
		default:
			errMsgs = append(errMsgs, fmt.Sprintf("field %s is not valid", err.Field()))
		}
	}

	return Response{
		Status: StatusError,
		Error:  strings.Join(errMsgs, ", "),
	}
}
