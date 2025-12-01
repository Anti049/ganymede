package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// YoutubeCredential holds the schema definition for the YoutubeCredential entity.
type YoutubeCredential struct {
	ent.Schema
}

// Fields of the YoutubeCredential.
func (YoutubeCredential) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("access_token").Sensitive().Comment("OAuth2 access token for YouTube API"),
		field.String("refresh_token").Sensitive().Comment("OAuth2 refresh token for YouTube API"),
		field.String("token_type").Default("Bearer"),
		field.Time("expiry").Comment("Token expiration time"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the YoutubeCredential.
func (YoutubeCredential) Edges() []ent.Edge {
	return nil
}
