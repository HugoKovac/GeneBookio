package user

import (
	"context"
	"crypto/rsa"
	"errors"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/errorwrapper"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/matthewhartstonge/argon2"
)

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

func (s *Service) List(ctx context.Context) ([]*User, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, user *User) (*User, error) {
	return s.repo.Create(ctx, user)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, firstname, lastname string, language primitive.Language) (*User, error) {
	return s.repo.Update(ctx, id, firstname, lastname, language)
}

func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id)
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

// ParseToken verifies tokenString against publicKey and returns its claims.
// Callers pass configJWT.PublicKey for an access token or
// configJWT.RefreshPublicKey for a refresh token — the two are signed with
// different keypairs so one can't be presented as the other.
func (s *Service) ParseToken(tokenString string, publicKey *rsa.PublicKey) (*UserTokenClaims, error) {
	var claims UserTokenClaims

	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return publicKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errorwrapper.Wrap(errors.New("invalid token"))
	}

	return &claims, nil
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
