package scanner

import (
	"regexp"
	"strings"
)

var trackNumberPrefix = regexp.MustCompile(`^\d+[\s._-]+`)

func titleFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, extension(filename))
	name = trackNumberPrefix.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
}

func extension(filename string) string {
	if i := strings.LastIndex(filename, "."); i >= 0 {
		return filename[i:]
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
