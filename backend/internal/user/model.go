package user

import (
	"crypto/rsa"
	"hkorpo/book/internal/primitive"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	Firstname    string
	Lastname     string
	Role         primitive.UserRole
	Language     primitive.Language
	PasswordHash []byte
}

type ConfigJWT struct {
	PUBLIC_KEY_PATH               string        `envconfig:"PUBLIC_KEY_PATH"`
	PRIVATE_KEY_PATH              string        `envconfig:"PRIVATE_KEY_PATH"`
	PUBLIC_REFRESH_KEY_PATH       string        `envconfig:"PUBLIC_REFRESH_KEY_PATH"`
	PRIVATE_REFRESH_KEY_PATH      string        `envconfig:"PRIVATE_REFRESH_KEY_PATH"`
	JWT_TOKEN_EXP                 time.Duration `envconfig:"JWT_TOKEN_EXP"`
	JWT_REFRESH_TOKEN_EXP         time.Duration `envconfig:"JWT_REFRESH_TOKEN_EXP"`
	PublicKey, RefreshPublicKey   *rsa.PublicKey
	PrivateKey, RefreshPrivateKey *rsa.PrivateKey
}

type UserIDURI struct {
	ID uuid.UUID `uri:"id" validate:"required"`
}

type RegisterRequestDTO struct {
	Email     string `json:"email" validate:"required,email,max=200"`
	Firstname string `json:"firstname" validate:"required,max=100"`
	Lastname  string `json:"lastname" validate:"required,max=100"`
	Password  string `json:"password" validate:"required,min=12,max=100"`
}

type LoginRequestDTO struct {
	Email    string `json:"email" validate:"required,email,max=200"`
	Password string `json:"password" validate:"required,min=12,max=100"`
}

type UpdateUserRequestDTO struct {
	Firstname string             `json:"firstname" validate:"required,max=100"`
	Lastname  string             `json:"lastname" validate:"required,max=100"`
	Language  primitive.Language `json:"language" validate:"required,oneof=en fr"`
}

type UserTokenClaims struct {
	jwt.RegisteredClaims
	Role   primitive.UserRole `json:"role"`
	UserID uuid.UUID          `json:"userID"`
}
