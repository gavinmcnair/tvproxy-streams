# tvproxy-streams

Lightweight media file server that generates M3U playlists from local directories. Designed to be pulled by one or more TVProxy instances for aggregation.

## Architecture

tvproxy-streams is intentionally dumb — it serves what's on disk with structured M3U tags. All intelligence (TMDB lookups, deduplication, metadata enrichment) lives in TVProxy.

Multiple tvproxy-streams instances can run across different machines. TVProxy pulls from all of them and merges the content.

## Directory Structure

Content is organised into typed root directories:

```
/media/
  movies/
    Sunset Boulevard (1993).mkv
    Space Trilogy/
      Journey to Mars (2019).mkv
      Return from Mars (2021).mkv
  tv/
    The Radio Hour/
      Season 1/
        S01E01 - Pilot.mkv
        S01E02 - The Interview.mkv
      Season 2/
        Episode 1 - New Beginnings.mkv
  other/
    training/
      safety-briefing.mp4
    events/
      company-party-2024.mp4
```

### Movies (`/media/movies/`)

- Single file: `Name (Year).ext` — detected as a standalone movie
- Directory with one video: directory name becomes the movie name
- Directory with multiple videos: detected as a collection/franchise (e.g. a trilogy)
- Subdirectories within a collection: each subdirectory is a separate movie in the collection

### TV Series (`/media/tv/`)

- Structure: `SeriesName/SeasonDir/EpisodeFile`
- Season directories: `Season 1`, `S01`, `Series 1`, or just `1`
- Episode files: `S01E01`, `1x01`, `Episode 1`, `E01`, or any file with a number
- Episode titles are extracted from the filename after the episode identifier

### Other (`/media/other/`)

- Any content that isn't movies or TV series
- Subdirectories become group names
- No filename parsing — just serves files as-is

## M3U Output

The playlist is available at `/playlist.m3u` and includes structured tags:

```
#EXTM3U
#EXTINF:-1 tvg-name="Sunset Boulevard" tvp-type="movie" group-title="Movies",Sunset Boulevard
http://host:8090/stream/movies/Sunset%20Boulevard%20(1993).mkv

#EXTINF:-1 tvg-name="Pilot" tvp-type="series" tvp-series="The Radio Hour" tvp-season="1" tvp-episode="1" group-title="TV|The Radio Hour",The Radio Hour - S01E01
http://host:8090/stream/tv/The%20Radio%20Hour/Season%201/S01E01%20-%20Pilot.mkv
```

### Tags

| Tag | Description |
|-----|-------------|
| `tvp-type` | Content type: `movie`, `series`, or `other` |
| `tvp-collection` | Movie collection/franchise name |
| `tvp-series` | TV series name |
| `tvp-season` | Season number |
| `tvp-episode` | Episode number |
| `tvp-group` | Group name (for `other` type) |
| `tvp-vcodec` | Video codec (from probe) |
| `tvp-acodec` | Audio codec (from probe) |
| `tvp-resolution` | Resolution e.g. `1080p` (from probe) |
| `tvp-audio` | Audio layout e.g. `5.1` (from probe) |
| `tvp-duration` | Duration in seconds (from probe) |

## API

| Endpoint | Description |
|----------|-------------|
| `GET /playlist.m3u` | M3U playlist with all content |
| `GET /api/library` | Full library as JSON |
| `GET /api/library?type=movie` | Movies only |
| `GET /api/library?type=series` | TV series only |
| `GET /api/library?type=other` | Other content only |
| `GET /api/library?series=Name` | Episodes for a specific series |
| `GET /api/status` | Library stats and probe progress |
| `GET /stream/{path}` | Stream a media file |
| `POST /enroll` | mTLS client enrollment (TLS mode only) |

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8090` | HTTP listen port |
| `MEDIA_DIR` | `/media` | Root media directory |
| `BASE_URL` | `http://localhost:PORT` | Public URL for stream links |
| `PROBE_DIR` | `/data/probes` | Directory for cached ffprobe results |
| `CONFIG_DIR` | `/config` | Directory for CA, server cert, and enrolled clients |
| `TLS` | (unset) | Set to `true` to enable mTLS |

