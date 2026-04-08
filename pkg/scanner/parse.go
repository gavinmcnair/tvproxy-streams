package scanner

import (
	"fmt"
	"path/filepath"
	"strings"
)

var videoExtensions = map[string]bool{
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true, ".ts": true,
	".m4v": true, ".wmv": true, ".flv": true, ".webm": true, ".mpg": true, ".mpeg": true,
}

var qualitySuffixes = []string{
	"720p", "1080p", "2160p", "4k", "480p", "bluray", "webrip",
	"web-dl", "web dl", "hdtv", "x264", "x265", "h264", "h265",
	"aac", "ac3", "brrip", "dvdrip",
}

func IsVideo(name string) bool {
	return videoExtensions[strings.ToLower(filepath.Ext(name))]
}

var extrasKeywords = []string{
	"extras", "bonus", "behind the scenes", "making of",
	"featurettes", "featurette", "specials", "deleted scenes",
	"interviews", "bloopers", "gag reel", "commentary",
}

func IsExtrasDir(name string) bool {
	lower := strings.ToLower(name)
	for _, kw := range extrasKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func CleanTitle(name string) string {
	name = strings.NewReplacer(".", " ", "_", " ").Replace(name)
	for _, suffix := range qualitySuffixes {
		idx := strings.Index(strings.ToLower(name), suffix)
		if idx > 0 {
			name = strings.TrimSpace(name[:idx])
		}
	}
	return strings.TrimSpace(name)
}

func ParseSeasonDir(name string) int {
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

func ParseEpisodeFilename(filename string) (season, episode int, title string) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	lower := strings.ToLower(base)

	var s, e int
	if _, err := fmt.Sscanf(lower, "s%de%d", &s, &e); err == nil && s > 0 && e > 0 {
		season = s
		episode = e
		title = extractTitleAfterPattern(base, lower, fmt.Sprintf("s%02de%02d", s, e))
		return
	}

	for i := 0; i < len(lower)-4; i++ {
		if lower[i] == 's' && lower[i+1] >= '0' && lower[i+1] <= '9' {
			if _, err := fmt.Sscanf(lower[i+1:], "%de%d", &s, &e); err == nil && s > 0 && e > 0 {
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
			if _, err := fmt.Sscanf(lower[i:], "%dx%d", &s, &e); err == nil && s > 0 && e > 0 {
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
				title = extractTitleAfterEpisodeNum(base, idx+len(prefix))
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

func extractTitleAfterPattern(base, lower, pattern string) string {
	idx := strings.Index(lower, pattern)
	if idx < 0 {
		shortPattern := strings.TrimLeft(pattern, "0")
		idx = strings.Index(lower, shortPattern)
	}
	if idx < 0 {
		return ""
	}
	after := base[idx+len(pattern):]
	after = strings.TrimLeft(after, " .-_")
	after = strings.NewReplacer(".", " ", "_", " ").Replace(after)
	after = strings.TrimSpace(after)
	for _, suffix := range qualitySuffixes {
		si := strings.Index(strings.ToLower(after), suffix)
		if si >= 0 {
			after = strings.TrimSpace(after[:si])
		}
	}
	return after
}

func extractTitleAfterEpisodeNum(base string, numStart int) string {
	i := numStart
	for i < len(base) && base[i] >= '0' && base[i] <= '9' {
		i++
	}
	after := strings.TrimLeft(base[i:], " .-_")
	after = strings.NewReplacer(".", " ", "_", " ").Replace(after)
	return strings.TrimSpace(after)
}
