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
	ID uuid.UUID `uri:"id" validate:"required"`
}

type RegisterRequestDTO struct {
	Email     string `json:"email" validate:"required,email,max=200"`
	Firstname string `json:"firstname" validate:"required,max=100"`
	Lastname  string `json:"lastname" validate:"required,max=100"`
}
