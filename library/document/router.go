package document

import (
	"path/filepath"
	"strings"
)

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".bmp": true,
}

var textExts = map[string]bool{
	".txt": true, ".md": true, ".markdown": true, ".csv": true, ".json": true,
	".yaml": true, ".yml": true, ".log": true, ".html": true, ".xml": true,
}

// DetectInputType returns img, pdf, txt, or empty when the type is unsupported.
func DetectInputType(path, requested string) string {
	if requested != "" && requested != "auto" {
		return strings.ToLower(strings.TrimSpace(requested))
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case imageExts[ext]:
		return "img"
	case ext == ".pdf":
		return "pdf"
	case textExts[ext]:
		return "txt"
	default:
		return ""
	}
}
