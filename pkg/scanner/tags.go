package scanner

import (
	"path/filepath"
	"strings"
)

func ExtractTags(relPath string, rootType MediaType) []string {
	dir := filepath.Dir(relPath)
	if dir == "." || dir == "" {
		return nil
	}

	parts := strings.Split(filepath.ToSlash(dir), "/")

	var structuralDepth int
	switch rootType {
	case TypeMovie:
		structuralDepth = 1
	case TypeSeries:
		structuralDepth = 2
	case TypeFiles:
		structuralDepth = 0
	}

	var tags []string
	tagDepth := len(parts) - structuralDepth
	if tagDepth < 0 {
		tagDepth = 0
	}
	for i := 0; i < tagDepth; i++ {
		tag := strings.TrimSpace(parts[i])
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
