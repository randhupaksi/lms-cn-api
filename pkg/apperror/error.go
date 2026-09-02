package apperror

import "errors"

type Error struct {
	Code    string
	Message string
	Status  int
	Fields  map[string][]string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func WithFields(status int, code, message string, fields map[string][]string) *Error {
	return &Error{Status: status, Code: code, Message: message, Fields: fields}
}

func Wrap(status int, code, message string, cause error) *Error {
	return &Error{Status: status, Code: code, Message: message, Cause: cause}
}

func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
