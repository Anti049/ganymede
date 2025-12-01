package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	"github.com/zibbp/ganymede/internal/queue"
	"github.com/zibbp/ganymede/internal/tasks"
	"github.com/zibbp/ganymede/internal/vod"
	"github.com/zibbp/ganymede/internal/youtube"
)

type YoutubeService interface {
	GetYoutubeConfig(ctx echo.Context, channelID uuid.UUID) (*ent.YoutubeConfig, error)
	CreateYoutubeConfig(ctx echo.Context, channelID uuid.UUID, input CreateYoutubeConfigInput) (*ent.YoutubeConfig, error)
	UpdateYoutubeConfig(ctx echo.Context, configID uuid.UUID, input UpdateYoutubeConfigInput) (*ent.YoutubeConfig, error)
	DeleteYoutubeConfig(ctx echo.Context, configID uuid.UUID) error
	CreatePlaylistMapping(ctx echo.Context, configID uuid.UUID, input CreatePlaylistMappingInput) (*ent.YoutubePlaylistMapping, error)
	UpdatePlaylistMapping(ctx echo.Context, mappingID uuid.UUID, input UpdatePlaylistMappingInput) (*ent.YoutubePlaylistMapping, error)
	DeletePlaylistMapping(ctx echo.Context, mappingID uuid.UUID) error
	GetUploadStatus(ctx echo.Context, vodID uuid.UUID) (*ent.YoutubeUpload, error)
	GetVodQueue(ctx echo.Context, vodID uuid.UUID) (uuid.UUID, error)
	UpdateUploadStatus(ctx echo.Context, vodID uuid.UUID, status string, errorMsg string) error
}

type CreateYoutubeConfigInput = youtube.CreateYoutubeConfigInput

type UpdateYoutubeConfigInput = youtube.UpdateYoutubeConfigInput

type CreatePlaylistMappingInput = youtube.CreatePlaylistMappingInput

type UpdatePlaylistMappingInput = youtube.UpdatePlaylistMappingInput

// GetYoutubeAuthURL godoc
//
//	@Summary		Get YouTube OAuth URL
//	@Description	Get the OAuth2 authorization URL for YouTube
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Failure		500	{object}	utils.ErrorResponse
//	@Router			/youtube/auth/url [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetYoutubeAuthURL(c echo.Context) error {
	authURL := youtube.GetAuthURL()
	return c.JSON(http.StatusOK, map[string]string{"url": authURL})
}

// ExchangeYoutubeCode godoc
//
//	@Summary		Exchange YouTube OAuth code
//	@Description	Exchange an OAuth2 authorization code for credentials
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			code	body		map[string]string	true	"Authorization code"
//	@Success		200		{object}	utils.SuccessResponse
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/youtube/auth/callback [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) ExchangeYoutubeCode(c echo.Context) error {
	var input struct {
		Code string `json:"code" validate:"required"`
	}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Exchange code for token
	token, err := youtube.ExchangeCode(c.Request().Context(), input.Code)
	if err != nil {
		log.Error().Err(err).Msg("failed to exchange YouTube code")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to exchange code")
	}

	// Save credentials to database
	db := h.Service.VodService.(*vod.Service).Store
	youtubeService := youtube.NewService(db)
	if err := youtubeService.SaveCredentials(c.Request().Context(), token); err != nil {
		log.Error().Err(err).Msg("failed to save YouTube credentials")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save credentials")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "YouTube credentials saved successfully"})
}

// GetYoutubeConfig godoc
//
//	@Summary		Get YouTube config for channel
//	@Description	Get YouTube upload configuration for a channel
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			channelId	path		string	true	"Channel ID"
//	@Success		200			{object}	ent.YoutubeConfig
//	@Failure		404			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/youtube/config/channel/{channelId} [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetYoutubeConfig(c echo.Context) error {
	channelID, err := uuid.Parse(c.Param("channelId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel ID")
	}

	config, err := h.Service.YoutubeService.GetYoutubeConfig(c, channelID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "YouTube config not found")
	}

	return c.JSON(http.StatusOK, config)
}

// CreateYoutubeConfig godoc
//
//	@Summary		Create YouTube config
//	@Description	Create YouTube upload configuration for a channel
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			channelId	path		string						true	"Channel ID"
//	@Param			config		body		CreateYoutubeConfigInput	true	"YouTube config"
//	@Success		201			{object}	ent.YoutubeConfig
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/youtube/config/channel/{channelId} [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) CreateYoutubeConfig(c echo.Context) error {
	channelID, err := uuid.Parse(c.Param("channelId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid channel ID")
	}

	var input CreateYoutubeConfigInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	config, err := h.Service.YoutubeService.CreateYoutubeConfig(c, channelID, input)
	if err != nil {
		log.Error().Err(err).Msg("failed to create YouTube config")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create config")
	}

	return c.JSON(http.StatusCreated, config)
}

// UpdateYoutubeConfig godoc
//
//	@Summary		Update YouTube config
//	@Description	Update YouTube upload configuration
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			configId	path		string						true	"Config ID"
//	@Param			config		body		UpdateYoutubeConfigInput	true	"YouTube config"
//	@Success		200			{object}	ent.YoutubeConfig
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/youtube/config/{configId} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdateYoutubeConfig(c echo.Context) error {
	configID, err := uuid.Parse(c.Param("configId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid config ID")
	}

	var input UpdateYoutubeConfigInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	config, err := h.Service.YoutubeService.UpdateYoutubeConfig(c, configID, input)
	if err != nil {
		log.Error().Err(err).Msg("failed to update YouTube config")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update config")
	}

	return c.JSON(http.StatusOK, config)
}

