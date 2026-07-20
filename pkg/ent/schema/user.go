package schema

import (
	"hkorpo/book/pkg/ent/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Mixin{},
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("firstname").MaxLen(100),
		field.String("lastname").MaxLen(100),
		field.String("email").MaxLen(200),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
