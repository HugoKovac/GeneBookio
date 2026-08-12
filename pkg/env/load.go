package env

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	if os.Getenv("ENV_STATE") != "" {
		return
	}
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		panic("Unable to get file path")
	}
	dir := filepath.Dir(filename)
	if err := godotenv.Load(dir + "/.env"); err != nil {
		panic("load environment")
	}
}
