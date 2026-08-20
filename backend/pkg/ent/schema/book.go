package schema

import (
	"hkorpo/book/internal/primitive"
	"hkorpo/book/pkg/ent/mixin"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Book struct {
	ent.Schema
}

func (Book) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Mixin{},
	}
}

func (Book) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New),
		field.String("key").MaxLen(20).Unique(),
		field.String("title").MaxLen(100),
		field.JSON("author_names", []string{}).Optional(),
		field.JSON("author_keys", []string{}).Optional(),
		field.Text("description").Optional(),
		field.String("cover_url").MaxLen(1000).Optional(),
		field.Bool("uploaded").Default(false),
		field.Bool("parsed").Default(false),
		field.Bool("prepared").Default(false),
		field.Bool("script_generated").Default(false),
		field.Bool("tts_generated").Default(false),
		field.Enum("language").GoType(primitive.Language("")).
			Default(primitive.French.String()),
		field.Bool("failed").Default(false),
		field.String("failed_stage").MaxLen(20).Optional(),
		field.Text("error_message").Optional(),
	}
}

func (Book) Edges() []ent.Edge {
	return nil
}
