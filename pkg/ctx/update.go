package ctx

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// managedFiles are the files ctx update refreshes (managed-block swap). All
// other .ctx/ files are user-owned and never touched by update.
var managedFiles = []string{"README.md", "REVIEW.md"}

// Update refreshes the managed blocks in <repo>/<folder>/ managedFiles from the
// embedded templates, preserves all user content, and bumps .ctx-version.
// Files without markers (user took ownership) or missing files are skipped.
func Update(repo, folder string) ([]string, error) {
	dest := filepath.Join(repo, folder)
	if _, err := os.Stat(dest); err != nil {
		return nil, fmt.Errorf("no %s here; run `ctx init` first", dest)
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	project := filepath.Base(abs)
	date := time.Now().Format("2006-01-02")
	tfs := TemplateFS()

	var touched []string
	for _, name := range managedFiles {
		tmpl, err := fs.ReadFile(tfs, name)
		if err != nil {
			return touched, fmt.Errorf("read embedded template %s: %w", name, err)
		}
		tmplStr := substitute(string(tmpl), project, date)

		existingPath := filepath.Join(dest, name)
		existing, err := os.ReadFile(existingPath)
		if err != nil {
			continue // user deleted it; skip
		}
		if !hasManaged(string(existing)) {
			continue // user took ownership (no markers); skip
		}
		updated, existingN, newN := updateManagedContent(string(existing), tmplStr)
		if existingN != newN {
			// Not fatal — refresh what we can, note the mismatch for the caller.
			touched = append(touched, fmt.Sprintf("%s (managed blocks %d→%d; refreshed matching, left extras)", name, existingN, newN))
		} else {
			touched = append(touched, name)
		}
		if err := os.WriteFile(existingPath, []byte(updated), 0o644); err != nil {
			return touched, err
		}
	}

	if err := os.WriteFile(filepath.Join(dest, ".ctx-version"), []byte(Version+"\n"), 0o644); err != nil {
		return touched, err
	}
	return touched, nil
}