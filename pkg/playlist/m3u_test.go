package playlist

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gavinmcnair/tvproxy-streams/pkg/probe"
	"github.com/gavinmcnair/tvproxy-streams/pkg/scanner"
)

func TestServeM3UMovie(t *testing.T) {
	items := []scanner.MediaItem{
		{Type: scanner.TypeMovie, Path: "movies/Film.mp4", Name: "Film", Group: "Movies", Filename: "Film.mp4"},
	}
	cache := probe.NewCache("")
	w := httptest.NewRecorder()
	ServeM3U(items, cache, "http://localhost:8090", w)

	body := w.Body.String()
	if !strings.Contains(body, "#EXTM3U") {
		t.Error("missing M3U header")
	}
	if !strings.Contains(body, `tvp-type="movie"`) {
		t.Error("missing tvp-type tag")
	}
	if !strings.Contains(body, `group-title="Movies"`) {
		t.Error("missing group-title")
	}
	if !strings.Contains(body, "http://localhost:8090/stream/") {
		t.Error("missing stream URL")
	}
}

func TestServeM3USeries(t *testing.T) {
	items := []scanner.MediaItem{
		{Type: scanner.TypeSeries, Path: "tv/Show/S01/ep.mkv", Name: "Pilot", Series: "Show", Season: 1, Episode: 1, Filename: "ep.mkv"},
	}
	cache := probe.NewCache("")
	w := httptest.NewRecorder()
	ServeM3U(items, cache, "http://localhost:8090", w)

	body := w.Body.String()
	if !strings.Contains(body, `tvp-series="Show"`) {
		t.Error("missing tvp-series")
	}
	if !strings.Contains(body, `tvp-season="1"`) {
		t.Error("missing tvp-season")
	}
	if !strings.Contains(body, `tvp-episode="1"`) {
		t.Error("missing tvp-episode")
	}
	if !strings.Contains(body, `group-title="Show"`) {
		t.Error("group-title should be series name without prefix")
	}
	if strings.Contains(body, "TV|") {
		t.Error("group-title should NOT have TV| prefix")
	}
}

func TestServeM3UTags(t *testing.T) {
	items := []scanner.MediaItem{
		{Type: scanner.TypeMovie, Path: "movies/SciFi/Film.mp4", Name: "Film", Group: "Movies", Tags: []string{"SciFi"}, Filename: "Film.mp4"},
	}
	cache := probe.NewCache("")
	w := httptest.NewRecorder()
	ServeM3U(items, cache, "http://localhost:8090", w)

	body := w.Body.String()
	if !strings.Contains(body, `tvp-tags="SciFi"`) {
		t.Error("missing tvp-tags")
	}
}

func TestServeM3UCollection(t *testing.T) {
	items := []scanner.MediaItem{
		{Type: scanner.TypeMovie, Path: "movies/Trilogy/Film1.mp4", Name: "Film One", Group: "Trilogy", Collection: "Trilogy", Filename: "Film1.mp4"},
	}
	cache := probe.NewCache("")
	w := httptest.NewRecorder()
	ServeM3U(items, cache, "http://localhost:8090", w)

	body := w.Body.String()
	if !strings.Contains(body, `tvp-collection="Trilogy"`) {
		t.Error("missing tvp-collection")
	}
	if !strings.Contains(body, `group-title="Trilogy"`) {
		t.Error("collection should be the group-title")
	}
}
