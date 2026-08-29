package ctx

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Init scaffolds <repo>/<folder>/ from the embedded templates, substitutes
// {{PROJECT}}/{{DATE}} placeholders, writes a .ctx-version stamp, and adds the
// folder to the repo's .git/info/exclude. It refuses to overwrite an existing folder.
func Init(repo, folder string) error {
	if _, err := os.Stat(repo); err != nil {
		return fmt.Errorf("target repo: %w", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return fmt.Errorf("target %s is not a git repository (no .git)", repo)
	}
	dest := filepath.Join(repo, folder)
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("%s already exists — refusing to overwrite; remove it first for a fresh scaffold", dest)
	}

	abs, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	project := filepath.Base(abs)
	date := time.Now().Format("2006-01-02")

	tfs := TemplateFS()
	if err := fs.WalkDir(tfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, rerr := fs.ReadFile(tfs, path)
		if rerr != nil {
			return rerr
		}
		content := substitute(string(data), project, date)
		out := filepath.Join(dest, path)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, []byte(content), 0o644)
	}); err != nil {
		return err
	}

	// version stamp so `ctx update` (future) can tell which scaffold version produced this.
	if err := os.WriteFile(filepath.Join(dest, ".ctx-version"), []byte(Version+"\n"), 0o644); err != nil {
		return err
	}

	if err := ensureExcluded(repo, folder); err != nil {
		return err
	}
	return nil
}

func substitute(s, project, date string) string {
	s = strings.ReplaceAll(s, "{{PROJECT}}", project)
	s = strings.ReplaceAll(s, "{{DATE}}", date)
	return s
}