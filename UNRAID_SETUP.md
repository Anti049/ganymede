# Ganymede Unraid Setup Guide

This guide will walk you through setting up Ganymede with YouTube upload functionality on your Unraid server.

## Prerequisites

- Unraid 6.9+ with Docker support
- Community Applications plugin installed
- Sufficient storage space for VOD archiving
- Twitch application credentials
- (Optional) Google Cloud Console OAuth2 credentials for YouTube uploads

## Method 1: Docker Compose (Recommended)

### 1. Enable Docker Compose Support

Install the **Compose Manager** plugin from Community Applications if not already installed.

### 2. Create Directory Structure

Open Unraid terminal and create the following directories:

```bash
mkdir -p /mnt/user/appdata/ganymede/{config,temp,logs}
mkdir -p /mnt/user/ganymede-videos
mkdir -p /mnt/user/appdata/ganymede-db
```

### 3. Create docker-compose.yml

Navigate to your appdata folder:

```bash
cd /mnt/user/appdata/ganymede
```

Create a `docker-compose.yml` file with the following content:

```yaml
services:
  ganymede:
    container_name: ganymede
    image: ghcr.io/zibbp/ganymede:latest
    restart: unless-stopped
    depends_on:
      - ganymede-db
    environment:
      - DEBUG=false
      - TZ=America/Chicago # Set to your timezone
      # Data paths in container
      - VIDEOS_DIR=/data/videos
      - TEMP_DIR=/data/temp
      - LOGS_DIR=/data/logs
      - CONFIG_DIR=/data/config
      # Database settings
      - DB_HOST=ganymede-db
      - DB_PORT=5432
      - DB_USER=ganymede
      - DB_PASS=ChangeThisPassword123
      - DB_NAME=ganymede-prd
      - DB_SSL=disable
      # Twitch credentials (REQUIRED - get from https://dev.twitch.tv/console)
      - TWITCH_CLIENT_ID=your_twitch_client_id_here
      - TWITCH_CLIENT_SECRET=your_twitch_client_secret_here
      # YouTube settings (OPTIONAL - for automatic YouTube uploads)
      - YOUTUBE_CLIENT_ID=your_youtube_client_id.apps.googleusercontent.com
      - YOUTUBE_CLIENT_SECRET=your_youtube_client_secret
      - YOUTUBE_REDIRECT_URL=http://YOUR_UNRAID_IP:4800/api/v1/youtube/callback
      # Worker settings
      - MAX_CHAT_DOWNLOAD_EXECUTIONS=3
      - MAX_CHAT_RENDER_EXECUTIONS=2
      - MAX_VIDEO_DOWNLOAD_EXECUTIONS=2
      - MAX_VIDEO_CONVERT_EXECUTIONS=3
      - MAX_VIDEO_SPRITE_THUMBNAIL_EXECUTIONS=2
      # Optional OAuth/SSO settings (for user authentication)
      # - OAUTH_ENABLED=false
      # - OAUTH_PROVIDER_URL=
      # - OAUTH_CLIENT_ID=
      # - OAUTH_CLIENT_SECRET=
      # - OAUTH_REDIRECT_URL=http://YOUR_UNRAID_IP:4800/api/v1/auth/oauth/callback
      # Frontend settings
      - SHOW_SSO_LOGIN_BUTTON=true
      - FORCE_SSO_AUTH=false
      - REQUIRE_LOGIN=false
    volumes:
      - /mnt/user/ganymede-videos:/data/videos
      - /mnt/user/appdata/ganymede/temp:/data/temp
      - /mnt/user/appdata/ganymede/logs:/data/logs
      - /mnt/user/appdata/ganymede/config:/data/config
    ports:
      - 4800:4000
    healthcheck:
      test: curl --fail http://localhost:4000/health || exit 1
      interval: 60s
      retries: 5
      start_period: 60s
      timeout: 10s
    networks:
      - ganymede-network

  ganymede-db:
    container_name: ganymede-db
    image: postgres:14
    volumes:
      - /mnt/user/appdata/ganymede-db:/var/lib/postgresql/data
    environment:
      - POSTGRES_PASSWORD=ChangeThisPassword123
      - POSTGRES_USER=ganymede
      - POSTGRES_DB=ganymede-prd
    ports:
      - 4801:5432
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB}"]
      interval: 30s
      timeout: 60s
      retries: 5
      start_period: 60s
    networks:
      - ganymede-network

networks:
  ganymede-network:
    driver: bridge
```

