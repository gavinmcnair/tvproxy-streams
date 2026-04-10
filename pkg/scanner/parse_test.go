package scanner

import "testing"

func TestIsVideo(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"movie.mp4", true},
		{"show.mkv", true},
		{"clip.avi", true},
		{"file.txt", false},
		{"image.jpg", false},
		{"video.MP4", true},
		{"doc.pdf", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsVideo(tt.name); got != tt.want {
			t.Errorf("IsVideo(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Sunset.Boulevard.1993", "Sunset Boulevard 1993"},
		{"The_Great_Escape", "The Great Escape"},
		{"My.Film.1080p.BluRay", "My Film"},
		{"Simple Title", "Simple Title"},
		{"Movie.720p.x264", "Movie"},
		{"No.Quality.Tags.Here", "No Quality Tags Here"},
	}
	for _, tt := range tests {
		if got := CleanTitle(tt.input); got != tt.want {
			t.Errorf("CleanTitle(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseSeasonDir(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Season 1", 1},
		{"Season 12", 12},
		{"S01", 1},
		{"S3", 3},
		{"Series 2", 2},
		{"1", 1},
		{"03", 3},
		{"Downton Abbey Season 1", 1},
		{"Downton Abbey Season 6", 6},
		{"Show Name Season 12", 12},
		{"Random Name", 0},
	}
	for _, tt := range tests {
		if got := ParseSeasonDir(tt.input); got != tt.want {
			t.Errorf("ParseSeasonDir(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestIsExtrasDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Season 1", false},
		{"Season 1 Extras", true},
		{"Bonus Features", true},
		{"Behind The Scenes", true},
		{"Making Of", true},
		{"Featurettes", true},
		{"Deleted Scenes", true},
		{"S01", false},
		{"Specials", true},
		{"Interviews", true},
	}
	for _, tt := range tests {
		if got := IsExtrasDir(tt.name); got != tt.want {
			t.Errorf("IsExtrasDir(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestParseEpisodeFilename(t *testing.T) {
	tests := []struct {
		filename               string
		wantSeason, wantEp     int
		wantTitle              string
	}{
		{"S01E05 - The Pilot.mkv", 1, 5, "The Pilot"},
		{"S02E10.mkv", 2, 10, ""},
		{"s1e3 - Something.mp4", 1, 3, ""},
		{"2x04.avi", 2, 4, ""},
		{"Episode 7 - Arrival.mkv", 0, 7, "Arrival"},
		{"E03 The Return.mp4", 0, 3, "The Return"},
		{"ep5.mkv", 0, 5, ""},
		{"03 - Third One.mp4", 0, 3, ""},
		{"S01E01 - First.720p.mkv", 1, 1, "First"},
	}
	for _, tt := range tests {
		s, e, title := ParseEpisodeFilename(tt.filename)
		if s != tt.wantSeason || e != tt.wantEp {
			t.Errorf("ParseEpisodeFilename(%q) season=%d ep=%d, want season=%d ep=%d", tt.filename, s, e, tt.wantSeason, tt.wantEp)
		}
		if title != tt.wantTitle {
			t.Errorf("ParseEpisodeFilename(%q) title=%q, want %q", tt.filename, title, tt.wantTitle)
		}
	}
}
