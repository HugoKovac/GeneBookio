package user

import (
	"context"
	"crypto/rsa"
	"hkorpo/book/pkg/errorwrapper"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/matthewhartstonge/argon2"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) (*User, error)
}

type Service struct {
	repo      Repository
	configJWT *ConfigJWT
}

func NewService(r Repository, configJWT *ConfigJWT) *Service {
	return &Service{
		repo:      r,
		configJWT: configJWT,
	}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *Service) Create(ctx context.Context, user *User) (*User, error) {
	return s.repo.Create(ctx, user)
}

func (s *Service) GenerateToken(ctx context.Context, user *User, privateKey *rsa.PrivateKey, exp time.Duration) (string, error) {
	var (
		currentTime    = time.Now()
		expirationTime = currentTime.Add(exp)
	)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, UserTokenClaims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(currentTime),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	})

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", errorwrapper.Wrap(err)
	}

	return signedToken, nil

}

func (s *Service) HashPassword(password string) ([]byte, error) {
	cfg := argon2.MemoryConstrainedDefaults()

	raw, err := cfg.Hash([]byte(password), nil)
	if err != nil {
		return []byte{}, errorwrapper.Wrap(err)
	}
	// Encode the raw secret byte
	encoded := raw.Encode()

	return encoded, nil
}

func (s *Service) ValidatePasswordHash(storedPassword, suppliedPassword []byte) (bool, error) {
	ok, err := argon2.VerifyEncoded(suppliedPassword, storedPassword)
	return ok, errorwrapper.Wrap(err)
}
