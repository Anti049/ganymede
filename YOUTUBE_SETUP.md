# YouTube Upload Feature - Setup Guide

## Overview

The YouTube upload functionality has been successfully integrated into Ganymede. VODs will automatically upload to YouTube upon completion with configurable settings for metadata, chapters, and playlists.

## What Was Added

### 1. Database Schemas

-   **YoutubeCredential** - Stores OAuth2 access/refresh tokens for YouTube API authentication
-   **YoutubeConfig** - Per-channel YouTube upload configuration (privacy, templates, chapters)
-   **YoutubePlaylistMapping** - Maps game/category names to YouTube playlist IDs with wildcard support
-   **YoutubeUpload** - Tracks upload status, YouTube video ID, and errors for each VOD

### 2. Core Services

-   **internal/youtube/youtube.go** - YouTube upload logic with OAuth2 client, video upload, chapter generation, playlist assignment
-   **internal/youtube/service.go** - Business logic CRUD operations for configs, mappings, upload status
-   **internal/tasks/youtube.go** - Background worker for asynchronous YouTube uploads via River queue

### 3. API Endpoints

11 new REST endpoints added (see `/api/v1/youtube/*`):

-   OAuth2 authentication (`GET /auth-url`, `POST /exchange-code`)
-   Config management (create, update, delete, get by channel)
-   Playlist mappings (create, update, delete, list by config)
-   Upload management (status, retry)

### 4. Platform Compatibility Fixes

Fixed Windows compilation issues:

-   `internal/utils/file_windows.go` & `file_unix.go` - Platform-specific disk space checks
-   `internal/exec/exec_windows.go` & `exec_unix.go` - Platform-specific process group management
-   `internal/tasks/periodic/periodic.go` - Fixed ent upsert method usage

## Setup Instructions

### 1. Environment Variables

Add these to your `.env` file:

```env
# YouTube OAuth2 Credentials (from Google Cloud Console)
YOUTUBE_CLIENT_ID=your_client_id_here.apps.googleusercontent.com
YOUTUBE_CLIENT_SECRET=your_client_secret_here
YOUTUBE_REDIRECT_URL=http://localhost:4000/api/v1/youtube/callback
```

### 2. Google Cloud Console Setup

1. Go to https://console.cloud.google.com/
2. Create a new project or select existing
3. Enable **YouTube Data API v3**
4. Create OAuth 2.0 credentials (Desktop application or Web application)
5. Add authorized redirect URI: `http://localhost:4000/api/v1/youtube/callback`
6. Copy Client ID and Client Secret to your `.env` file

### 3. Database Migration

The new tables will be created automatically when you run the server:

```powershell
.\ganymede.exe migrate
# or just start the server
.\ganymede.exe server
```

This creates 4 new tables:

-   `youtube_credentials`
-   `youtube_configs`
-   `youtube_playlist_mappings`
-   `youtube_uploads`

### 4. OAuth2 Authentication (Per Channel)

For each Twitch channel you want to upload to YouTube:

1. **Get auth URL** (requires admin/editor role):

    ```bash
    GET /api/v1/youtube/channels/{channel_id}/auth-url
    ```

    Returns: `{ "url": "https://accounts.google.com/o/oauth2/auth?..." }`

2. **Visit the URL** in a browser and authorize Ganymede to access your YouTube channel

3. **Exchange code** for tokens:
    ```bash
    POST /api/v1/youtube/channels/{channel_id}/exchange-code
    Body: { "code": "4/0AfJ..." }
    ```

### 5. Configure Upload Settings

Create YouTube config for a channel:

```bash
POST /api/v1/youtube/channels/{channel_id}/config
Body:
{
  "upload_enabled": true,
  "default_privacy": "unlisted",
  "default_category_id": "20",
  "title_template": "{{.StreamerName}} - {{.StreamTitle}} - {{.Date}}",
  "description_template": "VOD from {{.Date}}\n\nOriginal stream: {{.TwitchURL}}",
  "tags": ["gaming", "twitch", "vod"],
  "add_chapters": true,
  "notify_subscribers": false
}
```

**Template Variables:**

-   `{{.StreamerName}}` - Twitch channel name
-   `{{.StreamTitle}}` - VOD title
-   `{{.Date}}` - Stream date (YYYY-MM-DD)
-   `{{.Game}}` - Game/category name
-   `{{.TwitchURL}}` - Original Twitch VOD URL
-   `{{.Duration}}` - Video duration

**Privacy Options:** `private`, `unlisted`, `public`

**Category IDs:** See https://developers.google.com/youtube/v3/docs/videoCategories/list

