package dto

import "users-api/internal/models"

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    int    `json:"code,omitempty"`
}

func NewSuccessResponse(message string, data interface{}) APIResponse {
	return APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func NewErrorResponse(message string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error:   message,
	}
}

func NewErrorResponseWithCode(message string, code int) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error:   message,
		Code:    code,
	}
}

func ToLoginResponse(auth *models.AuthResponse) LoginResponse {
	return LoginResponse{
		User:         ToUserResponse(auth.User),
		AccessToken:  auth.AccessToken,
		RefreshToken: auth.RefreshToken,
		ExpiresIn:    auth.ExpiresIn,
	}
}