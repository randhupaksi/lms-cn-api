package request

import (
	"net/http"

	"lms-cn-api/pkg/apperror"

	"github.com/gin-gonic/gin"
)

func BindJSON(c *gin.Context, target any) error {
	if err := c.ShouldBindJSON(target); err != nil {
		return apperror.New(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Data yang dikirim belum valid")
	}
	return nil
}

func RequireID(c *gin.Context, name string) (string, error) {
	value := c.Param(name)
	if value == "" {
		return "", apperror.New(http.StatusBadRequest, "INVALID_RESOURCE_ID", "ID resource tidak valid")
	}
	return value, nil
}
