package youtube

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/ent/channel"
	"github.com/zibbp/ganymede/ent/vod"
	"github.com/zibbp/ganymede/ent/youtubeconfig"
	"github.com/zibbp/ganymede/ent/youtubeupload"
	"github.com/zibbp/ganymede/internal/database"
)

// CreateYoutubeConfigInput defines the input for creating a YouTube config
type CreateYoutubeConfigInput struct {
	UploadEnabled       bool     `json:"upload_enabled"`
	DefaultPrivacy      string   `json:"default_privacy" validate:"required,oneof=private unlisted public"`
	DefaultCategoryID   string   `json:"default_category_id"`
	DescriptionTemplate string   `json:"description_template"`
	TitleTemplate       string   `json:"title_template"`
	Tags                []string `json:"tags"`
	AddChapters         bool     `json:"add_chapters"`
	NotifySubscribers   bool     `json:"notify_subscribers"`
}

// UpdateYoutubeConfigInput defines the input for updating a YouTube config
type UpdateYoutubeConfigInput struct {
	UploadEnabled       *bool     `json:"upload_enabled"`
	DefaultPrivacy      *string   `json:"default_privacy" validate:"omitempty,oneof=private unlisted public"`
	DefaultCategoryID   *string   `json:"default_category_id"`
	DescriptionTemplate *string   `json:"description_template"`
	TitleTemplate       *string   `json:"title_template"`
	Tags                *[]string `json:"tags"`
	AddChapters         *bool     `json:"add_chapters"`
	NotifySubscribers   *bool     `json:"notify_subscribers"`
}

// CreatePlaylistMappingInput defines the input for creating a playlist mapping
type CreatePlaylistMappingInput struct {
	GameCategory string `json:"game_category" validate:"required"`
	PlaylistID   string `json:"playlist_id" validate:"required"`
	PlaylistName string `json:"playlist_name"`
	Priority     int    `json:"priority"`
}

// UpdatePlaylistMappingInput defines the input for updating a playlist mapping
type UpdatePlaylistMappingInput struct {
	GameCategory *string `json:"game_category"`
	PlaylistID   *string `json:"playlist_id"`
	PlaylistName *string `json:"playlist_name"`
	Priority     *int    `json:"priority"`
}

type Service struct {
	Store *database.Database
}

func NewService(store *database.Database) *Service {
	return &Service{
		Store: store,
	}
}

