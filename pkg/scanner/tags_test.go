package scanner

import (
	"reflect"
	"testing"
)

func TestExtractTags(t *testing.T) {
	tests := []struct {
		relPath  string
		rootType MediaType
		want     []string
	}{
		{"Sunset Boulevard (1993).mkv", TypeMovie, nil},

		{"Sci-Fi/Journey to Mars (2019).mkv", TypeMovie, nil},

		{"Sci-Fi/Space Trilogy/Journey to Mars.mkv", TypeMovie, []string{"Sci-Fi"}},

		{"Drama/Period/War/Trenches (2020).mkv", TypeMovie, []string{"Drama", "Period"}},

		{"The Radio Hour/Season 1/S01E01.mkv", TypeSeries, nil},

		{"Drama/The Radio Hour/Season 1/S01E01.mkv", TypeSeries, []string{"Drama"}},

		{"training/safety.mp4", TypeFiles, []string{"training"}},

		{"cameras/front/2024/clip.mp4", TypeFiles, []string{"cameras", "front", "2024"}},

		{"clip.mp4", TypeFiles, nil},
	}
	for _, tt := range tests {
		got := ExtractTags(tt.relPath, tt.rootType)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ExtractTags(%q, %s) = %v, want %v", tt.relPath, tt.rootType, got, tt.want)
		}
	}
}
