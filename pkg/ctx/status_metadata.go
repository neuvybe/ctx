package ctx

import (
	"bufio"
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	documentMetadataPrefix = "<!-- ctx:doc "
	documentMetadataSuffix = " -->"
)

type documentMetadata struct {
	Status     string   `json:"status"`
	VerifiedAt string   `json:"verifiedAt"`
	Sources    []string `json:"sources"`
}

// parseDocumentMetadata extracts the single machine-readable ctx:doc comment
// from a project-fact document. Keeping the payload as JSON lets humans and
// agents edit it without adding a configuration-file dependency to the CLI.
func parseDocumentMetadata(content []byte) (documentMetadata, bool, error) {
	var metadata documentMetadata
	found := false
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, documentMetadataPrefix) || !strings.HasSuffix(line, documentMetadataSuffix) {
			continue
		}
		if found {
			return documentMetadata{}, false, fmt.Errorf("multiple ctx:doc metadata comments")
		}
		payload := strings.TrimSuffix(strings.TrimPrefix(line, documentMetadataPrefix), documentMetadataSuffix)
		if err := decodeStrictJSON([]byte(payload), &metadata); err != nil {
			return documentMetadata{}, false, fmt.Errorf("parse ctx:doc metadata: %w", err)
		}
		found = true
	}
	if err := scanner.Err(); err != nil {
		return documentMetadata{}, false, fmt.Errorf("scan ctx:doc metadata: %w", err)
	}
	return metadata, found, nil
}

func documentWordCount(content []byte) int {
	return len(strings.Fields(string(content)))
}

// documentWordWarningLimit returns deliberately generous warning thresholds:
// twice the documented writing target. Size is a routing smell, not a health
// failure, so callers should never make these limits fatal.
func documentWordWarningLimit(path string) int {
	path = strings.ToLower(filepath.ToSlash(path))
	switch path {
	case "index.md":
		return 500
	case "local/continue.md", "continue.md":
		return 600
	case "context/overview.md":
		return 1000
	default:
		if strings.HasPrefix(path, "context/") && strings.HasSuffix(path, ".md") {
			return 1600
		}
		return 0
	}
}