func (s *Service) GetYoutubeConfig(ctx echo.Context, channelID uuid.UUID) (*ent.YoutubeConfig, error) {
	config, err := s.Store.Client.YoutubeConfig.Query().
		Where(youtubeconfig.HasChannelWith(channel.ID(channelID))).
		WithPlaylistMappings().
		Only(ctx.Request().Context())
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (s *Service) CreateYoutubeConfig(ctx echo.Context, channelID uuid.UUID, input CreateYoutubeConfigInput) (*ent.YoutubeConfig, error) {
	channel, err := s.Store.Client.Channel.Get(ctx.Request().Context(), channelID)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}

	config, err := s.Store.Client.YoutubeConfig.Create().
		SetChannel(channel).
		SetUploadEnabled(input.UploadEnabled).
		SetDefaultPrivacy(input.DefaultPrivacy).
		SetDefaultCategoryID(input.DefaultCategoryID).
		SetDescriptionTemplate(input.DescriptionTemplate).
		SetTitleTemplate(input.TitleTemplate).
		SetTags(input.Tags).
		SetAddChapters(input.AddChapters).
		SetNotifySubscribers(input.NotifySubscribers).
		Save(ctx.Request().Context())
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (s *Service) UpdateYoutubeConfig(ctx echo.Context, configID uuid.UUID, input UpdateYoutubeConfigInput) (*ent.YoutubeConfig, error) {
	update := s.Store.Client.YoutubeConfig.UpdateOneID(configID)

	if input.UploadEnabled != nil {
		update.SetUploadEnabled(*input.UploadEnabled)
	}
	if input.DefaultPrivacy != nil {
		update.SetDefaultPrivacy(*input.DefaultPrivacy)
	}
	if input.DefaultCategoryID != nil {
		update.SetDefaultCategoryID(*input.DefaultCategoryID)
	}
	if input.DescriptionTemplate != nil {
		update.SetDescriptionTemplate(*input.DescriptionTemplate)
	}
	if input.TitleTemplate != nil {
		update.SetTitleTemplate(*input.TitleTemplate)
	}
	if input.Tags != nil {
		update.SetTags(*input.Tags)
	}
	if input.AddChapters != nil {
		update.SetAddChapters(*input.AddChapters)
	}
	if input.NotifySubscribers != nil {
		update.SetNotifySubscribers(*input.NotifySubscribers)
	}

	config, err := update.Save(ctx.Request().Context())
	if err != nil {
		return nil, err
	}

	return config, nil
}

func (s *Service) DeleteYoutubeConfig(ctx echo.Context, configID uuid.UUID) error {
	return s.Store.Client.YoutubeConfig.DeleteOneID(configID).Exec(ctx.Request().Context())
}

func (s *Service) CreatePlaylistMapping(ctx echo.Context, configID uuid.UUID, input CreatePlaylistMappingInput) (*ent.YoutubePlaylistMapping, error) {
	config, err := s.Store.Client.YoutubeConfig.Get(ctx.Request().Context(), configID)
	if err != nil {
		return nil, fmt.Errorf("config not found: %w", err)
	}

	mapping, err := s.Store.Client.YoutubePlaylistMapping.Create().
		SetYoutubeConfig(config).
		SetGameCategory(input.GameCategory).
		SetPlaylistID(input.PlaylistID).
		SetPlaylistName(input.PlaylistName).
		SetPriority(input.Priority).
		Save(ctx.Request().Context())
	if err != nil {
		return nil, err
	}

	return mapping, nil
}

func (s *Service) UpdatePlaylistMapping(ctx echo.Context, mappingID uuid.UUID, input UpdatePlaylistMappingInput) (*ent.YoutubePlaylistMapping, error) {
	update := s.Store.Client.YoutubePlaylistMapping.UpdateOneID(mappingID)

	if input.GameCategory != nil {
		update.SetGameCategory(*input.GameCategory)
	}
	if input.PlaylistID != nil {
		update.SetPlaylistID(*input.PlaylistID)
	}
	if input.PlaylistName != nil {
		update.SetPlaylistName(*input.PlaylistName)
	}
	if input.Priority != nil {
		update.SetPriority(*input.Priority)
	}

	mapping, err := update.Save(ctx.Request().Context())
	if err != nil {
		return nil, err
	}

	return mapping, nil
}

func (s *Service) DeletePlaylistMapping(ctx echo.Context, mappingID uuid.UUID) error {
	return s.Store.Client.YoutubePlaylistMapping.DeleteOneID(mappingID).Exec(ctx.Request().Context())
}

func (s *Service) GetUploadStatus(ctx echo.Context, vodID uuid.UUID) (*ent.YoutubeUpload, error) {
	upload, err := s.Store.Client.YoutubeUpload.Query().
		Where(youtubeupload.HasVodWith(vod.ID(vodID))).
		Only(ctx.Request().Context())
	if err != nil {
		return nil, err
	}

	return upload, nil
}

// GetVodQueue returns the queue for a VOD (used by transport layer for retry)
func (s *Service) GetVodQueue(ctx echo.Context, vodID uuid.UUID) (uuid.UUID, error) {
	vodRecord, err := s.Store.Client.Vod.Query().
		Where(vod.ID(vodID)).
		WithQueue().
		Only(ctx.Request().Context())
	if err != nil {
		return uuid.Nil, fmt.Errorf("VOD not found: %w", err)
	}

	if vodRecord.Edges.Queue == nil {
		return uuid.Nil, fmt.Errorf("no queue found for VOD")
	}

	return vodRecord.Edges.Queue.ID, nil
}

// UpdateUploadStatus updates the upload status (used by transport layer for retry)
func (s *Service) UpdateUploadStatus(ctx echo.Context, vodID uuid.UUID, status string, errorMsg string) error {
	upload, err := s.Store.Client.YoutubeUpload.Query().
		Where(youtubeupload.HasVodWith(vod.ID(vodID))).
		Only(ctx.Request().Context())
	if err != nil {
		return err
	}

	_, err = upload.Update().SetStatus(status).SetErrorMessage(errorMsg).Save(ctx.Request().Context())
	return err
}
