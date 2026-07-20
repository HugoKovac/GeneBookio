package user

import (
	"context"
	"hkorpo/book/pkg/ent"

	"hkorpo/book/pkg/ent/user"

	"hkorpo/book/pkg/errorwrapper"

	"github.com/google/uuid"
)

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
	).First(ctx)

	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	return &User{
		ID:        e.ID,
		Email:     e.Email,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		Firstname: e.Firstname,
		Lastname:  e.Lastname,
	}, nil
}

func (r *RepositoryImpl) Create(ctx context.Context, user *User) (*User, error) {
	e, err := r.dbClient.User.Create().
		SetFirstname(user.Firstname).
		SetLastname(user.Lastname).
		SetEmail(user.Email).
		Save(ctx)

	if err != nil {
		return nil, errorwrapper.Wrap(err)
	}

	return &User{
		ID:        e.ID,
		Email:     e.Email,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		Firstname: e.Firstname,
		Lastname:  e.Lastname,
	}, nil
}
