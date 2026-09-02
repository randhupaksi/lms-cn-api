package response

import (
	"net/http"

	"lms-cn-api/pkg/apperror"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Success bool                `json:"success"`
	Message string              `json:"message"`
	Code    string              `json:"code,omitempty"`
	Data    any                 `json:"data,omitempty"`
	Errors  map[string][]string `json:"errors,omitempty"`
	Meta    any                 `json:"meta,omitempty"`
}

func Success(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Envelope{Success: true, Message: message, Data: data})
}

func SuccessWithMeta(c *gin.Context, status int, message string, data, meta any) {
	c.JSON(status, Envelope{Success: true, Message: message, Data: data, Meta: meta})
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, Envelope{Success: false, Message: message, Code: code})
}

func FromError(c *gin.Context, err error) {
	if appErr, ok := apperror.As(err); ok {
		c.JSON(appErr.Status, Envelope{
			Success: false,
			Message: appErr.Message,
			Code:    appErr.Code,
			Errors:  appErr.Fields,
		})
		return
	}
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan pada server")
}
