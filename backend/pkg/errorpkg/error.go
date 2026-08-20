package errorpkg

import (
	"errors"
	"fmt"
	"hkorpo/book/pkg/errorwrapper"
	"os"
)

func DeepestErrorWrapper(err error) errorwrapper.IErrorWrapper {
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

func PrintTrace(err error) {
	errwrap := DeepestErrorWrapper(err)
	if errwrap != nil {
		fmt.Println(errwrap.FormatTrace())
	} else {
		fmt.Println(err)
	}
}

func ExitTrace(err error) {
	if err != nil {
		PrintTrace(err)
		os.Exit(1)
	}
}
