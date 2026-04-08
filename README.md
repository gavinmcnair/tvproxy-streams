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

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8090` | HTTP listen port |
| `MEDIA_DIR` | `/media` | Root media directory |
| `BASE_URL` | `http://localhost:PORT` | Public URL for stream links |
| `PROBE_DIR` | `/data/probes` | Directory for cached ffprobe results |

## Probing

Files are probed in the background using ffprobe. Results are cached in `PROBE_DIR` as JSON files. Probe data is included in the M3U output as `tvp-*` tags. The library rescans every 5 minutes.

## Docker

```bash
docker build -t tvproxy-streams .
docker run -d \
  -p 8090:8090 \
  -v /path/to/media:/media:ro \
  -v /path/to/probes:/data/probes \
  -e BASE_URL=http://192.168.1.100:8090 \
  tvproxy-streams
```

## Planned Improvements

- Config-driven scan roots with per-directory type declarations
- Directory-as-tags: every directory above the content structure becomes a filterable tag
- Clean group names without type prefixes
- Per-root type: `movie`, `series`, or `files` (no assumptions, just filename + directory tags)
- Sort movies by date, series by season/episode order
