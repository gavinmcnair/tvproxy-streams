package scanner

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ScanRoots(roots []ScanRoot) []MediaItem {
	var items []MediaItem
	for _, root := range roots {
		switch root.Type {
		case TypeMovie:
			scanMovieRoot(root.Path, &items)
		case TypeSeries:
			scanSeriesRoot(root.Path, &items)
		case TypeFiles:
			scanFilesRoot(root.Path, &items)
		}
	}
	sortItems(items)
	return items
}

func scanMovieRoot(rootDir string, items *[]MediaItem) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		log.Printf("skipping movie root %s: %v", rootDir, err)
		return
	}
	rootName := filepath.Base(rootDir)

	for _, e := range entries {
		if !e.IsDir() {
			if IsVideo(e.Name()) {
				name := CleanTitle(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
				*items = append(*items, MediaItem{
					Type:     TypeMovie,
					Path:     filepath.Join(rootName, e.Name()),
					Name:     name,
					Group:    "Movies",
					Filename: e.Name(),
				})
			}
			continue
		}

		scanMovieDir(rootDir, rootName, e.Name(), items)
	}
}

func scanMovieDir(rootDir, rootName, dirName string, items *[]MediaItem) {
	subDir := filepath.Join(rootDir, dirName)
	files, _ := os.ReadDir(subDir)

	var videoFiles []os.DirEntry
	var subDirs []os.DirEntry
	for _, f := range files {
		if f.IsDir() {
			subDirs = append(subDirs, f)
		} else if IsVideo(f.Name()) {
			videoFiles = append(videoFiles, f)
		}
	}

	if len(subDirs) > 0 {
		for _, sd := range subDirs {
			childDir := filepath.Join(subDir, sd.Name())
			childFiles, _ := os.ReadDir(childDir)
			for _, cf := range childFiles {
				if !cf.IsDir() && IsVideo(cf.Name()) {
					relPath := filepath.Join(dirName, sd.Name(), cf.Name())
					*items = append(*items, MediaItem{
						Type:       TypeMovie,
						Path:       filepath.Join(rootName, relPath),
						Name:       CleanTitle(sd.Name()),
						Group:      dirName,
						Collection: dirName,
						Tags:       ExtractTags(relPath, TypeMovie),
						Filename:   cf.Name(),
					})
					break
				}
			}
		}
		for _, vf := range videoFiles {
			relPath := filepath.Join(dirName, vf.Name())
			*items = append(*items, MediaItem{
				Type:       TypeMovie,
				Path:       filepath.Join(rootName, relPath),
				Name:       CleanTitle(strings.TrimSuffix(vf.Name(), filepath.Ext(vf.Name()))),
				Group:      dirName,
				Collection: dirName,
				Tags:       ExtractTags(relPath, TypeMovie),
				Filename:   vf.Name(),
			})
		}
		return
	}

	if len(videoFiles) > 1 {
		for _, f := range videoFiles {
			relPath := filepath.Join(dirName, f.Name())
			*items = append(*items, MediaItem{
				Type:       TypeMovie,
				Path:       filepath.Join(rootName, relPath),
				Name:       CleanTitle(strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))),
				Group:      dirName,
				Collection: dirName,
				Tags:       ExtractTags(relPath, TypeMovie),
				Filename:   f.Name(),
			})
		}
		return
	}

	if len(videoFiles) == 1 {
		relPath := filepath.Join(dirName, videoFiles[0].Name())
		*items = append(*items, MediaItem{
			Type:     TypeMovie,
			Path:     filepath.Join(rootName, relPath),
			Name:     dirName,
			Group:    "Movies",
			Tags:     ExtractTags(relPath, TypeMovie),
			Filename: videoFiles[0].Name(),
		})
	}
}

func scanSeriesRoot(rootDir string, items *[]MediaItem) {
	seriesDirs, err := os.ReadDir(rootDir)
	if err != nil {
		log.Printf("skipping series root %s: %v", rootDir, err)
		return
	}
	rootName := filepath.Base(rootDir)

	for _, s := range seriesDirs {
		if !s.IsDir() {
			continue
		}
		seriesName := s.Name()
		seriesDir := filepath.Join(rootDir, seriesName)
		seasons, _ := os.ReadDir(seriesDir)

		for _, sea := range seasons {
			if !sea.IsDir() {
				continue
			}

			if IsExtrasDir(sea.Name()) {
				extrasDir := filepath.Join(seriesDir, sea.Name())
				extrasFiles, _ := os.ReadDir(extrasDir)
				for epIdx, ef := range extrasFiles {
					if ef.IsDir() || !IsVideo(ef.Name()) {
						continue
					}
					name := CleanTitle(strings.TrimSuffix(ef.Name(), filepath.Ext(ef.Name())))
					relPath := filepath.Join(seriesName, sea.Name(), ef.Name())
					*items = append(*items, MediaItem{
						Type:       TypeSeries,
						Path:       filepath.Join(rootName, relPath),
						Name:       name,
						Group:      seriesName,
						Series:     seriesName,
						SeasonName: sea.Name(),
						Season:     0,
						Episode:    epIdx + 1,
						Tags:       []string{sea.Name()},
						Filename:   ef.Name(),
					})
				}
				continue
			}

			seasonNum := ParseSeasonDir(sea.Name())
			seasonDir := filepath.Join(seriesDir, sea.Name())
			episodes, _ := os.ReadDir(seasonDir)

			for _, ep := range episodes {
				if ep.IsDir() || !IsVideo(ep.Name()) {
					continue
				}
				fileSeason, epNum, epTitle := ParseEpisodeFilename(ep.Name())
				if fileSeason > 0 {
					seasonNum = fileSeason
				}
				epName := epTitle
				if epName == "" {
					epName = strings.TrimSuffix(ep.Name(), filepath.Ext(ep.Name()))
				}

				relPath := filepath.Join(seriesName, sea.Name(), ep.Name())
				*items = append(*items, MediaItem{
					Type:     TypeSeries,
					Path:     filepath.Join(rootName, relPath),
					Name:     epName,
					Group:    seriesName,
					Series:   seriesName,
					Season:   seasonNum,
					Episode:  epNum,
					Tags:     ExtractTags(relPath, TypeSeries),
					Filename: ep.Name(),
				})
			}
		}
	}
}

func scanFilesRoot(rootDir string, items *[]MediaItem) {
	rootName := filepath.Base(rootDir)
	filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !IsVideo(d.Name()) {
			return nil
		}
		relPath, _ := filepath.Rel(rootDir, path)
		name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		dir := filepath.Dir(relPath)
		group := ""
		if dir != "." {
			parts := strings.Split(filepath.ToSlash(dir), "/")
			group = parts[0]
		}

		*items = append(*items, MediaItem{
			Type:     TypeFiles,
			Path:     filepath.Join(rootName, relPath),
			Name:     name,
			Group:    group,
			Tags:     ExtractTags(relPath, TypeFiles),
			Filename: d.Name(),
		})
		return nil
	})
}

func sortItems(items []MediaItem) {
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
}

func CountType(items []MediaItem, t MediaType) int {
	n := 0
	for _, item := range items {
		if item.Type == t {
			n++
		}
	}
	return n
}
