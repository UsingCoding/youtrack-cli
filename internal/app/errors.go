package app

import (
	"errors"
	"fmt"
)

type ErrorKind int

const (
	ErrorRuntime ErrorKind = iota
	ErrorValidation
	ErrorAuth
	ErrorNotFound
	ErrorAmbiguous
)

type Error struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "application error"
}

func (e *Error) Unwrap() error { return e.Err }

func errorf(kind ErrorKind, format string, args ...any) error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

func Validationf(format string, args ...any) error { return errorf(ErrorValidation, format, args...) }
func Authf(format string, args ...any) error       { return errorf(ErrorAuth, format, args...) }
func NotFoundf(format string, args ...any) error   { return errorf(ErrorNotFound, format, args...) }
func Ambiguousf(format string, args ...any) error  { return errorf(ErrorAmbiguous, format, args...) }
func Runtimef(format string, args ...any) error    { return errorf(ErrorRuntime, format, args...) }

func KindOf(err error) ErrorKind {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Kind
	}
	return ErrorRuntime
}
