package config

import (
	"hkorpo/book/internal/platform/database"
	"hkorpo/book/internal/user"
)

type Config struct {
	database.ConfigDB
	user.ConfigJWT
}
