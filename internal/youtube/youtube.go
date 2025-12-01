package youtube

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/zibbp/ganymede/ent"
	entVod "github.com/zibbp/ganymede/ent/vod"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// getYouTubeClient creates an authenticated YouTube API client
func (s *Service) getYouTubeClient(ctx context.Context) (*youtube.Service, error) {
	// Get credentials from database
	creds, err := s.Store.Client.YoutubeCredential.Query().First(ctx)
	if err != nil {
		return nil, fmt.Errorf("no YouTube credentials found: %w", err)
	}

	// Check if token is expired
	if time.Now().After(creds.Expiry) {
		// Refresh the token
		token := &oauth2.Token{
			AccessToken:  creds.AccessToken,
			RefreshToken: creds.RefreshToken,
			TokenType:    creds.TokenType,
			Expiry:       creds.Expiry,
		}

		// Create OAuth2 config (client ID and secret should come from environment)
		// Note: You'll need to add these to your config
		config := &oauth2.Config{
			ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
			ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
			Scopes: []string{
				youtube.YoutubeUploadScope,
				youtube.YoutubeForceSslScope,
			},
		}

		// Refresh token
		tokenSource := config.TokenSource(ctx, token)
		newToken, err := tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		// Update database with new token
		_, err = creds.Update().
			SetAccessToken(newToken.AccessToken).
			SetExpiry(newToken.Expiry).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to save refreshed token: %w", err)
		}

		token = newToken
	}

	// Create OAuth2 config
	config := &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	token := &oauth2.Token{
		AccessToken:  creds.AccessToken,
		RefreshToken: creds.RefreshToken,
		TokenType:    creds.TokenType,
		Expiry:       creds.Expiry,
	}

	client := config.Client(ctx, token)

	// Create YouTube service
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	return service, nil
}

// UploadVideo uploads a video to YouTube
func (s *Service) UploadVideo(ctx context.Context, vodID string) error {
	logger := log.With().Str("service", "youtube").Str("vod_id", vodID).Logger()
	logger.Info().Msg("starting YouTube upload")

	// Parse VOD ID
	vodUUID, err := uuid.Parse(vodID)
	if err != nil {
		return fmt.Errorf("invalid VOD ID: %w", err)
	}

	// Get VOD from database
	vod, err := s.Store.Client.Vod.Query().
		Where(entVod.ID(vodUUID)).
		WithChannel().
		WithChapters().
		WithYoutubeUpload().
		Only(ctx)
	if err != nil {
		return fmt.Errorf("failed to get VOD: %w", err)
	}

	// Get YouTube config for channel
	youtubeConfig, err := vod.Edges.Channel.QueryYoutubeConfig().
		WithPlaylistMappings().
		Only(ctx)
	if err != nil {
		return fmt.Errorf("no YouTube config found for channel: %w", err)
	}

	if !youtubeConfig.UploadEnabled {
		logger.Info().Msg("YouTube upload disabled for channel")
		return nil
	}

	// Create or get upload record
	var uploadRecord *ent.YoutubeUpload
	if vod.Edges.YoutubeUpload != nil {
		uploadRecord = vod.Edges.YoutubeUpload
		// Skip if already uploaded
		if uploadRecord.Status == "completed" {
			logger.Info().Msg("video already uploaded to YouTube")
			return nil
		}
	} else {
		uploadRecord, err = s.Store.Client.YoutubeUpload.Create().
			SetVod(vod).
			SetStatus("pending").
			Save(ctx)
		if err != nil {
			return fmt.Errorf("failed to create upload record: %w", err)
		}
	}

	// Update status to uploading
	uploadRecord, err = uploadRecord.Update().
		SetStatus("uploading").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update upload status: %w", err)
	}

	// Get YouTube client
	ytService, err := s.getYouTubeClient(ctx)
	if err != nil {
		uploadRecord.Update().
			SetStatus("failed").
			SetErrorMessage(err.Error()).
			Save(ctx)
		return fmt.Errorf("failed to get YouTube client: %w", err)
	}

	// Prepare video metadata
	title := s.formatTitle(youtubeConfig, vod)
	description := s.formatDescription(youtubeConfig, vod)
	tags := youtubeConfig.Tags

	video := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       title,
			Description: description,
			CategoryId:  youtubeConfig.DefaultCategoryID,
			Tags:        tags,
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus:           youtubeConfig.DefaultPrivacy,
			SelfDeclaredMadeForKids: false,
		},
	}

	// Open video file
	videoPath := filepath.Join(vod.VideoPath)
	file, err := os.Open(videoPath)
	if err != nil {
		uploadRecord.Update().
			SetStatus("failed").
			SetErrorMessage(fmt.Sprintf("failed to open video file: %v", err)).
			Save(ctx)
		return fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	// Create upload call
	call := ytService.Videos.Insert([]string{"snippet", "status"}, video)
	call = call.Media(file)

	if !youtubeConfig.NotifySubscribers {
		call = call.NotifySubscribers(false)
	}

	logger.Info().Msg("uploading video to YouTube...")

	// Execute upload
	response, err := call.Do()
	if err != nil {
		retryCount := uploadRecord.RetryCount + 1
		uploadRecord.Update().
			SetStatus("failed").
			SetErrorMessage(fmt.Sprintf("upload failed: %v", err)).
			SetRetryCount(retryCount).
			Save(ctx)
		return fmt.Errorf("upload failed: %w", err)
	}

	logger.Info().Str("youtube_id", response.Id).Msg("video uploaded successfully")

	// Add chapters if enabled
	if youtubeConfig.AddChapters && len(vod.Edges.Chapters) > 0 {
		chapters := s.buildChapterDescription(vod.Edges.Chapters)
		if chapters != "" {
			// Append chapters to description
			fullDescription := description + "\n\n" + chapters
			_, err = ytService.Videos.Update([]string{"snippet"}, &youtube.Video{
				Id: response.Id,
				Snippet: &youtube.VideoSnippet{
					Title:       title,
					Description: fullDescription,
					CategoryId:  youtubeConfig.DefaultCategoryID,
					Tags:        tags,
				},
			}).Do()
			if err != nil {
				logger.Warn().Err(err).Msg("failed to add chapters")
			} else {
				logger.Info().Msg("chapters added to video")
			}
		}
	}

	// Add video to playlists
	playlistIDs := s.matchPlaylists(youtubeConfig, vod)
	for _, playlistID := range playlistIDs {
		err = s.addVideoToPlaylist(ctx, ytService, response.Id, playlistID)
		if err != nil {
			logger.Warn().Err(err).Str("playlist_id", playlistID).Msg("failed to add video to playlist")
		} else {
			logger.Info().Str("playlist_id", playlistID).Msg("video added to playlist")
		}
	}

	// Update upload record
	youtubeURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", response.Id)
	_, err = uploadRecord.Update().
		SetYoutubeVideoID(response.Id).
		SetYoutubeURL(youtubeURL).
		SetStatus("completed").
		SetUploadedAt(time.Now()).
		SetPlaylistIds(playlistIDs).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update upload record: %w", err)
	}

	logger.Info().Str("url", youtubeURL).Msg("YouTube upload completed")
	return nil
}

