package domain

import "errors"

// Общие ошибки домена, по ним хендлеры выбирают HTTP-код.
var (
	ErrNotFound      = errors.New("resource not found")        // 404
	ErrAlreadyExists = errors.New("resource already exists")   // 409
	ErrUnauthorized  = errors.New("unauthorized")              // 401
	ErrForbidden     = errors.New("forbidden")                 // 403
	ErrValidation    = errors.New("validation failed")         // 400
	ErrConflict      = errors.New("concurrent state conflict") // 409
	ErrInternal      = errors.New("internal server error")     // 500
)
