package utils

import (
	"net/http"
	"users-api/internal/dto"

	"github.com/gin-gonic/gin"
)

func SendSuccessResponse(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, dto.NewSuccessResponse(message, data))
}

func SendErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, dto.NewErrorResponse(message))
}

func SendErrorResponseWithCode(c *gin.Context, status int, message string, code int) {
	c.JSON(status, dto.NewErrorResponseWithCode(message, code))
}

func SendValidationError(c *gin.Context, err error) {
	SendErrorResponse(c, http.StatusBadRequest, "Validation failed: "+err.Error())
}

func SendInternalError(c *gin.Context, err error) {
	SendErrorResponse(c, http.StatusInternalServerError, "Internal server error: "+err.Error())
}

func SendNotFoundError(c *gin.Context, resource string) {
	SendErrorResponse(c, http.StatusNotFound, resource+" not found")
}

func SendUnauthorizedError(c *gin.Context, message string) {
	SendErrorResponse(c, http.StatusUnauthorized, message)
}

func SendForbiddenError(c *gin.Context, message string) {
	SendErrorResponse(c, http.StatusForbidden, message)
}

func SendConflictError(c *gin.Context, message string) {
	SendErrorResponse(c, http.StatusConflict, message)
}

func SendTooManyRequestsError(c *gin.Context, message string) {
	SendErrorResponse(c, http.StatusTooManyRequests, message)
}