### 4. Configure Environment Variables

Edit the `docker-compose.yml` file and update the following:

#### Required Settings:
- **TZ**: Your timezone (e.g., `America/New_York`, `Europe/London`)
- **DB_PASS** and **POSTGRES_PASSWORD**: Change to a strong password (must match)
- **TWITCH_CLIENT_ID**: From https://dev.twitch.tv/console
- **TWITCH_CLIENT_SECRET**: From https://dev.twitch.tv/console

#### Optional YouTube Settings (if you want automatic uploads):
- **YOUTUBE_CLIENT_ID**: From Google Cloud Console
- **YOUTUBE_CLIENT_SECRET**: From Google Cloud Console
- **YOUTUBE_REDIRECT_URL**: Replace `YOUR_UNRAID_IP` with your Unraid server's IP

#### Storage Paths:
Update volume paths if you want to store data elsewhere:
```yaml
volumes:
  - /mnt/user/your-video-storage:/data/videos
  - /mnt/cache/ganymede/temp:/data/temp  # Consider using cache for temp files
```

### 5. Start the Stack

Using Compose Manager plugin or terminal:

```bash
cd /mnt/user/appdata/ganymede
docker-compose up -d
```

Or use the Compose Manager UI in Unraid.

### 6. Access Ganymede

Open your browser and navigate to:
```
http://YOUR_UNRAID_IP:4800
```

The first time you access Ganymede, you'll need to create an admin account.

## Method 2: Unraid Template (Manual Setup)

If you prefer not to use Docker Compose, you can manually create containers:

### 1. Add PostgreSQL Container

**Docker Hub Repository**: `postgres:14`

**Network Type**: Bridge

**Port Mappings**:
- Container Port: `5432` → Host Port: `4801`

**Paths**:
- Container Path: `/var/lib/postgresql/data` → Host Path: `/mnt/user/appdata/ganymede-db`

**Environment Variables**:
- `POSTGRES_USER=ganymede`
- `POSTGRES_PASSWORD=ChangeThisPassword123`
- `POSTGRES_DB=ganymede-prd`

### 2. Add Ganymede Container

**Docker Hub Repository**: `ghcr.io/zibbp/ganymede:latest`

**Network Type**: Bridge

**Port Mappings**:
- Container Port: `4000` → Host Port: `4800`

**Paths**:
- `/data/videos` → `/mnt/user/ganymede-videos`
- `/data/temp` → `/mnt/user/appdata/ganymede/temp`
- `/data/logs` → `/mnt/user/appdata/ganymede/logs`
- `/data/config` → `/mnt/user/appdata/ganymede/config`

**Environment Variables**:
- `DB_HOST=YOUR_UNRAID_IP`
- `DB_PORT=4801`
- `DB_USER=ganymede`
- `DB_PASS=ChangeThisPassword123`
- `DB_NAME=ganymede-prd`
- `DB_SSL=disable`
- `TWITCH_CLIENT_ID=your_client_id`
- `TWITCH_CLIENT_SECRET=your_client_secret`
- `YOUTUBE_CLIENT_ID=your_youtube_id` (optional)
- `YOUTUBE_CLIENT_SECRET=your_youtube_secret` (optional)
- `YOUTUBE_REDIRECT_URL=http://YOUR_UNRAID_IP:4800/api/v1/youtube/callback` (optional)
- `VIDEOS_DIR=/data/videos`
- `TEMP_DIR=/data/temp`
- `LOGS_DIR=/data/logs`
- `CONFIG_DIR=/data/config`
- `TZ=America/Chicago`

**Important**: Wait for PostgreSQL to fully start before starting Ganymede container.

## Setting Up Twitch Credentials

1. Go to https://dev.twitch.tv/console
2. Click **"Register Your Application"**
3. Fill in:
   - **Name**: `Ganymede` (or any name)
   - **OAuth Redirect URLs**: `http://YOUR_UNRAID_IP:4800/api/v1/auth/callback`
   - **Category**: `Application Integration`
