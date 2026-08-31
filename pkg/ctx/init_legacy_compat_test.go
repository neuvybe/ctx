package ctx

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

func TestInitLegacyCompatibilityExactLayoutAndDetection(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatalf("Init: %v", err)
	}

	dest := filepath.Join(repo, ".ctx")
	var files []string
	if err := fs.WalkDir(os.DirFS(dest), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, filepath.ToSlash(path))
		}
		return nil
	}); err != nil {
		t.Fatalf("walk legacy scaffold: %v", err)
	}
	want := []string{
		".ctx-version",
		"CONTINUE.md",
		"INDEX.md",
		"OPERATING.md",
		"README.md",
		"REVIEW.md",
		"context/architecture.md",
		"context/extending.md",
		"context/format.md",
		"context/glossary.md",
		"context/known-issues.md",
		"context/overview.md",
	}
	sort.Strings(files)
	sort.Strings(want)
	if !slices.Equal(files, want) {
		t.Fatalf("legacy files = %v, want exact pre-schema layout %v", files, want)
	}

	for _, absent := range []string{"config.json", ".gitignore", "local"} {
		if _, err := os.Lstat(filepath.Join(dest, absent)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("new-layout path %s exists in legacy scaffold (stat error: %v)", absent, err)
		}
	}
	state, err := loadScaffoldState(dest)
	if err != nil {
		t.Fatalf("load legacy scaffold state: %v", err)
	}
	if !state.Legacy || state.Config.Mode != ModeLocal || state.continuePath() != "CONTINUE.md" {
		t.Fatalf("legacy scaffold state = %+v (continuation %q), want local legacy root continuation", state, state.continuePath())
	}
}

func TestInitLegacyCompatibilityDoctorAndPrivacy(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatalf("Init: %v", err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	for _, check := range checks {
		if !check.OK {
			t.Errorf("Doctor check %q failed for compatibility scaffold: %s", check.Name, check.Detail)
		}
	}

	excluded, err := hasFolderExclusion(repo, ".ctx")
	if err != nil {
		t.Fatalf("inspect common info/exclude: %v", err)
	}
	if !excluded {
		t.Fatal("legacy scaffold is missing its whole-folder common info/exclude rule")
	}
	for _, path := range []string{".ctx", ".ctx/README.md", ".ctx/CONTINUE.md", ".ctx/future-private-state"} {
		ignored, err := gitCheckIgnored(repo, path)
		if err != nil {
			t.Fatalf("check privacy for %s: %v", path, err)
		}
		if !ignored {
			t.Errorf("legacy path %s is visible to Git", path)
		}
	}
	tracked, err := gitTrackedFiles(repo, ".ctx")
	if err != nil {
		t.Fatalf("inspect tracked legacy paths: %v", err)
	}
	if len(tracked) != 0 {
		t.Fatalf("legacy scaffold has tracked paths: %v", tracked)
	}
}

func TestInitLegacyCompatibilityAcceptsSafeContainedCustomPaths(t *testing.T) {
	for _, folder := range []string{filepath.Join("docs", "ctx"), "agent context"} {
		t.Run(strings.ReplaceAll(folder, string(filepath.Separator), "_"), func(t *testing.T) {
			repo := modeTestGitRepo(t)
			if err := Init(repo, folder); err != nil {
				t.Fatalf("Init(%q): %v", folder, err)
			}

			dest := filepath.Join(repo, folder)
			modeTestRequireFile(t, filepath.Join(dest, "CONTINUE.md"))
			for _, absent := range []string{"config.json", ".gitignore", "local"} {
				if _, err := os.Lstat(filepath.Join(dest, absent)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("new-layout path %s exists for legacy folder %q: %v", absent, folder, err)
				}
			}
			state, err := loadScaffoldState(dest)
			if err != nil || !state.Legacy {
				t.Fatalf("legacy state for %q = %+v, err=%v", folder, state, err)
			}
			checks, err := Doctor(repo, folder)
			if err != nil {
				t.Fatalf("Doctor(%q): %v", folder, err)
			}
			for _, check := range checks {
				if !check.OK {
					t.Errorf("Doctor check %q failed for %q: %s", check.Name, folder, check.Detail)
				}
			}
		})
	}
}

func TestInitLegacyCompatibilityRejectsTraversalAndSymlinkParents(t *testing.T) {
	t.Run("traversal", func(t *testing.T) {
		root := t.TempDir()
		repo := filepath.Join(root, "repo")
		if err := os.Mkdir(repo, 0o755); err != nil {
			t.Fatal(err)
		}
		modeTestGitInit(t, repo)
		if err := Init(repo, filepath.Join("..", "outside")); err == nil {
			t.Fatal("Init accepted traversal outside the repository")
		}
		if _, err := os.Lstat(filepath.Join(root, "outside")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("traversal created an outside path: %v", err)
		}
	})

	t.Run("symlink parent", func(t *testing.T) {
		repo := modeTestGitRepo(t)
		outside := t.TempDir()
		linkedParent := filepath.Join(repo, "linked")
		if err := os.Symlink(outside, linkedParent); err != nil {
			t.Skipf("symbolic links unavailable: %v", err)
		}
		err := Init(repo, filepath.Join("linked", "ctx"))
		if err == nil || !strings.Contains(err.Error(), "symbolic-link parent") {
			t.Fatalf("Init through symlink parent error = %v", err)
		}
		if _, err := os.Lstat(filepath.Join(outside, "ctx")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("symlink-parent init wrote outside the repository: %v", err)
		}
	})
}

func TestInitLegacyCompatibilityCleansMissingParentsOnPrivacyFailure(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("!/docs/ctx/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join("docs", "ctx")
	if err := Init(repo, folder); err == nil {
		t.Fatal("Init succeeded despite an overriding visibility rule")
	}
	if _, err := os.Lstat(filepath.Join(repo, "docs")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed legacy init left its created parent behind: %v", err)
	}
}

func TestInitLegacyCompatibilityPrivacyFailureRollsBack(t *testing.T) {
	repo := modeTestGitRepo(t)
	rules := "!/.ctx/\n/.ctx/*\n!/.ctx/future-private-state\n"
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(rules), 0o644); err != nil {
		t.Fatal(err)
	}
	exclude := filepath.Join(repo, ".git", "info", "exclude")
	before, err := os.ReadFile(exclude)
	if err != nil {
		t.Fatal(err)
	}

	err = Init(repo, ".ctx")
	if err == nil || !strings.Contains(err.Error(), "as a directory") {
		t.Fatalf("Init error = %v, want privacy postcondition failure", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ".ctx")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed compatibility init published a scaffold: %v", statErr)
	}
	after, err := os.ReadFile(exclude)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("failed compatibility init did not restore info/exclude\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
