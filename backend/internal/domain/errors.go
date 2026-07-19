// Package domain holds framework-free entities, rules and error kinds shared by
// services. It must not import gin, pgx, or any transport/storage detail.
package domain

import "errors"

// Error kinds. Handlers map these to HTTP status codes via errors.Is.
var (
	ErrValidation         = errors.New("validation error")
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
)

// Error carries a human message alongside a kind so callers can both classify
// (errors.Is) and display (Message).
type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Kind }

func Validation(msg string) error { return &Error{Kind: ErrValidation, Message: msg} }
func NotFound(msg string) error   { return &Error{Kind: ErrNotFound, Message: msg} }
func Conflict(msg string) error   { return &Error{Kind: ErrConflict, Message: msg} }
func Forbidden(msg string) error  { return &Error{Kind: ErrForbidden, Message: msg} }