4. Click **Create**
5. Click **Manage** on your new application
6. Copy **Client ID** and **Client Secret** to your docker-compose.yml

## Setting Up YouTube (Optional)

### 1. Create Google Cloud Project

1. Go to https://console.cloud.google.com/
2. Create a new project (e.g., "Ganymede YouTube")
3. Enable **YouTube Data API v3**:
   - Click **"Enable APIs and Services"**
   - Search for **"YouTube Data API v3"**
   - Click **Enable**

### 2. Create OAuth2 Credentials

1. Go to **APIs & Services** → **Credentials**
2. Click **"Create Credentials"** → **"OAuth client ID"**
3. Configure consent screen if prompted (Internal or External)
4. Choose **"Web application"**
5. Add **Authorized redirect URI**: `http://YOUR_UNRAID_IP:4800/api/v1/youtube/callback`
6. Click **Create**
7. Copy **Client ID** and **Client Secret** to your docker-compose.yml

### 3. Authenticate with YouTube

After starting Ganymede:

1. Log into Ganymede web UI
2. Go to a channel's settings
3. Click **YouTube** tab
4. Click **"Authenticate with YouTube"**
5. Follow the OAuth flow to grant permissions
6. Configure upload settings (privacy, templates, playlists)

See [YOUTUBE_SETUP.md](YOUTUBE_SETUP.md) for detailed YouTube configuration.

## Networking Considerations

### Reverse Proxy (Optional)

If using a reverse proxy (nginx, Traefik, Caddy):

**Example Nginx configuration**:
```nginx
server {
    listen 80;
    server_name ganymede.yourdomain.com;

    location / {
        proxy_pass http://localhost:4800;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Update redirect URLs** if using a domain:
- `YOUTUBE_REDIRECT_URL=https://ganymede.yourdomain.com/api/v1/youtube/callback`
- Update Google Cloud Console redirect URIs accordingly

### Port Conflicts

If port 4800 is already in use, change the host port in docker-compose.yml:
```yaml
ports:
  - 4850:4000  # Changed from 4800 to 4850
```

## Storage Recommendations

### VOD Storage
- Use array storage for long-term VOD retention
- Path: `/mnt/user/ganymede-videos`

### Temp Directory
- Use cache SSD for better performance during downloads/conversions
- Path: `/mnt/cache/ganymede/temp`
- Temp files are automatically cleaned up after processing

### Database
- Use cache SSD for database performance
- Path: `/mnt/cache/appdata/ganymede-db`

## Performance Tuning

### Worker Concurrency

Adjust based on your Unraid hardware:

**Low-end (4 cores, 8GB RAM)**:
```yaml
- MAX_CHAT_DOWNLOAD_EXECUTIONS=2
- MAX_CHAT_RENDER_EXECUTIONS=1
- MAX_VIDEO_DOWNLOAD_EXECUTIONS=1
- MAX_VIDEO_CONVERT_EXECUTIONS=2
- MAX_VIDEO_SPRITE_THUMBNAIL_EXECUTIONS=1
```

**Mid-range (8 cores, 16GB RAM)**:
```yaml
- MAX_CHAT_DOWNLOAD_EXECUTIONS=3
- MAX_CHAT_RENDER_EXECUTIONS=2
- MAX_VIDEO_DOWNLOAD_EXECUTIONS=2
- MAX_VIDEO_CONVERT_EXECUTIONS=3
- MAX_VIDEO_SPRITE_THUMBNAIL_EXECUTIONS=2
```

**High-end (16+ cores, 32GB+ RAM)**:
```yaml
- MAX_CHAT_DOWNLOAD_EXECUTIONS=5
- MAX_CHAT_RENDER_EXECUTIONS=3
- MAX_VIDEO_DOWNLOAD_EXECUTIONS=3
- MAX_VIDEO_CONVERT_EXECUTIONS=5
- MAX_VIDEO_SPRITE_THUMBNAIL_EXECUTIONS=3
```

### CPU Pinning (Advanced)

For dedicated Ganymede performance, consider CPU pinning in Unraid Docker settings.

## Backup Strategy

### Database Backups

