package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Port     int    `json:"port"`
	MediaDir string `json:"media_dir"`
	BaseURL  string `json:"base_url"`
	ProbeDir string `json:"probe_dir"`
}

type MediaType string

const (
	TypeMovie  MediaType = "movie"
	TypeSeries MediaType = "series"
	TypeOther  MediaType = "other"
)

type ProbeInfo struct {
	VideoCodec  string `json:"video_codec,omitempty"`
	AudioCodec  string `json:"audio_codec,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Duration    float64 `json:"duration,omitempty"`
	AudioLayout string `json:"audio_layout,omitempty"`
	ProbedAt    string `json:"probed_at,omitempty"`
}

type MediaItem struct {
	Type       MediaType `json:"type"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Group      string    `json:"group,omitempty"`
	Series     string    `json:"series,omitempty"`
	Season     int       `json:"season,omitempty"`
	Episode    int       `json:"episode,omitempty"`
	Filename   string    `json:"filename"`
	Probe      *ProbeInfo `json:"probe,omitempty"`
}

type Library struct {
	mu       sync.RWMutex
	items    []MediaItem
	probes   map[string]*ProbeInfo
	mediaDir string
	probeDir string
	baseURL  string
}

func NewLibrary(mediaDir, probeDir, baseURL string) *Library {
	return &Library{
		mediaDir: mediaDir,
		probeDir: probeDir,
		baseURL:  baseURL,
		probes:   make(map[string]*ProbeInfo),
	}
}

func (l *Library) Scan() {
	var items []MediaItem

	moviesDir := filepath.Join(l.mediaDir, "movies")
	scanMovies(moviesDir, &items)

	tvDir := filepath.Join(l.mediaDir, "tv")
	scanTV(tvDir, &items)

	otherDir := filepath.Join(l.mediaDir, "other")
	scanOther(otherDir, &items)

	sort.Slice(items, func(i, j int) bool {
		if items[i].Type != items[j].Type {
			return items[i].Type < items[j].Type
		}
		if items[i].Type == TypeSeries {
			if items[i].Series != items[j].Series {
				return items[i].Series < items[j].Series
			}
			if items[i].Season != items[j].Season {
				return items[i].Season < items[j].Season
			}
			return items[i].Episode < items[j].Episode
		}
		return items[i].Name < items[j].Name
	})

	l.mu.Lock()
	l.items = items
	l.mu.Unlock()

	l.loadProbeCache()

	for i := range items {
		l.mu.RLock()
		p := l.probes[items[i].Path]
		l.mu.RUnlock()
		if p != nil {
			l.mu.Lock()
			l.items[i].Probe = p
			l.mu.Unlock()
		}
	}

	log.Printf("scanned %d items (%d movies, %d episodes, %d other)",
		len(items),
		countType(items, TypeMovie),
		countType(items, TypeSeries),
		countType(items, TypeOther))
}

func countType(items []MediaItem, t MediaType) int {
	n := 0
	for _, item := range items {
		if item.Type == t {
			n++
		}
	}
	return n
}

func cleanTitle(name string) string {
	name = strings.NewReplacer(".", " ", "_", " ").Replace(name)
	for _, suffix := range []string{"720p", "1080p", "2160p", "4k", "480p", "bluray", "webrip", "web-dl", "web dl", "hdtv", "x264", "x265", "h264", "h265", "aac", "ac3", "brrip", "dvdrip"} {
		idx := strings.Index(strings.ToLower(name), suffix)
		if idx > 0 {
			name = strings.TrimSpace(name[:idx])
		}
	}
	return strings.TrimSpace(name)
}

func isVideo(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".mp4", ".mkv", ".avi", ".mov", ".ts", ".m4v", ".wmv", ".flv", ".webm", ".mpg", ".mpeg":
		return true
	}
	return false
}

func scanMovies(dir string, items *[]MediaItem) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			if isVideo(e.Name()) {
				name := cleanTitle(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
				*items = append(*items, MediaItem{
					Type:     TypeMovie,
					Path:     filepath.Join("movies", e.Name()),
					Name:     name,
					Filename: e.Name(),
				})
			}
			continue
		}
		movieName := e.Name()
		subDir := filepath.Join(dir, movieName)
		files, _ := os.ReadDir(subDir)
		for _, f := range files {
			if !f.IsDir() && isVideo(f.Name()) {
				*items = append(*items, MediaItem{
					Type:     TypeMovie,
					Path:     filepath.Join("movies", movieName, f.Name()),
					Name:     movieName,
					Filename: f.Name(),
				})
				break
			}
		}
	}
}

