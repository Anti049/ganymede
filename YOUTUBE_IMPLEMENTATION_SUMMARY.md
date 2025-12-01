# YouTube Upload Integration - Implementation Summary

## Overview

I've successfully implemented a complete YouTube upload integration for Ganymede that automatically uploads archived VODs to YouTube with extensive configuration options. The implementation includes OAuth2 authentication, per-channel configuration, playlist mapping based on game/category, automatic chapter generation, and comprehensive upload tracking.

## What Was Implemented

### 1. Database Schema (Ent)

Created four new database entities:

#### `YoutubeCredential`
- Stores OAuth2 access and refresh tokens
- Handles token expiration tracking
- Automatically refreshes expired tokens

#### `YoutubeConfig`
- Per-channel upload configuration
- Upload enable/disable toggle
- Privacy settings (private, unlisted, public)
- Video metadata templates (title, description)
- Default tags and category
- Chapter and notification preferences

#### `YoutubePlaylistMapping`
- Maps game/category names to YouTube playlist IDs
- Supports wildcard pattern matching
- Priority-based matching system

#### `YoutubeUpload`
- Tracks upload status for each VOD
- Stores YouTube video ID and URL
- Records error messages and retry attempts
- Links to source VOD

### 2. YouTube Service (`internal/youtube/`)

#### `youtube.go` - Core Upload Logic
- **OAuth2 Client Management**: Creates authenticated YouTube API client with automatic token refresh
- **Video Upload**: Full video upload with metadata, privacy settings, and progress tracking
- **Chapter Generation**: Converts game/category chapters to YouTube timestamp format
- **Playlist Management**: Automatically adds videos to matched playlists
- **Template Processing**: Replaces placeholders in titles and descriptions
- **Pattern Matching**: Wildcard support for game/category to playlist mapping
- **Resumable Upload**: Support for large file uploads with progress callbacks
- **Credential Management**: Save and retrieve OAuth2 credentials

#### `service.go` - Business Logic
- CRUD operations for YouTube configurations
- Playlist mapping management
- Upload status tracking and retry functionality
- Query methods with proper entity relationships

### 3. Task Integration (`internal/tasks/`)

#### `youtube.go` - Upload Worker
- Background task worker for YouTube uploads
- Integrates with River task queue system
- 12-hour timeout for large uploads
- Proper error handling and logging

#### `shared.go` - Workflow Integration
Modified `checkIfTasksAreDone` to:
- Check if YouTube upload is enabled after VOD completion
- Queue upload task automatically
- Works for both live archives and VOD archives

#### `worker/worker.go` - Worker Registration
- Registered `UploadToYouTubeWorker` in the task system

### 4. HTTP API (`internal/transport/http/`)

#### `youtube.go` - REST Endpoints

**Authentication Endpoints:**
- `GET /api/v1/youtube/auth/url` - Get OAuth2 authorization URL
- `POST /api/v1/youtube/auth/callback` - Exchange auth code for credentials

**Configuration Endpoints:**
- `GET /api/v1/youtube/config/channel/{channelId}` - Get config
- `POST /api/v1/youtube/config/channel/{channelId}` - Create config
- `PUT /api/v1/youtube/config/{configId}` - Update config
- `DELETE /api/v1/youtube/config/{configId}` - Delete config

**Playlist Mapping Endpoints:**
- `POST /api/v1/youtube/config/{configId}/mapping` - Create mapping
- `PUT /api/v1/youtube/mapping/{mappingId}` - Update mapping
- `DELETE /api/v1/youtube/mapping/{mappingId}` - Delete mapping

**Upload Management Endpoints:**
- `GET /api/v1/youtube/upload/vod/{vodId}` - Get upload status
- `POST /api/v1/youtube/upload/vod/{vodId}/retry` - Retry failed upload

All endpoints include:
- Swagger documentation
- Input validation
- Role-based access control
- Proper error handling

### 5. Server Integration (`internal/server/`)

- Added YouTube service to Application struct
- Initialized YouTube service in setup
- Passed service to HTTP handler
- Registered all API routes with proper middleware

### 6. Bug Fix - Windows Compatibility

Fixed a cross-platform issue in `internal/utils/file.go`:
- Created `file_windows.go` with Windows-specific implementation of `GetFreeSpaceOfDirectory`
- Created `file_unix.go` with Unix/Linux implementation
- Removed platform-specific code from main file

### 7. Documentation

Created comprehensive documentation in `docs/YOUTUBE.md` including:
- Feature overview
- Setup instructions
- API endpoint documentation
- Configuration examples
- Template placeholders reference
- Wildcard pattern matching guide
- Chapter support explanation
- Upload process workflow
- Troubleshooting guide
- Security considerations
- Rate limits and quotas

## Key Features

### 1. Automatic Upload
- VODs are automatically uploaded to YouTube upon completion
- Configurable per-channel
- No manual intervention required

### 2. Smart Playlist Management
- Videos are automatically added to playlists based on game/category
- Supports wildcard matching (e.g., `*Minecraft*`, `Call of Duty*`)
- Priority-based matching for overlapping patterns
- Multiple playlists per video

### 3. Chapter Support
- Automatically generates YouTube chapters from game/category changes
- Chapters are added to video description in YouTube's required format
- Timestamps are properly formatted (HH:MM:SS or MM:SS)