-   20 = Gaming
-   22 = People & Blogs
-   24 = Entertainment

### 6. Configure Playlist Mappings (Optional)

Automatically add uploads to playlists based on game/category:

```bash
POST /api/v1/youtube/config/{config_id}/playlists
Body:
{
  "game_category": "League of Legends",
  "playlist_id": "PLxxxxxxxxxxxxxx",
  "playlist_name": "LoL VODs",
  "priority": 100
}
```

**Wildcard Support:**

-   `*` matches any characters
-   Example: `"game_category": "League*"` matches "League of Legends", "League of Legends: Wild Rift"
-   Higher priority mappings are checked first

### 7. Monitor Uploads

Check upload status:

```bash
GET /api/v1/youtube/vods/{vod_id}/upload
```

Returns:

```json
{
	"id": "uuid",
	"vod_id": "uuid",
	"youtube_video_id": "dQw4w9WgXcQ",
	"youtube_url": "https://youtube.com/watch?v=dQw4w9WgXcQ",
	"status": "completed",
	"uploaded_at": "2024-01-15T10:30:00Z"
}
```

**Status values:** `pending`, `uploading`, `completed`, `failed`

Retry failed uploads:

```bash
POST /api/v1/youtube/vods/{vod_id}/retry
```

## How It Works

1. **VOD Completion** - When all archiving tasks complete (download, convert, upload to S3)
2. **Check Config** - If YouTube upload enabled for the channel
3. **Queue Task** - `UploadToYouTubeArgs` queued in River (default queue)
4. **Background Upload** - Worker:
    - Loads OAuth2 credentials and refreshes if needed
    - Generates chapters from game/category changes
    - Matches playlists using wildcard patterns
    - Uploads video with metadata
    - Updates upload status in database
5. **Auto-Retry** - Failed uploads retry up to 3 times

## Automatic Chapter Generation

When `add_chapters: true`, chapters are automatically generated from game/category changes:

Example VOD with game changes:

-   00:00:00 - Just Chatting
-   00:15:30 - League of Legends
-   01:45:00 - Valorant

YouTube description:

```
Chapters:
00:00:00 Just Chatting
00:15:30 League of Legends
01:45:00 Valorant
```

## API Documentation

Full API documentation available at:

-   Swagger UI: http://localhost:4000/swagger/index.html
-   After implementation, regenerate docs: `swag init -g cmd/server/main.go`

## Troubleshooting

### OAuth Errors

-   **Invalid credentials**: Check `YOUTUBE_CLIENT_ID` and `YOUTUBE_CLIENT_SECRET` in `.env`
-   **Redirect URI mismatch**: Ensure redirect URI in Google Cloud Console matches `YOUTUBE_REDIRECT_URL`
-   **Token expired**: Tokens refresh automatically, but if issues persist, re-authenticate via `/auth-url`

### Upload Failures

-   Check logs for detailed error messages
-   Verify video file exists and is readable
-   Check YouTube quota limits (default 10,000 units/day)
-   Retry failed uploads via `/api/v1/youtube/vods/{vod_id}/retry`

### Playlist Issues

-   Verify playlist ID is correct (found in YouTube playlist URL)
-   Ensure OAuth2 user has permission to add videos to the playlist
-   Check wildcard patterns are matching correctly (higher priority = checked first)

### Database Issues

-   Run migrations: `.\ganymede.exe migrate`
-   Check PostgreSQL connection
-   Verify ent code generation: `go generate ./ent`

## Architecture Notes

### Import Cycle Resolution

To avoid circular dependencies between `internal/youtube` and `internal/tasks`:

-   Input types defined in `internal/youtube/service.go`
-   Transport layer uses type aliases to youtube package types
-   Task queueing moved to transport layer (youtube.go handlers)

### Platform-Specific Code

Uses Go build tags for Windows/Unix compatibility:

-   `//go:build windows` in `*_windows.go` files
-   `//go:build unix` in `*_unix.go` files
-   Abstracts process groups, signals, disk space checks

## Future Enhancements

Potential improvements:

-   [ ] Scheduled uploads (delay publishing)
-   [ ] Thumbnail upload support
-   [ ] Multi-language descriptions
-   [ ] Community post announcements
-   [ ] Playlist creation from API
-   [ ] Batch retry for failed uploads
-   [ ] Upload progress tracking via websockets

## References

-   YouTube Data API v3: https://developers.google.com/youtube/v3
-   OAuth2 Flow: https://developers.google.com/identity/protocols/oauth2
-   Video Categories: https://developers.google.com/youtube/v3/docs/videoCategories/list
-   Quota Usage: https://developers.google.com/youtube/v3/getting-started#quota