// DeleteYoutubeConfig godoc
//
//	@Summary		Delete YouTube config
//	@Description	Delete YouTube upload configuration
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			configId	path		string	true	"Config ID"
//	@Success		200			{object}	utils.SuccessResponse
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/youtube/config/{configId} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeleteYoutubeConfig(c echo.Context) error {
	configID, err := uuid.Parse(c.Param("configId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid config ID")
	}

	if err := h.Service.YoutubeService.DeleteYoutubeConfig(c, configID); err != nil {
		log.Error().Err(err).Msg("failed to delete YouTube config")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete config")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "YouTube config deleted successfully"})
}

// CreatePlaylistMapping godoc
//
//	@Summary		Create playlist mapping
//	@Description	Create a game/category to playlist mapping
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			configId	path		string						true	"Config ID"
//	@Param			mapping		body		CreatePlaylistMappingInput	true	"Playlist mapping"
//	@Success		201			{object}	ent.YoutubePlaylistMapping
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/youtube/config/{configId}/mapping [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) CreatePlaylistMapping(c echo.Context) error {
	configID, err := uuid.Parse(c.Param("configId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid config ID")
	}

	var input CreatePlaylistMappingInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	mapping, err := h.Service.YoutubeService.CreatePlaylistMapping(c, configID, input)
	if err != nil {
		log.Error().Err(err).Msg("failed to create playlist mapping")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create mapping")
	}

	return c.JSON(http.StatusCreated, mapping)
}

// UpdatePlaylistMapping godoc
//
//	@Summary		Update playlist mapping
//	@Description	Update a game/category to playlist mapping
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			mappingId	path		string						true	"Mapping ID"
//	@Param			mapping		body		UpdatePlaylistMappingInput	true	"Playlist mapping"
//	@Success		200			{object}	ent.YoutubePlaylistMapping
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/youtube/mapping/{mappingId} [put]
//	@Security		ApiKeyCookieAuth
func (h *Handler) UpdatePlaylistMapping(c echo.Context) error {
	mappingID, err := uuid.Parse(c.Param("mappingId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mapping ID")
	}

	var input UpdatePlaylistMappingInput
	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	mapping, err := h.Service.YoutubeService.UpdatePlaylistMapping(c, mappingID, input)
	if err != nil {
		log.Error().Err(err).Msg("failed to update playlist mapping")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update mapping")
	}

	return c.JSON(http.StatusOK, mapping)
}

// DeletePlaylistMapping godoc
//
//	@Summary		Delete playlist mapping
//	@Description	Delete a game/category to playlist mapping
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			mappingId	path		string	true	"Mapping ID"
//	@Success		200			{object}	utils.SuccessResponse
//	@Failure		400			{object}	utils.ErrorResponse
//	@Failure		500			{object}	utils.ErrorResponse
//	@Router			/youtube/mapping/{mappingId} [delete]
//	@Security		ApiKeyCookieAuth
func (h *Handler) DeletePlaylistMapping(c echo.Context) error {
	mappingID, err := uuid.Parse(c.Param("mappingId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mapping ID")
	}

	if err := h.Service.YoutubeService.DeletePlaylistMapping(c, mappingID); err != nil {
		log.Error().Err(err).Msg("failed to delete playlist mapping")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete mapping")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Playlist mapping deleted successfully"})
}

// GetUploadStatus godoc
//
//	@Summary		Get upload status
//	@Description	Get YouTube upload status for a VOD
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			vodId	path		string	true	"VOD ID"
//	@Success		200		{object}	ent.YoutubeUpload
//	@Failure		404		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/youtube/upload/vod/{vodId} [get]
//	@Security		ApiKeyCookieAuth
func (h *Handler) GetUploadStatus(c echo.Context) error {
	vodID, err := uuid.Parse(c.Param("vodId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid VOD ID")
	}

	upload, err := h.Service.YoutubeService.GetUploadStatus(c, vodID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "upload status not found")
	}

	return c.JSON(http.StatusOK, upload)
}

// RetryUpload godoc
//
//	@Summary		Retry upload
//	@Description	Retry a failed YouTube upload
//	@Tags			youtube
//	@Accept			json
//	@Produce		json
//	@Param			vodId	path		string	true	"VOD ID"
//	@Success		200		{object}	utils.SuccessResponse
//	@Failure		400		{object}	utils.ErrorResponse
//	@Failure		500		{object}	utils.ErrorResponse
//	@Router			/youtube/upload/vod/{vodId}/retry [post]
//	@Security		ApiKeyCookieAuth
func (h *Handler) RetryUpload(c echo.Context) error {
	vodID, err := uuid.Parse(c.Param("vodId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid VOD ID")
	}

	// Get queue ID for the VOD
	queueID, err := h.Service.YoutubeService.GetVodQueue(c, vodID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get VOD queue")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get VOD queue")
	}

	// Queue the upload task
	_, err = h.Service.QueueService.(*queue.Service).RiverClient.Client.Insert(c.Request().Context(), &tasks.UploadToYouTubeArgs{
		Input: tasks.ArchiveVideoInput{
			QueueId: queueID,
		},
	}, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to queue upload")
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to queue upload")
	}

	// Update upload status to pending
	err = h.Service.YoutubeService.UpdateUploadStatus(c, vodID, "pending", "")
	if err != nil {
		log.Warn().Err(err).Msg("failed to update upload status")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Upload retry queued successfully"})
}