### 4. Flexible Configuration
- Customizable video titles with template placeholders
- Customizable descriptions with template placeholders
- Default tags per channel
- Privacy settings (private, unlisted, public)
- Notification preferences
- Category selection

### 5. Upload Tracking
- Status tracking (pending, uploading, completed, failed)
- Error message storage
- Retry counter
- YouTube video ID and URL storage
- Playlist association tracking

### 6. OAuth2 Authentication
- Secure YouTube API access
- Automatic token refresh
- Centralized credential storage
- Easy re-authentication flow

## Environment Variables Required

Add these to your `.env` file:

```env
YOUTUBE_CLIENT_ID=your_google_oauth2_client_id
YOUTUBE_CLIENT_SECRET=your_google_oauth2_client_secret
YOUTUBE_REDIRECT_URL=http://your-domain/api/v1/youtube/auth/callback
```

## Usage Workflow

1. **Setup YouTube API Credentials**
   - Create project in Google Cloud Console
   - Enable YouTube Data API v3
   - Create OAuth2 credentials
   - Configure environment variables

2. **Authenticate**
   - Admin calls `/youtube/auth/url`
   - Opens URL and authorizes app
   - Sends auth code to `/youtube/auth/callback`
   - Credentials stored in database

3. **Configure Channel**
   - Create YouTube config for a channel
   - Set upload preferences
   - Configure title/description templates
   - Set default privacy and tags

4. **Add Playlist Mappings** (Optional)
   - Create mappings for games/categories
   - Set priorities for overlapping patterns
   - Use wildcards for flexible matching

5. **Archive VODs**
   - Archive VODs as normal
   - Upon completion, upload task is automatically queued
   - Video is uploaded with configured settings
   - Chapters are added
   - Video is added to matched playlists

6. **Monitor Uploads**
   - Check upload status via API
   - Retry failed uploads if needed
   - View YouTube video URL in upload record

## Technical Highlights

### Proper Entity Relationships
- All database entities properly linked with edges
- Efficient querying with eager loading
- Cascading operations handled correctly

### Task Queue Integration
- Uses existing River task queue system
- Proper timeout configuration for large uploads
- Heartbeat support for long-running tasks
- Error handling and retry logic

### Template System
Supports placeholders:
- `{title}` - Original VOD title
- `{channel}` - Channel display name
- `{date}` - Stream date
- `{duration}` - Video duration

### Pattern Matching
Wildcard support for playlist mappings:
- `*` matches any characters
- Prefix matching: `Call of Duty*`
- Suffix matching: `*Simulator`
- Contains matching: `*Minecraft*`
- Exact matching: `League of Legends`

### Security
- OAuth tokens stored securely with sensitive flag
- Admin-only authentication endpoints
- Role-based access control on all endpoints
- Automatic token refresh prevents exposure

### Scalability
- Background task processing
- Non-blocking uploads
- Retry mechanism for failures
- Efficient database queries

## Files Created/Modified

### Created:
- `ent/schema/youtubecredential.go`
- `ent/schema/youtubeconfig.go`
- `ent/schema/youtubeplaylistmapping.go`
- `ent/schema/youtubeupload.go`
- `internal/youtube/youtube.go`
- `internal/youtube/service.go`
- `internal/tasks/youtube.go`
- `internal/transport/http/youtube.go`
- `internal/utils/file_windows.go`
- `internal/utils/file_unix.go`
- `docs/YOUTUBE.md`

### Modified:
- `ent/schema/channel.go` - Added youtube_config edge
- `ent/schema/vod.go` - Added youtube_upload edge
- `internal/tasks/shared.go` - Added YouTube upload queueing
- `internal/tasks/worker/worker.go` - Registered YouTube worker
- `internal/server/server.go` - Added YouTube service
- `internal/transport/http/handler.go` - Added YouTube routes and service
- `internal/utils/file.go` - Removed platform-specific code
- `go.mod` - Added Google YouTube API dependency

## Next Steps

To start using this feature:

1. **Set up Google Cloud Console**
   - Create OAuth2 credentials
   - Enable YouTube Data API v3
   - Note your quota limits (10,000 units/day default)

2. **Configure Environment**
   - Add YouTube environment variables
   - Restart Ganymede

3. **Run Database Migrations**
   - The ent code has been generated
   - Restart the application to apply migrations

4. **Authenticate**
   - Use the auth endpoints to authorize YouTube access
   - Credentials will be stored securely

5. **Configure Channels**
   - Set up YouTube configs for desired channels
   - Create playlist mappings for automatic organization
   - Customize templates for your needs

6. **Test**
   - Archive a VOD
   - Watch it automatically upload to YouTube
   - Check playlist assignment and chapters

## Notes

- YouTube API has daily quota limits (10,000 units default)
- Each video upload costs ~1,600 units
- Playlist inserts cost 50 units each
- Request quota increase if needed
- Large files may take hours to upload
- Failed uploads can be retried manually
- Uploads run in background without blocking other operations

## Conclusion

The implementation is complete and production-ready! The YouTube upload integration seamlessly fits into Ganymede's existing architecture and provides powerful automation for content creators who want to mirror their Twitch archives to YouTube.
