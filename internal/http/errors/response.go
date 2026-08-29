package errors

import (
	stderrors "errors"
	"net/http"

	"github.com/tae2089/go-template/internal/apperr"
)

type Code string

const (
	CodeBadRequest      Code = "bad_request"
	CodeUnauthenticated Code = "unauthenticated"
	CodeAccessDenied    Code = "access_denied"
	CodeNotFound        Code = "not_found"
	CodeAlreadyExists   Code = "already_exists"
	CodeConflict        Code = "conflict"
	CodeLimitExceeded   Code = "limit_exceeded"
	CodeCanceled        Code = "canceled"
	CodeNotImplemented  Code = "not_implemented"
	CodeUnavailable     Code = "unavailable"
	CodeTimeout         Code = "timeout"
	CodeInternal        Code = "internal"
)

type Response struct {
	Error Body `json:"error"`
}

type Body struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id"`
}

func Resolve(err error, traceID string) (int, Response) {
	status, code, message := classifyError(err)
	return status, Response{
		Error: Body{
			Code:    code,
			Message: message,
			TraceID: traceID,
		},
	}
}

func classifyError(err error) (int, Code, string) {
	var target *apperr.Error
	if !stderrors.As(err, &target) {
		return http.StatusInternalServerError, CodeInternal, "internal server error"
	}

	switch target.Kind {
	case apperr.KindBadParameter:
		return http.StatusBadRequest, CodeBadRequest, publicMessage(target.Message, "bad request")
	case apperr.KindUnauthenticated:
		return http.StatusUnauthorized, CodeUnauthenticated, "authentication required"
	case apperr.KindAccessDenied:
		return http.StatusForbidden, CodeAccessDenied, "access denied"
	case apperr.KindNotFound:
		return http.StatusNotFound, CodeNotFound, publicMessage(target.Message, "not found")
	case apperr.KindAlreadyExists:
		return http.StatusConflict, CodeAlreadyExists, publicMessage(target.Message, "already exists")
	case apperr.KindConflict:
		return http.StatusConflict, CodeConflict, publicMessage(target.Message, "conflict")
	case apperr.KindLimitExceeded:
		return http.StatusTooManyRequests, CodeLimitExceeded, publicMessage(target.Message, "limit exceeded")
	case apperr.KindCanceled:
		return 499, CodeCanceled, "request canceled"
	case apperr.KindNotImplemented:
		return http.StatusNotImplemented, CodeNotImplemented, "internal server error"
	case apperr.KindUnavailable:
		return http.StatusServiceUnavailable, CodeUnavailable, "internal server error"
	case apperr.KindTimeout:
		return http.StatusGatewayTimeout, CodeTimeout, "internal server error"
	default:
		return http.StatusInternalServerError, CodeInternal, "internal server error"
	}
}

func publicMessage(message, fallback string) string {
	if message == "" {
		return fallback
	}
	return message
}
