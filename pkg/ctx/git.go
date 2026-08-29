package ctx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveGitDir returns the .git directory for a repo. For a normal repo it is
// <repo>/.git; for worktrees/submodules .git is a gitfile pointing elsewhere.
func resolveGitDir(repo string) (string, error) {
	g := filepath.Join(repo, ".git")
	info, err := os.Stat(g)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return g, nil
	}
	// gitfile: "gitdir: <path>"
	b, err := os.ReadFile(g)
	if err != nil {
		return "", err
	}
	dir, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir: ")
	if !ok {
		return "", fmt.Errorf("unrecognized gitfile at %s", g)
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repo, dir)
	}
	return dir, nil
}

// ensureExcluded appends folder to <gitdir>/info/exclude (idempotent). It never
// touches the repo's .gitignore — the folder stays private and repo-local.
func ensureExcluded(repo, folder string) error {
	gd, err := resolveGitDir(repo)
	if err != nil {
		return fmt.Errorf("resolve .git: %w", err)
	}
	exclude := filepath.Join(gd, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		return err
	}
	if data, err := os.ReadFile(exclude); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimRight(strings.TrimSpace(line), "/") == folder {
				return nil // already excluded
			}
		}
	}
	f, err := os.OpenFile(exclude, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "\n# agent-context working folder (private, repo-local)\n%s/\n", folder); err != nil {
		return err
	}
	return nil
}