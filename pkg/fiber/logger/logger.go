package logger

import (
	"errors"
	"fmt"
	"hkorpo/book/pkg/errorwrapper"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func deepestErrorWrapper(err error) errorwrapper.IErrorWrapper {
	var deepest errorwrapper.IErrorWrapper

	for err != nil {
		var st errorwrapper.IErrorWrapper
		if !errors.As(err, &st) {
			break
		}

		deepest = st

		wrapper, ok := st.(interface{ Unwrap() error })
		if !ok {
			break
		}

		err = wrapper.Unwrap()
	}

	return deepest
}

func ErrorTraceLoggerTags(output logger.Buffer, c fiber.Ctx, data *logger.Data, _ string) (int, error) {
	st := deepestErrorWrapper(data.ChainErr)
	if st != nil {
		status, msg := errorwrapper.EntError(data.ChainErr)

		c.Status(status).JSON(msg)

		return output.WriteString(fmt.Sprintf("%+v", st.FormatTrace()))
	}
	return 0, nil
}