// formatTitle formats the video title using the template
func (s *Service) formatTitle(config *ent.YoutubeConfig, vod *ent.Vod) string {
	if config.TitleTemplate == "" {
		return vod.Title
	}

	title := config.TitleTemplate
	title = strings.ReplaceAll(title, "{title}", vod.Title)
	title = strings.ReplaceAll(title, "{channel}", vod.Edges.Channel.DisplayName)
	title = strings.ReplaceAll(title, "{date}", vod.StreamedAt.Format("2006-01-02"))
	return title
}

// formatDescription formats the video description using the template
func (s *Service) formatDescription(config *ent.YoutubeConfig, vod *ent.Vod) string {
	if config.DescriptionTemplate == "" {
		return fmt.Sprintf("Streamed by %s on %s", vod.Edges.Channel.DisplayName, vod.StreamedAt.Format("January 2, 2006"))
	}

	desc := config.DescriptionTemplate
	desc = strings.ReplaceAll(desc, "{title}", vod.Title)
	desc = strings.ReplaceAll(desc, "{channel}", vod.Edges.Channel.DisplayName)
	desc = strings.ReplaceAll(desc, "{date}", vod.StreamedAt.Format("January 2, 2006"))
	desc = strings.ReplaceAll(desc, "{duration}", formatDuration(vod.Duration))
	return desc
}

// buildChapterDescription creates a chapter description for YouTube
func (s *Service) buildChapterDescription(chapters []*ent.Chapter) string {
	if len(chapters) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Chapters:\n")

	for _, chapter := range chapters {
		timestamp := formatTimestamp(chapter.Start)
		title := chapter.Title
		if title == "" {
			title = chapter.Type
		}
		sb.WriteString(fmt.Sprintf("%s - %s\n", timestamp, title))
	}

	return sb.String()
}

// matchPlaylists finds matching playlists for a VOD based on game/category
func (s *Service) matchPlaylists(config *ent.YoutubeConfig, vod *ent.Vod) []string {
	var playlistIDs []string

	// Get the primary category from chapters
	categories := s.getCategoriesFromChapters(vod.Edges.Chapters)

	// Match against playlist mappings (sorted by priority)
	for _, mapping := range config.Edges.PlaylistMappings {
		for _, category := range categories {
			if s.matchCategory(mapping.GameCategory, category) {
				playlistIDs = append(playlistIDs, mapping.PlaylistID)
				break // Only add playlist once
			}
		}
	}

	return playlistIDs
}

