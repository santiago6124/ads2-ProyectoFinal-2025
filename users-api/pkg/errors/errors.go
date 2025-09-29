package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code int, message string, details ...string) *AppError {
	var detail string
	if len(details) > 0 {
		detail = details[0]
	}
	return &AppError{
		Code:    code,
		Message: message,
		Details: detail,
	}
}

func NewValidationError(message string, details ...string) *AppError {
	return NewAppError(http.StatusBadRequest, message, details...)
}

func NewNotFoundError(resource string) *AppError {
	return NewAppError(http.StatusNotFound, fmt.Sprintf("%s not found", resource))
}

func NewUnauthorizedError(message string) *AppError {
	return NewAppError(http.StatusUnauthorized, message)
}

func NewForbiddenError(message string) *AppError {
	return NewAppError(http.StatusForbidden, message)
}

func NewConflictError(message string) *AppError {
	return NewAppError(http.StatusConflict, message)
}

func NewInternalError(message string, details ...string) *AppError {
	return NewAppError(http.StatusInternalServerError, message, details...)
}

func NewTooManyRequestsError(message string) *AppError {
	return NewAppError(http.StatusTooManyRequests, message)
}

var (
	ErrUserNotFound         = NewNotFoundError("User")
	ErrInvalidCredentials   = NewUnauthorizedError("Invalid credentials")
	ErrTokenExpired         = NewUnauthorizedError("Token expired")
	ErrTokenInvalid         = NewUnauthorizedError("Invalid token")
	ErrAccessDenied         = NewForbiddenError("Access denied")
	ErrUserAlreadyExists    = NewConflictError("User already exists")
	ErrInvalidPassword      = NewValidationError("Invalid password format")
	ErrInvalidEmail         = NewValidationError("Invalid email format")
	ErrAccountDeactivated   = NewUnauthorizedError("Account is deactivated")
	ErrTooManyAttempts      = NewTooManyRequestsError("Too many failed attempts")
)