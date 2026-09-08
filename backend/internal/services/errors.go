package services

import "errors"

type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

func newError(code, message string, err error) error {
	return &Error{Code: code, Message: message, Err: err}
}

var errInvalidCredentials = errors.New("invalid credentials")