// getCategoriesFromChapters extracts unique categories from chapters
func (s *Service) getCategoriesFromChapters(chapters []*ent.Chapter) []string {
	categoryMap := make(map[string]bool)
	var categories []string

	for _, chapter := range chapters {
		if chapter.Type != "" && !categoryMap[chapter.Type] {
			categoryMap[chapter.Type] = true
			categories = append(categories, chapter.Type)
		}
	}

	return categories
}

// matchCategory checks if a category matches a pattern (supports wildcards)
func (s *Service) matchCategory(pattern, category string) bool {
	pattern = strings.ToLower(pattern)
	category = strings.ToLower(category)

	// Simple wildcard matching
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			// Prefix and suffix matching
			if parts[0] == "" {
				return strings.HasSuffix(category, parts[1])
			}
			if parts[1] == "" {
				return strings.HasPrefix(category, parts[0])
			}
			return strings.HasPrefix(category, parts[0]) && strings.HasSuffix(category, parts[1])
		}
	}

	return pattern == category
}

// addVideoToPlaylist adds a video to a YouTube playlist
func (s *Service) addVideoToPlaylist(ctx context.Context, service *youtube.Service, videoID, playlistID string) error {
	playlistItem := &youtube.PlaylistItem{
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistID,
			ResourceId: &youtube.ResourceId{
				Kind:    "youtube#video",
				VideoId: videoID,
			},
		},
	}

	_, err := service.PlaylistItems.Insert([]string{"snippet"}, playlistItem).Do()
	return err
}

// formatDuration formats duration in seconds to human-readable format
func formatDuration(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	}
	return fmt.Sprintf("%dm %ds", minutes, secs)
}

// formatTimestamp formats seconds to YouTube timestamp format (HH:MM:SS or MM:SS)
func formatTimestamp(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

// SaveCredentials saves YouTube OAuth2 credentials to the database
func (s *Service) SaveCredentials(ctx context.Context, token *oauth2.Token) error {
	// Check if credentials already exist
	existing, err := s.Store.Client.YoutubeCredential.Query().First(ctx)
	if err == nil {
		// Update existing
		_, err = existing.Update().
			SetAccessToken(token.AccessToken).
			SetRefreshToken(token.RefreshToken).
			SetTokenType(token.TokenType).
			SetExpiry(token.Expiry).
			Save(ctx)
		return err
	}

	// Create new
	_, err = s.Store.Client.YoutubeCredential.Create().
		SetAccessToken(token.AccessToken).
		SetRefreshToken(token.RefreshToken).
		SetTokenType(token.TokenType).
		SetExpiry(token.Expiry).
		Save(ctx)
	return err
}

// GetAuthURL returns the OAuth2 authorization URL
func GetAuthURL() string {
	config := &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("YOUTUBE_REDIRECT_URL"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{
			youtube.YoutubeUploadScope,
			youtube.YoutubeForceSslScope,
		},
	}

	return config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an OAuth2 authorization code for a token
func ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	config := &oauth2.Config{
		ClientID:     os.Getenv("YOUTUBE_CLIENT_ID"),
		ClientSecret: os.Getenv("YOUTUBE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("YOUTUBE_REDIRECT_URL"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		},
		Scopes: []string{
			youtube.YoutubeUploadScope,
			youtube.YoutubeForceSslScope,
		},
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// ResumableUpload performs a resumable upload for large files
type ResumableUpload struct {
	Service    *youtube.Service
	Video      *youtube.Video
	FilePath   string
	ChunkSize  int64
	OnProgress func(bytesUploaded, totalBytes int64)
	OnComplete func(videoID string)
	OnError    func(err error)
}

func (r *ResumableUpload) Upload(ctx context.Context) error {
	file, err := os.Open(r.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	call := r.Service.Videos.Insert([]string{"snippet", "status"}, r.Video)
	call = call.Media(file)

	// Use progress reader if callback is provided
	if r.OnProgress != nil {
		reader := &progressReader{
			reader:     file,
			totalBytes: fileInfo.Size(),
			onProgress: r.OnProgress,
		}
		call = call.Media(reader)
	}

	response, err := call.Do()
	if err != nil {
		if r.OnError != nil {
			r.OnError(err)
		}
		return fmt.Errorf("upload failed: %w", err)
	}

	if r.OnComplete != nil {
		r.OnComplete(response.Id)
	}

	return nil
}

type progressReader struct {
	reader        io.Reader
	totalBytes    int64
	bytesUploaded int64
	onProgress    func(bytesUploaded, totalBytes int64)
}

func (pr *progressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.bytesUploaded += int64(n)
	if pr.onProgress != nil {
		pr.onProgress(pr.bytesUploaded, pr.totalBytes)
	}
	return
}