func scanTV(dir string, items *[]MediaItem) {
	series, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, s := range series {
		if !s.IsDir() {
			continue
		}
		seriesName := s.Name()
		seriesDir := filepath.Join(dir, seriesName)
		seasons, _ := os.ReadDir(seriesDir)
		for _, sea := range seasons {
			if !sea.IsDir() {
				continue
			}
			seasonNum := parseSeasonNum(sea.Name())
			seasonDir := filepath.Join(seriesDir, sea.Name())
			episodes, _ := os.ReadDir(seasonDir)
			for _, ep := range episodes {
				if ep.IsDir() || !isVideo(ep.Name()) {
					continue
				}
				fileSeason, epNum, epTitle := parseEpisodeInfo(ep.Name())
				if fileSeason > 0 && seasonNum == 0 {
					seasonNum = fileSeason
				}
				epName := epTitle
				if epName == "" {
					epName = strings.TrimSuffix(ep.Name(), filepath.Ext(ep.Name()))
				}
				*items = append(*items, MediaItem{
					Type:     TypeSeries,
					Path:     filepath.Join("tv", seriesName, sea.Name(), ep.Name()),
					Name:     epName,
					Series:   seriesName,
					Season:   seasonNum,
					Episode:  epNum,
					Filename: ep.Name(),
				})
			}
		}
	}
}

func scanOther(dir string, items *[]MediaItem) {
	groups, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, g := range groups {
		if !g.IsDir() {
			continue
		}
		groupName := g.Name()
		groupDir := filepath.Join(dir, groupName)
		files, _ := os.ReadDir(groupDir)
		for _, f := range files {
			if f.IsDir() || !isVideo(f.Name()) {
				continue
			}
			name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			*items = append(*items, MediaItem{
				Type:     TypeOther,
				Path:     filepath.Join("other", groupName, f.Name()),
				Name:     name,
				Group:    groupName,
				Filename: f.Name(),
			})
		}
	}
}

func parseSeasonNum(name string) int {
	lower := strings.ToLower(strings.TrimSpace(name))
	var n int
	for _, pattern := range []string{
		"season %d", "season%d", "s%d", "series %d", "series%d",
	} {
		if _, err := fmt.Sscanf(lower, pattern, &n); err == nil && n > 0 {
			return n
		}
	}
	fmt.Sscanf(lower, "%d", &n)
	return n
}

func parseEpisodeInfo(filename string) (season, episode int, title string) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	lower := strings.ToLower(base)

	var s, e int
	if _, err := fmt.Sscanf(lower, "s%de%d", &s, &e); err == nil && s > 0 && e > 0 {
		season = s
		episode = e
		idx := strings.Index(lower, fmt.Sprintf("s%02de%02d", s, e))
		if idx < 0 {
			idx = strings.Index(lower, fmt.Sprintf("s%de%d", s, e))
		}
		if idx >= 0 {
			after := base[idx+len(fmt.Sprintf("s%02de%02d", s, e)):]
			after = strings.TrimLeft(after, " .-_")
			after = strings.NewReplacer(".", " ", "_", " ").Replace(after)
			after = strings.TrimSpace(after)
			for _, suffix := range []string{"720p", "1080p", "2160p", "4k", "480p", "bluray", "webrip", "web-dl", "hdtv", "x264", "x265", "h264", "h265", "aac", "ac3"} {
				idx := strings.Index(strings.ToLower(after), suffix)
				if idx >= 0 {
					after = strings.TrimSpace(after[:idx])
				}
			}
			if after != "" {
				title = after
			}
		}
		return
	}

	for i := 0; i < len(lower)-4; i++ {
		if lower[i] == 's' && lower[i+1] >= '0' && lower[i+1] <= '9' {
			rest := lower[i+1:]
			if _, err := fmt.Sscanf(rest, "%de%d", &s, &e); err == nil && s > 0 && e > 0 {
				season = s
				episode = e
				return
			}
		}
	}

	if _, err := fmt.Sscanf(lower, "%dx%d", &s, &e); err == nil && s > 0 && e > 0 {
		season = s
		episode = e
		return
	}
	for i := 0; i < len(lower)-2; i++ {
		if lower[i] >= '0' && lower[i] <= '9' {
			rest := lower[i:]
			if _, err := fmt.Sscanf(rest, "%dx%d", &s, &e); err == nil && s > 0 && e > 0 {
				season = s
				episode = e
				return
			}
		}
	}

	for _, prefix := range []string{"episode ", "episode", "ep ", "ep", "e"} {
		idx := strings.Index(lower, prefix)
		if idx >= 0 {
			rest := lower[idx+len(prefix):]
			if _, err := fmt.Sscanf(rest, "%d", &e); err == nil && e > 0 {
				episode = e
				afterNum := rest
				for len(afterNum) > 0 && afterNum[0] >= '0' && afterNum[0] <= '9' {
					afterNum = afterNum[1:]
				}
				afterNum = strings.TrimLeft(afterNum, " .-_")
				if afterNum != "" {
					realIdx := idx + len(prefix)
					for realIdx < len(base) && (base[realIdx] >= '0' && base[realIdx] <= '9') {
						realIdx++
					}
					after := strings.TrimLeft(base[realIdx:], " .-_")
					after = strings.NewReplacer(".", " ", "_", " ").Replace(after)
					if after != "" {
						title = strings.TrimSpace(after)
					}
				}
				return
			}
		}
	}

	for i := 0; i < len(lower); i++ {
		if lower[i] >= '0' && lower[i] <= '9' {
			fmt.Sscanf(lower[i:], "%d", &e)
			episode = e
			return
		}
	}

	return
}

