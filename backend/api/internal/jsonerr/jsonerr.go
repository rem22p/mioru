// Package jsonerr provides small helpers for writing JSON error responses with
// a machine-readable "code" field. It is imported by both handler and middleware
// packages so that every error response — including authz/CSRF gates — follows
// the same envelope contract.
package jsonerr

import (
	"encoding/json"
	"net/http"
)

// Error writes a JSON error envelope {error, code} where code is derived from
// the HTTP status. Prefer ErrorCode when the status alone is ambiguous.
func Error(w http.ResponseWriter, msg string, httpStatus int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
		"code":  httpCodeToCode(httpStatus),
	})
}

// ErrorCode writes a JSON error envelope with an explicit machine-readable code.
// Use this when the HTTP status is ambiguous (e.g. 401 can mean missing session
// vs expired/invalid token) and the SPA needs to branch by code.
func ErrorCode(w http.ResponseWriter, msg string, httpStatus int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
		"code":  code,
	})
}

func httpCodeToCode(status int) string {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "VALIDATION_FAILED"
	case http.StatusUnauthorized:
		return "AUTH_REQUIRED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusTooManyRequests:
		return "RATE_LIMITED"
	case http.StatusRequestEntityTooLarge:
		return "PAYLOAD_TOO_LARGE"
	default:
		return "INTERNAL"
	}
}
