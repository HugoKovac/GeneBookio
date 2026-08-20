package user

import (
	"context"
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/ent"
	"time"

	"hkorpo/book/pkg/ent/user"

	"hkorpo/book/pkg/errorwrapper"

	"github.com/google/uuid"
)

type Repository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Create(ctx context.Context, user *User) (*User, error)
	Update(ctx context.Context, id uuid.UUID, firstname, lastname string, language primitive.Language) (*User, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

type RepositoryImpl struct {
	dbClient *ent.Client
}

func NewRepositoryImpl(dbClient *ent.Client) *RepositoryImpl {
	return &RepositoryImpl{
		dbClient: dbClient,
	}
}

func (r *RepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	e, err := r.dbClient.User.Query().Where(
		user.ID(id),
		user.DeletedAtIsNil(),
	).First(ctx)

	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	return toDomainUser(e), nil
}

func (r *RepositoryImpl) GetByEmail(ctx context.Context, email string) (*User, error) {
	e, err := r.dbClient.User.Query().Where(
		user.EmailEQ(email),
		user.DeletedAtIsNil(),
	).First(ctx)

	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	return toDomainUser(e), nil
}

func (r *RepositoryImpl) List(ctx context.Context) ([]*User, error) {
	entities, err := r.dbClient.User.Query().Where(
		user.DeletedAtIsNil(),
	).All(ctx)

	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	users := make([]*User, 0, len(entities))
	for _, e := range entities {
		users = append(users, toDomainUser(e))
	}

	return users, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, user *User) (*User, error) {
	e, err := r.dbClient.User.Create().
		SetFirstname(user.Firstname).
		SetLastname(user.Lastname).
		SetEmail(user.Email).
		SetPasswordHash(user.PasswordHash).
		Save(ctx)

	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	return toDomainUser(e), nil
}

func (r *RepositoryImpl) Update(ctx context.Context, id uuid.UUID, firstname, lastname string, language primitive.Language) (*User, error) {
	e, err := r.dbClient.User.UpdateOneID(id).
		SetFirstname(firstname).
		SetLastname(lastname).
		SetLanguage(language).
		Save(ctx)

	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	return toDomainUser(e), nil
}

func (r *RepositoryImpl) SoftDelete(ctx context.Context, id uuid.UUID) error {
	err := r.dbClient.User.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)

	return errorwrapper.Wrap(err)
}

func toDomainUser(e *ent.User) *User {
	return &User{
		ID:           e.ID,
		Email:        e.Email,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
		DeletedAt:    e.DeletedAt,
		Firstname:    e.Firstname,
		Lastname:     e.Lastname,
		Role:         e.Role,
		Language:     e.Language,
		PasswordHash: e.PasswordHash,
	}
}