Schedule regular PostgreSQL backups:

```bash
# Create backup script
cat > /mnt/user/scripts/backup-ganymede-db.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/mnt/user/backups/ganymede"
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p "$BACKUP_DIR"

docker exec ganymede-db pg_dump -U ganymede ganymede-prd > "$BACKUP_DIR/ganymede_$DATE.sql"

# Keep only last 7 days of backups
find "$BACKUP_DIR" -name "ganymede_*.sql" -mtime +7 -delete
EOF

chmod +x /mnt/user/scripts/backup-ganymede-db.sh
```

Add to Unraid User Scripts plugin to run daily.

### Config Backups

The `/mnt/user/appdata/ganymede/config` directory contains OAuth tokens and settings. Include in your regular Unraid appdata backups.

## Troubleshooting

### Container Won't Start

**Check logs**:
```bash
docker logs ganymede
docker logs ganymede-db
```

**Common issues**:
- Database not ready: Wait 30 seconds, then restart Ganymede container
- Port conflicts: Check if ports 4800/4801 are available
- Missing credentials: Ensure TWITCH_CLIENT_ID and TWITCH_CLIENT_SECRET are set

### Database Connection Issues

**Verify PostgreSQL is running**:
```bash
docker exec ganymede-db pg_isready -U ganymede
```

**Reset database** (WARNING: Deletes all data):
```bash
docker-compose down
rm -rf /mnt/user/appdata/ganymede-db/*
docker-compose up -d
```

### YouTube Upload Failures

**Check credentials**:
- Verify `YOUTUBE_CLIENT_ID` and `YOUTUBE_CLIENT_SECRET` are correct
- Ensure redirect URI matches in Google Cloud Console
- Re-authenticate channel in Ganymede UI

**Check logs**:
```bash
docker logs ganymede | grep -i youtube
tail -f /mnt/user/appdata/ganymede/logs/queue.log
```

**Quota exceeded**:
- YouTube API has a default quota of 10,000 units/day
- Each upload uses ~1600 units
- Request quota increase in Google Cloud Console if needed

### High CPU/Memory Usage

**During video conversion**: Normal - video transcoding is CPU-intensive

**Reduce concurrency**:
- Lower `MAX_VIDEO_CONVERT_EXECUTIONS` value
- Lower other MAX_*_EXECUTIONS values

**Monitor resources**:
```bash
docker stats ganymede
```

### Temp Directory Filling Up

**Manual cleanup**:
```bash
docker exec ganymede rm -rf /data/temp/*
```

**Auto-cleanup**: Ganymede should clean temp files after task completion. If not:
- Check logs for errors
- Restart container
- Report issue on GitHub

## Updates

### Update Ganymede

```bash
cd /mnt/user/appdata/ganymede
docker-compose pull
docker-compose up -d
```

Or use Unraid's Docker UI to check for updates.

### Backup Before Updates

Always backup database before major version updates:
```bash
/mnt/user/scripts/backup-ganymede-db.sh
```

## Additional Resources

- **Main README**: [README.md](README.md)
- **YouTube Setup Guide**: [YOUTUBE_SETUP.md](YOUTUBE_SETUP.md)
- **GitHub Repository**: https://github.com/Zibbp/ganymede
- **Discord**: Check GitHub README for invite link
- **Unraid Forums**: Search for "Ganymede" on forums.unraid.net

## Security Best Practices

1. **Change default passwords** in docker-compose.yml
2. **Use reverse proxy with SSL** for external access
3. **Restrict OAuth scopes** in Google Cloud Console (read-only where possible)
4. **Enable REQUIRE_LOGIN** if exposing to internet
5. **Regular backups** of database and config
6. **Keep containers updated** for security patches

## Getting Help

If you encounter issues:

1. Check container logs: `docker logs ganymede`
2. Check database logs: `docker logs ganymede-db`
3. Check queue logs: `/mnt/user/appdata/ganymede/logs/queue.log`
4. Search GitHub issues: https://github.com/Zibbp/ganymede/issues
5. Join Discord community (link in GitHub README)
6. Post on Unraid forums with logs and configuration details

---

**Enjoy archiving your Twitch streams with Ganymede on Unraid! 🚀**
