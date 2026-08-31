package ctx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitFolderContractRejectsNonTopLevelNames(t *testing.T) {
	tests := []struct {
		name   string
		folder string
	}{
		{name: "safe nested path", folder: "docs/ctx"},
		{name: "dot-directory nested path", folder: ".github/ctx"},
		{name: "space", folder: "agent context"},
		{name: "parent traversal", folder: "../ctx"},
		{name: "nested traversal", folder: "nested/../../ctx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mkRepo(t)
			err := InitWithOptions(repo, InitOptions{Folder: tt.folder, Mode: ModeTeam})
			if err == nil {
				t.Fatalf("InitWithOptions accepted folder %q", tt.folder)
			}
			if !strings.Contains(err.Error(), "single directory name") {
				t.Fatalf("InitWithOptions error = %q, want single-directory contract", err)
			}
		})
	}
}

func TestInitFolderContractAllowsTopLevelCustomNames(t *testing.T) {
	for _, folder := range []string{".agent", "team_context-2.0"} {
		t.Run(folder, func(t *testing.T) {
			repo := mkRepo(t)
			if err := InitWithOptions(repo, InitOptions{Folder: folder, Mode: ModeTeam}); err != nil {
				t.Fatalf("InitWithOptions folder %q: %v", folder, err)
			}
			if _, err := os.Stat(filepath.Join(repo, folder, "INDEX.md")); err != nil {
				t.Fatalf("initialized folder %q: %v", folder, err)
			}
		})
	}
}
