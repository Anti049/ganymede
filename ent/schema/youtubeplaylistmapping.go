package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// YoutubePlaylistMapping holds the schema definition for the YoutubePlaylistMapping entity.
type YoutubePlaylistMapping struct {
	ent.Schema
}

// Fields of the YoutubePlaylistMapping.
func (YoutubePlaylistMapping) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("game_category").Comment("Game/category name to match (case-insensitive, supports wildcards with *)"),
		field.String("playlist_id").Comment("YouTube playlist ID to add videos to"),
		field.String("playlist_name").Optional().Comment("Human-readable playlist name for reference"),
		field.Int("priority").Default(0).Comment("Priority for matching (higher = checked first)"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the YoutubePlaylistMapping.
func (YoutubePlaylistMapping) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("youtube_config", YoutubeConfig.Type).Ref("playlist_mappings").Unique().Required(),
	}
}