## mTLS Enrollment

When tvproxy-streams is exposed over the internet (not colocated with TVProxy), enable mTLS to secure access. Only enrolled clients can access the playlist and streams.

### Enable TLS

```bash
docker run -d \
  -p 8090:8090 \
  -v /path/to/media:/media:ro \
  -v /path/to/probes:/data/probes \
  -v /path/to/config:/config \
  -e TLS=true \
  -e BASE_URL=https://streams.example.com:8090 \
  tvproxy-streams
```

On first start with `TLS=true`, tvproxy-streams generates:
- `/config/ca.crt` + `/config/ca.key` — Certificate Authority (self-signed, 10-year validity)
- `/config/server.crt` + `/config/server.key` — Server certificate (signed by CA)
- `/config/clients.json` — Enrolled client list (initially empty)

### Enroll a TVProxy client

**Step 1:** Generate a one-time enrollment token on the tvproxy-streams server:

```bash
docker exec tvproxy-streams tvproxy-streams token gavin@home.com
```

Output:
```
Enrollment token for gavin@home.com (expires in 10 minutes):
  TVP-ENROLL-a8f3c91b2d4e6f70...
```

**Step 2:** In TVProxy's web UI, go to **M3U Accounts** and create or edit the source:
- Set the **URL** to your tvproxy-streams HTTPS URL (e.g. `https://streams.example.com:8090/playlist.m3u`)
- Paste the token into the **Enrollment Token** field
- Click **Save**

TVProxy calls the `/enroll` endpoint, receives a signed client certificate, and stores it. All future requests to this source use mTLS. The token is consumed and cannot be reused.

**Step 3:** Verify enrollment. The M3U accounts list shows a green checkmark in the **mTLS** column for enrolled sources.

### Manage enrolled clients

```bash
# List all enrolled clients
docker exec tvproxy-streams tvproxy-streams clients

# Output:
# FINGERPRINT                              EMAIL                          ENROLLED
# sha256:3f8a...c2d1                       gavin@home.com                 2026-04-09 00:15
# sha256:9b1e...f4a7                       dave@office.com                2026-04-02 14:30

# Revoke a client
docker exec tvproxy-streams tvproxy-streams revoke gavin@home.com
# Revoked 1 client certificate(s) for gavin@home.com.
```

After revocation, the client's existing certificate is immediately rejected. They must re-enroll with a new token to regain access.

### How it works

1. tvproxy-streams generates a self-signed CA on first start
2. `tvproxy-streams token <email>` creates a one-time token (10-minute TTL)
3. TVProxy sends the token to `POST /enroll` — tvproxy-streams issues a client certificate signed by its CA
4. TVProxy stores the cert bundle and uses it for all future HTTPS requests to this source
5. All endpoints except `/enroll` require a valid client certificate
6. The `/enroll` endpoint itself does not require a client cert (it validates the token instead)

### Security model

- **Mutual TLS**: Both server and client authenticate via certificates
- **Self-signed CA**: No external dependencies (Let's Encrypt doesn't work for private mTLS)
- **One-time tokens**: Enrollment tokens are single-use and time-limited
- **Per-client certs**: Each TVProxy instance gets its own certificate
- **Certificate revocation**: Immediate via CLI — revoked certs are rejected on next request
- **No secrets in transit**: After enrollment, authentication is purely certificate-based

## Probing

Files are probed in the background using ffprobe. Results are cached in `PROBE_DIR` as JSON files. Probe data is included in the M3U output as `tvp-*` tags. The library rescans every 5 minutes.

## Docker

### Without TLS (local network)

```bash
docker run -d \
  -p 8090:8090 \
  -v /path/to/media:/media:ro \
  -v /path/to/probes:/data/probes \
  -e BASE_URL=http://192.168.1.100:8090 \
  tvproxy-streams
```

### With TLS (internet-facing)

```bash
docker run -d \
  -p 8090:8090 \
  -v /path/to/media:/media:ro \
  -v /path/to/probes:/data/probes \
  -v /path/to/config:/config \
  -e TLS=true \
  -e BASE_URL=https://streams.example.com:8090 \
  tvproxy-streams
```
