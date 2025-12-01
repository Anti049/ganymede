# YouTube Upload Feature

This feature allows Ganymede to automatically upload archived Twitch VODs and live stream recordings to YouTube with configurable settings.

## Features

-   **Automatic Upload**: Automatically upload completed VODs to YouTube after archiving
-   **OAuth2 Authentication**: Secure YouTube API authentication using OAuth2
-   **Per-Channel Configuration**: Configure upload settings for each Twitch channel independently
-   **Playlist Mapping**: Automatically add videos to YouTube playlists based on game/category
-   **Chapter Support**: Automatically add chapter markers to YouTube videos based on category changes during streams
-   **Privacy Controls**: Set video privacy (private, unlisted, public) and notification preferences
-   **Template Support**: Customize video titles and descriptions with placeholders
-   **Retry Mechanism**: Retry failed uploads with automatic retry counter
-   **Upload Status Tracking**: Track upload progress and status for each VOD

## Setup

### 1. Create YouTube API Credentials

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the YouTube Data API v3
4. Go to "Credentials" and create OAuth 2.0 credentials
5. Add authorized redirect URI: `http://your-ganymede-domain/api/v1/youtube/auth/callback`
6. Download the credentials (client ID and secret)

### 2. Configure Environment Variables

Add the following environment variables to your `.env` file:

```env
YOUTUBE_CLIENT_ID=your_client_id_here
YOUTUBE_CLIENT_SECRET=your_client_secret_here
YOUTUBE_REDIRECT_URL=http://your-ganymede-domain/api/v1/youtube/auth/callback
```

### 3. Authenticate with YouTube

1. As an admin user, call the `/api/v1/youtube/auth/url` endpoint to get the OAuth URL
2. Open the URL in your browser and authorize the application
3. After authorization, you'll receive an authorization code
4. Send the code to `/api/v1/youtube/auth/callback` endpoint
5. Your YouTube credentials will be saved to the database

## API Endpoints

### Authentication

-   `GET /api/v1/youtube/auth/url` - Get OAuth2 authorization URL
-   `POST /api/v1/youtube/auth/callback` - Exchange authorization code for credentials

### Configuration

-   `GET /api/v1/youtube/config/channel/{channelId}` - Get YouTube config for a channel
-   `POST /api/v1/youtube/config/channel/{channelId}` - Create YouTube config for a channel
-   `PUT /api/v1/youtube/config/{configId}` - Update YouTube config
-   `DELETE /api/v1/youtube/config/{configId}` - Delete YouTube config

### Playlist Mappings

-   `POST /api/v1/youtube/config/{configId}/mapping` - Create playlist mapping
-   `PUT /api/v1/youtube/mapping/{mappingId}` - Update playlist mapping
-   `DELETE /api/v1/youtube/mapping/{mappingId}` - Delete playlist mapping

### Upload Management

-   `GET /api/v1/youtube/upload/vod/{vodId}` - Get upload status for a VOD
-   `POST /api/v1/youtube/upload/vod/{vodId}/retry` - Retry failed upload

## Configuration Example

### Creating a YouTube Config

```json
POST /api/v1/youtube/config/channel/{channelId}
{
  "upload_enabled": true,
  "default_privacy": "private",
  "default_category_id": "20",
  "description_template": "Streamed by {channel} on {date}\\n\\nOriginal VOD: {title}",
  "title_template": "{title} - {channel} - {date}",
  "tags": ["twitch", "gaming", "vod"],
  "add_chapters": true,
  "notify_subscribers": false
}
```

### Template Placeholders

The following placeholders can be used in title and description templates:

-   `{title}` - Original VOD title
-   `{channel}` - Channel display name
-   `{date}` - Stream date (formatted as YYYY-MM-DD)
-   `{duration}` - Video duration (formatted as XhYmZs)

### Creating Playlist Mappings

Map game/category names to YouTube playlist IDs:

```json
POST /api/v1/youtube/config/{configId}/mapping
{
  "game_category": "League of Legends",
  "playlist_id": "PLxxxxxxxxxxxxxxxxxxxxxx",
  "playlist_name": "LoL Streams",
  "priority": 10
}
```

#### Wildcard Matching

Playlist mappings support wildcard matching with `*`:

-   `Just Chatting` - Exact match
-   `*Minecraft*` - Contains "Minecraft"
-   `Call of Duty*` - Starts with "Call of Duty"
-   `*Simulator` - Ends with "Simulator"

Mappings are checked in priority order (higher priority first).

## Chapter Support

Chapters are automatically generated based on game/category changes during the stream:

1. Game/category changes are tracked in the database as chapters
2. When uploading to YouTube, chapters are added to the video description
3. YouTube will automatically parse the chapter timestamps

Example chapter format in description:

```
Chapters:
0:00 - League of Legends
1:23:45 - Just Chatting
2:15:30 - Valorant
```

## Upload Process

1. VOD archiving completes successfully
2. System checks if YouTube upload is enabled for the channel
3. Upload task is queued
4. Video is uploaded to YouTube with configured settings
5. Chapters are added to the description
6. Video is added to matching playlists based on game/category
7. Upload status is updated in the database

## Upload Status

The upload status can be one of:

-   `pending` - Upload queued but not started
-   `uploading` - Upload in progress
-   `completed` - Upload successful
-   `failed` - Upload failed (with error message)

Failed uploads can be retried using the retry endpoint.

## Troubleshooting

### Upload Fails with Authentication Error

-   Ensure YouTube credentials are properly configured
-   Try re-authenticating by getting a new OAuth URL and completing the flow again
-   Check that the OAuth token hasn't expired (tokens are automatically refreshed)

### Video Not Added to Playlist

-   Verify the playlist ID is correct
-   Ensure the playlist is owned by the authenticated account
-   Check the game/category matching rules (wildcards, priority)
-   Look at the upload record to see which playlists were matched

### Chapters Not Appearing

-   Ensure `add_chapters` is enabled in the YouTube config
-   Verify that chapters exist in the VOD database
-   Check that chapter timestamps are in the correct format
-   YouTube may take time to process chapters

### Upload Stuck in "Uploading" Status

-   Large files can take hours to upload
-   Check the task logs for progress
-   If truly stuck, use the retry endpoint to restart the upload

## Database Schema

### YoutubeCredential

Stores OAuth2 credentials for YouTube API access (one record per installation).

### YoutubeConfig

Per-channel configuration for YouTube uploads.

### YoutubePlaylistMapping

Maps game/category names to YouTube playlist IDs.

### YoutubeUpload

Tracks upload status and metadata for each VOD uploaded to YouTube.

## Security Considerations

-   YouTube credentials are stored encrypted in the database
-   OAuth tokens are automatically refreshed when expired
-   Only admin users can configure YouTube authentication
-   Editor/Admin roles required for managing upload configurations
-   API endpoints are protected by authentication middleware

## Rate Limits

YouTube API has the following limits:

-   **Quota**: 10,000 units per day (default)
-   **Video Upload**: ~1,600 units per upload
-   **Playlist Insert**: 50 units per video added to playlist

Plan your upload volume accordingly. If you need higher quotas, request an increase from Google Cloud Console.

## Future Enhancements

Potential improvements for future versions:

-   [ ] Progress tracking during upload
-   [ ] Bulk upload management
-   [ ] Video thumbnail upload
-   [ ] Custom thumbnail generation
-   [ ] Video cards and end screens
-   [ ] Translation/subtitle upload
-   [ ] Analytics integration
-   [ ] Multiple YouTube account support
-   [ ] Upload scheduling
-   [ ] Bandwidth limiting
