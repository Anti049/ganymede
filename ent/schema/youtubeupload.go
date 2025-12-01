package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// YoutubeUpload holds the schema definition for the YoutubeUpload entity.
type YoutubeUpload struct {
	ent.Schema
}

// Fields of the YoutubeUpload.
func (YoutubeUpload) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("youtube_video_id").Optional().Comment("YouTube video ID after successful upload"),
		field.String("youtube_url").Optional().Comment("Full YouTube URL of the uploaded video"),
		field.String("status").Default("pending").Comment("Upload status: pending, uploading, completed, failed"),
		field.String("error_message").Optional().Comment("Error message if upload failed"),
		field.Int("retry_count").Default(0).Comment("Number of times upload was retried"),
		field.Time("uploaded_at").Optional().Comment("Time when upload completed successfully"),
		field.Strings("playlist_ids").Optional().Comment("YouTube playlist IDs the video was added to"),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the YoutubeUpload.
func (YoutubeUpload) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("vod", Vod.Type).Ref("youtube_upload").Unique().Required(),
	}
}
