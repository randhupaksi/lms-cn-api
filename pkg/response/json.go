package response

import "github.com/gin-gonic/gin"

type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func Success(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Envelope{Success: true, Message: message, Data: data})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, Envelope{Success: false, Message: message, Data: nil})
}