func (l *Library) loadProbeCache() {
	if l.probeDir == "" {
		return
	}
	entries, err := os.ReadDir(l.probeDir)
	if err != nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(l.probeDir, e.Name()))
		if err != nil {
			continue
		}
		var probe ProbeInfo
		if json.Unmarshal(data, &probe) == nil {
			key := strings.TrimSuffix(e.Name(), ".json")
			l.probes[key] = &probe
		}
	}
}

func (l *Library) saveProbe(path string, probe *ProbeInfo) {
	if l.probeDir == "" {
		return
	}
	os.MkdirAll(l.probeDir, 0755)
	key := pathToKey(path)
	data, _ := json.MarshalIndent(probe, "", "  ")
	os.WriteFile(filepath.Join(l.probeDir, key+".json"), data, 0644)
	l.mu.Lock()
	l.probes[path] = probe
	l.mu.Unlock()
}

func pathToKey(path string) string {
	return strings.ReplaceAll(strings.ReplaceAll(path, "/", "_"), "\\", "_")
}

func (l *Library) probeWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.probeNext()
		}
	}
}

func (l *Library) probeNext() {
	l.mu.RLock()
	var target *MediaItem
	for i := range l.items {
		if l.items[i].Probe == nil {
			item := l.items[i]
			target = &item
			break
		}
	}
	l.mu.RUnlock()
	if target == nil {
		return
	}

	fullPath := filepath.Join(l.mediaDir, target.Path)
	probe := ffprobeFile(fullPath)
	if probe == nil {
		probe = &ProbeInfo{ProbedAt: time.Now().UTC().Format(time.RFC3339)}
	}

	l.saveProbe(target.Path, probe)

	l.mu.Lock()
	for i := range l.items {
		if l.items[i].Path == target.Path {
			l.items[i].Probe = probe
			break
		}
	}
	l.mu.Unlock()

	log.Printf("probed: %s (%s %s %s)", target.Path, probe.VideoCodec, probe.Resolution, probe.AudioLayout)
}

