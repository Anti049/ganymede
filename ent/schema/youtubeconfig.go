package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// YoutubeConfig holds the schema definition for the YoutubeConfig entity.
type YoutubeConfig struct {
	ent.Schema
}

// Fields of the YoutubeConfig.
func (YoutubeConfig) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.Bool("upload_enabled").Default(false).Comment("Enable automatic upload to YouTube for this channel"),
		field.String("default_privacy").Default("private").Comment("Default privacy status: private, unlisted, or public"),
		field.String("default_category_id").Default("20").Comment("YouTube category ID (20 = Gaming)"),
		field.String("description_template").Optional().Comment("Template for video description, supports placeholders"),
		field.String("title_template").Optional().Comment("Template for video title, supports placeholders like {title}, {date}, {channel}"),
		field.Strings("tags").Optional().Comment("Default tags to add to uploaded videos"),
		field.Bool("add_chapters").Default(true).Comment("Add chapter markers to YouTube video"),
		field.Bool("notify_subscribers").Default(false).Comment("Notify subscribers when video is uploaded"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the YoutubeConfig.
func (YoutubeConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("channel", Channel.Type).Ref("youtube_config").Unique().Required(),
		edge.To("playlist_mappings", YoutubePlaylistMapping.Type),
	}
}
