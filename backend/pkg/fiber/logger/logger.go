package logger

import (
	"fmt"
	"hkorpo/book/pkg/errorpkg"
	"hkorpo/book/pkg/errorwrapper"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func ErrorTraceLoggerTags(output logger.Buffer, c fiber.Ctx, data *logger.Data, _ string) (int, error) {
	st := errorpkg.DeepestErrorWrapper(data.ChainErr)
	if st != nil {
		status, msg := errorwrapper.ValidateError(data.ChainErr)
		if status == 0 {
			status, msg = errorwrapper.EntError(data.ChainErr)
		}
		if status == 0 {
			status, msg = errorwrapper.MinioError(data.ChainErr)
		}
		if status == 0 {
			status, msg = errorwrapper.FiberError(data.ChainErr)
		}
		if status == 0 {
			status, msg = errorwrapper.StatusOf(data.ChainErr)
		}
		if status == 0 {
			// Any wrapped error not covered above still needs a real
			// response — otherwise it silently falls through to a
			// misleading 200 with an empty body while the trace below
			// still logs correctly.
			status, msg = http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
		}

		c.Status(status).JSON(msg)

		return output.WriteString(fmt.Sprintf("%+v", st.FormatTrace()))
	}
	return 0, nil
}
