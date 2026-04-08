package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func createFile(t *testing.T, path string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte("fake"), 0644)
}

func TestScanMovieRoot(t *testing.T) {
	dir := t.TempDir()
	moviesDir := filepath.Join(dir, "movies")

	createFile(t, filepath.Join(moviesDir, "Sunset Boulevard (1993).mp4"))
	createFile(t, filepath.Join(moviesDir, "Space Trilogy", "Journey to Mars.mkv"))
	createFile(t, filepath.Join(moviesDir, "Space Trilogy", "Return from Mars.mkv"))

	roots := []ScanRoot{{Path: moviesDir, Type: TypeMovie}}
	items := ScanRoots(roots)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	for _, item := range items {
		if item.Type != TypeMovie {
			t.Errorf("expected type movie, got %s for %s", item.Type, item.Name)
		}
	}

	collectionCount := 0
	for _, item := range items {
		if item.Collection == "Space Trilogy" {
			collectionCount++
		}
	}
	if collectionCount != 2 {
		t.Errorf("expected 2 items in Space Trilogy collection, got %d", collectionCount)
	}
}

func TestScanSeriesRoot(t *testing.T) {
	dir := t.TempDir()
	tvDir := filepath.Join(dir, "tv")

	createFile(t, filepath.Join(tvDir, "The Radio Hour", "Season 1", "S01E01 - Pilot.mkv"))
	createFile(t, filepath.Join(tvDir, "The Radio Hour", "Season 1", "S01E02 - The Interview.mkv"))
	createFile(t, filepath.Join(tvDir, "The Radio Hour", "Season 2", "S02E01 - New Start.mkv"))

	roots := []ScanRoot{{Path: tvDir, Type: TypeSeries}}
	items := ScanRoots(roots)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	for _, item := range items {
		if item.Type != TypeSeries {
			t.Errorf("expected type series, got %s", item.Type)
		}
		if item.Series != "The Radio Hour" {
			t.Errorf("expected series 'The Radio Hour', got %q", item.Series)
		}
	}

	if items[0].Season != 1 || items[0].Episode != 1 {
		t.Errorf("first item should be S01E01, got S%02dE%02d", items[0].Season, items[0].Episode)
	}
	if items[2].Season != 2 || items[2].Episode != 1 {
		t.Errorf("last item should be S02E01, got S%02dE%02d", items[2].Season, items[2].Episode)
	}
}

func TestScanFilesRoot(t *testing.T) {
	dir := t.TempDir()
	filesDir := filepath.Join(dir, "files")

	createFile(t, filepath.Join(filesDir, "training", "safety.mp4"))
	createFile(t, filepath.Join(filesDir, "training", "onboarding.mp4"))
	createFile(t, filepath.Join(filesDir, "events", "party.mp4"))

	roots := []ScanRoot{{Path: filesDir, Type: TypeFiles}}
	items := ScanRoots(roots)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	for _, item := range items {
		if item.Type != TypeFiles {
			t.Errorf("expected type files, got %s", item.Type)
		}
	}

	groupCounts := make(map[string]int)
	for _, item := range items {
		groupCounts[item.Group]++
	}
	if groupCounts["training"] != 2 {
		t.Errorf("expected 2 training items, got %d", groupCounts["training"])
	}
	if groupCounts["events"] != 1 {
		t.Errorf("expected 1 events item, got %d", groupCounts["events"])
	}
}

func TestScanMultipleRoots(t *testing.T) {
	dir := t.TempDir()

	moviesDir := filepath.Join(dir, "movies")
	tvDir := filepath.Join(dir, "tv")
	filesDir := filepath.Join(dir, "misc")

	createFile(t, filepath.Join(moviesDir, "Film One.mp4"))
	createFile(t, filepath.Join(tvDir, "Show A", "Season 1", "S01E01.mkv"))
	createFile(t, filepath.Join(filesDir, "docs", "clip.mp4"))

	roots := []ScanRoot{
		{Path: moviesDir, Type: TypeMovie},
		{Path: tvDir, Type: TypeSeries},
		{Path: filesDir, Type: TypeFiles},
	}
	items := ScanRoots(roots)

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	types := make(map[MediaType]int)
	for _, item := range items {
		types[item.Type]++
	}
	if types[TypeMovie] != 1 || types[TypeSeries] != 1 || types[TypeFiles] != 1 {
		t.Errorf("expected 1 of each type, got movies=%d series=%d files=%d", types[TypeMovie], types[TypeSeries], types[TypeFiles])
	}
}

func TestDirectoryTags(t *testing.T) {
	dir := t.TempDir()
	moviesDir := filepath.Join(dir, "movies")

	createFile(t, filepath.Join(moviesDir, "Sci-Fi", "Space Trilogy", "Journey.mkv"))
	createFile(t, filepath.Join(moviesDir, "Sci-Fi", "Space Trilogy", "Return.mkv"))

	roots := []ScanRoot{{Path: moviesDir, Type: TypeMovie}}
	items := ScanRoots(roots)

	for _, item := range items {
		found := false
		for _, tag := range item.Tags {
			if tag == "Sci-Fi" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected tag 'Sci-Fi' on %s, got tags: %v", item.Name, item.Tags)
		}
	}
}
