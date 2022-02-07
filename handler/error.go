package handler

import (
	"fmt"

	"github.com/pkg/errors"
)

// Error is the handler package's error type. Is not meant to be compared as
// a type, but information should be extracted via the interfaces
// it implements with callback functions. Is not guaranteed to remain exported
// so shouldn't be treated as such.
type Error struct {
	err     error
	logData map[string]interface{}
}

// NewError creates a new Error
func NewError(err error, logData map[string]interface{}) *Error {
	return &Error{
		err:     err,
		logData: logData,
	}
}

// Error implements the Go standard error interface
func (e *Error) Error() string {
	if e.err == nil {
		return "nil"
	}
	return e.err.Error()
}

// LogData implements the DataLogger interface which allows you extract
// embedded log.Data from an error
func (e *Error) LogData() map[string]interface{} {
	return e.logData
}

func (e *Error) Unwrap() error {
	return e.err
}

// ErrorStack wraps the mesage with a stack trace
func ErrorStack(message string) error {
	return errors.Wrap(fmt.Errorf(message), "")
}
