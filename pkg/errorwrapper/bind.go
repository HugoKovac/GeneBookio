package errorwrapper

import (
	"hkorpo/book/pkg/ent"
	"net/http"
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
