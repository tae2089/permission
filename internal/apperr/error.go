// Package apperr defines application-owned error meanings.
package apperr

import "github.com/tae2089/trace/v3"

type Kind string

const (
	KindBadParameter    Kind = "bad_parameter"
	KindUnauthenticated Kind = "unauthenticated"
	KindAccessDenied    Kind = "access_denied"
	KindNotFound        Kind = "not_found"
	KindAlreadyExists   Kind = "already_exists"
	KindConflict        Kind = "conflict"
	KindLimitExceeded   Kind = "limit_exceeded"
	KindCanceled        Kind = "canceled"
	KindNotImplemented  Kind = "not_implemented"
	KindUnavailable     Kind = "unavailable"
	KindTimeout         Kind = "timeout"
)

type Error struct {
	Kind    Kind
	Message string
	cause   error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.cause
}

func New(kind Kind, message string) error {
	return trace.Errorf("%w", &Error{Kind: kind, Message: message})
}

func Wrap(kind Kind, cause error, message string) error {
	return trace.Errorf("%w", &Error{
		Kind:    kind,
		Message: message,
		cause:   cause,
	})
}
