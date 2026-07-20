package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Firstname string
	Lastname  string
}

type UserIDURI struct {
	ID string `uri:"id" validator:"required,uuid"`
}