func (l *Library) ServeM3U(w http.ResponseWriter, r *http.Request) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	w.Header().Set("Content-Type", "audio/x-mpegurl")
	w.Header().Set("Cache-Control", "no-cache")

	fmt.Fprintln(w, "#EXTM3U")

	for _, item := range l.items {
		streamURL := l.baseURL + "/stream/" + url.PathEscape(item.Path)

		var tags []string
		tags = append(tags, fmt.Sprintf(`tvg-name="%s"`, item.Name))
		tags = append(tags, fmt.Sprintf(`tvp-type="%s"`, item.Type))

		switch item.Type {
		case TypeMovie:
			tags = append(tags, `group-title="Movies"`)
		case TypeSeries:
			tags = append(tags, fmt.Sprintf(`group-title="TV|%s"`, item.Series))
			tags = append(tags, fmt.Sprintf(`tvp-series="%s"`, item.Series))
			if item.Season > 0 {
				tags = append(tags, fmt.Sprintf(`tvp-season="%d"`, item.Season))
			}
			if item.Episode > 0 {
				tags = append(tags, fmt.Sprintf(`tvp-episode="%d"`, item.Episode))
			}
		case TypeOther:
			tags = append(tags, fmt.Sprintf(`group-title="Other|%s"`, item.Group))
			tags = append(tags, fmt.Sprintf(`tvp-group="%s"`, item.Group))
		}

		if item.Probe != nil {
			p := item.Probe
			if p.VideoCodec != "" {
				tags = append(tags, fmt.Sprintf(`tvp-vcodec="%s"`, p.VideoCodec))
			}
			if p.AudioCodec != "" {
				tags = append(tags, fmt.Sprintf(`tvp-acodec="%s"`, p.AudioCodec))
			}
			if p.Resolution != "" {
				tags = append(tags, fmt.Sprintf(`tvp-resolution="%s"`, p.Resolution))
			}
			if p.AudioLayout != "" {
				tags = append(tags, fmt.Sprintf(`tvp-audio="%s"`, p.AudioLayout))
			}
			if p.Duration > 0 {
				tags = append(tags, fmt.Sprintf(`tvp-duration="%.0f"`, p.Duration))
			}
		}

		displayName := item.Name
		if item.Type == TypeSeries && item.Season > 0 && item.Episode > 0 {
			displayName = fmt.Sprintf("%s - S%02dE%02d", item.Series, item.Season, item.Episode)
		}

		fmt.Fprintf(w, "#EXTINF:-1 %s,%s\n", strings.Join(tags, " "), displayName)
		fmt.Fprintln(w, streamURL)
	}
}

func (l *Library) ServeJSON(w http.ResponseWriter, r *http.Request) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	typeFilter := r.URL.Query().Get("type")
	seriesFilter := r.URL.Query().Get("series")

	var filtered []MediaItem
	for _, item := range l.items {
		if typeFilter != "" && string(item.Type) != typeFilter {
			continue
		}
		if seriesFilter != "" && item.Series != seriesFilter {
			continue
		}
		filtered = append(filtered, item)
	}
	if filtered == nil {
		filtered = []MediaItem{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

func (l *Library) ServeStream(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	decoded, err := url.PathUnescape(path)
	if err != nil {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	if strings.Contains(decoded, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(l.mediaDir, decoded)
	http.ServeFile(w, r, fullPath)
}

func (l *Library) ServeStatus(w http.ResponseWriter, r *http.Request) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	probed := 0
	for _, item := range l.items {
		if item.Probe != nil {
			probed++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"total":    len(l.items),
		"probed":   probed,
		"movies":   countType(l.items, TypeMovie),
		"episodes": countType(l.items, TypeSeries),
		"other":    countType(l.items, TypeOther),
	})
}

func main() {
	port := 8090
	mediaDir := "/media"
	baseURL := ""
	probeDir := "/data/probes"

	if v := os.Getenv("PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port)
	}
	if v := os.Getenv("MEDIA_DIR"); v != "" {
		mediaDir = v
	}
	if v := os.Getenv("BASE_URL"); v != "" {
		baseURL = v
	}
	if v := os.Getenv("PROBE_DIR"); v != "" {
		probeDir = v
	}
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", port)
	}

	lib := NewLibrary(mediaDir, probeDir, baseURL)
	lib.Scan()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go lib.probeWorker(ctx)

	rescanTicker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-rescanTicker.C:
				lib.Scan()
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/playlist.m3u", lib.ServeM3U)
	mux.HandleFunc("/api/library", lib.ServeJSON)
	mux.HandleFunc("/api/status", lib.ServeStatus)
	mux.HandleFunc("/stream/", lib.ServeStream)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>
			<h1>TVProxy Streams</h1>
			<ul>
				<li><a href="/playlist.m3u">M3U Playlist</a></li>
				<li><a href="/api/library">Library JSON</a></li>
				<li><a href="/api/library?type=movie">Movies</a></li>
				<li><a href="/api/library?type=series">TV Series</a></li>
				<li><a href="/api/library?type=other">Other</a></li>
				<li><a href="/api/status">Status</a></li>
			</ul>
		</body></html>`)
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("tvproxy-streams listening on %s (media: %s)", addr, mediaDir)

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		cancel()
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
