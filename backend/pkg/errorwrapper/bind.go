package errorwrapper

import (
	"errors"
	"hkorpo/book/pkg/ent"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
)

func EntError(err error) (status int, msg string) {
	switch {
	case ent.IsConstraintError(err):
		return http.StatusConflict, http.StatusText(http.StatusConflict)

	case ent.IsNotFound(err):
		return http.StatusNotFound, http.StatusText(http.StatusNotFound)

	case ent.IsValidationError(err):
		return http.StatusBadRequest, http.StatusText(http.StatusBadRequest)
	}
	return 0, ""
}

func ValidateError(err error) (status int, msg string) {
	var errs validator.ValidationErrors
	if errors.As(err, &errs) {
		for _, e := range errs {
			msg += e.Error()
		}
		return http.StatusUnprocessableEntity, msg
	}
	return 0, ""
}

func MinioError(err error) (status int, msg string) {
	var resp minio.ErrorResponse
	if errors.As(err, &resp) && resp.Code == minio.NoSuchKey {
		return http.StatusNotFound, http.StatusText(http.StatusNotFound)
	}
	return 0, ""
}

func FiberError(err error) (status int, msg string) {
	var bindErr *fiber.BindError
	if errors.As(err, &bindErr) {
		return http.StatusUnprocessableEntity, bindErr.Error()
	}
	return 0, ""
}

// StatusOf reads the status/message attached via WithStatus, for errors that
// don't fit any of the categories above (e.g. a third-party API failure that
// should map to a specific status rather than the generic 500 fallback).
func StatusOf(err error) (status int, msg string) {
	var se *StatusError
	if errors.As(err, &se) {
		return se.status, se.Error()
	}
	return 0, ""
}
