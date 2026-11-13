package request

import (
	"errors"
	"io"
	"net/http"

	"lms-cn-api/pkg/apperror"

	"github.com/gin-gonic/gin"
)

const maxJSONBodyBytes = 4 << 20

func BindJSON(c *gin.Context, target any) error {
	if c.Request.ContentLength > maxJSONBodyBytes {
		return requestBodyTooLarge()
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)
	if err := c.ShouldBindJSON(target); err != nil {
		if isRequestBodyTooLarge(err) {
			return requestBodyTooLarge()
		}
		return apperror.New(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Data yang dikirim belum valid")
	}
	// Drain after decoding so chunked requests with a valid JSON value followed
	// by a large body are subject to the same limit as Content-Length requests.
	if _, err := io.Copy(io.Discard, c.Request.Body); err != nil {
		if isRequestBodyTooLarge(err) {
			return requestBodyTooLarge()
		}
		return apperror.New(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Data yang dikirim belum valid")
	}
	return nil
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func requestBodyTooLarge() error {
	return apperror.New(http.StatusRequestEntityTooLarge, "REQUEST_BODY_TOO_LARGE", "Ukuran data yang dikirim melebihi batas")
}

func RequireID(c *gin.Context, name string) (string, error) {
	value := c.Param(name)
	if value == "" {
		return "", apperror.New(http.StatusBadRequest, "INVALID_RESOURCE_ID", "ID resource tidak valid")
	}
	return value, nil
}
