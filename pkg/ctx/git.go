package ctx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
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

// resolveCommonGitDir returns the repository's common Git directory. In a
// linked worktree, resolveGitDir points at .git/worktrees/<name>, while Git's
// info/exclude lives in the common directory named by the commondir file.
func resolveCommonGitDir(repo string) (string, error) {
	gd, err := resolveGitDir(repo)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(filepath.Join(gd, "commondir"))
	if os.IsNotExist(err) {
		return gd, nil
	}
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(string(b))
	if dir == "" {
		return "", fmt.Errorf("empty commondir in %s", gd)
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(gd, dir)
	}
	return filepath.Clean(dir), nil
}

func acquireInitLock(repo string) (func() error, error) {
	gd, err := resolveCommonGitDir(repo)
	if err != nil {
		return nil, fmt.Errorf("resolve Git directory for init lock: %w", err)
	}
	lockDir := filepath.Join(gd, "ctx-locks")
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return nil, fmt.Errorf("create init lock directory: %w", err)
	}
	release, err := acquireFileLock(filepath.Join(lockDir, "init"))
	if err != nil {
		return nil, fmt.Errorf("acquire repository init lock: %w", err)
	}
	return release, nil
}

func excludePattern(folder string) string {
	return "/" + filepath.ToSlash(folder) + "/"
}

type excludeChange struct {
	path    string
	before  []byte
	after   []byte
	mode    os.FileMode
	existed bool
	changed bool
}

func (c excludeChange) rollback() error {
	if !c.changed {
		return nil
	}
	current, err := os.ReadFile(c.path)
	if err != nil {
		return fmt.Errorf("read exclude for rollback: %w", err)
	}
	if !bytes.Equal(current, c.after) {
		return fmt.Errorf("refusing to roll back %s because it changed concurrently", c.path)
	}
	if !c.existed {
		if err := os.Remove(c.path); err != nil {
			return fmt.Errorf("remove new exclude file: %w", err)
		}
		return nil
	}
	if err := os.WriteFile(c.path, c.before, c.mode.Perm()); err != nil {
		return fmt.Errorf("restore exclude file: %w", err)
	}
	return nil
}

// ensureExcluded appends folder to <gitdir>/info/exclude (idempotent). It never
// touches the repo's .gitignore — the folder stays private and repo-local.
func ensureExcluded(repo, folder string) error {
	_, err := addFolderExclusion(repo, folder)
	return err
}

// addFolderExclusion adds the local-mode rule and returns enough state to undo
// only this invocation's change if initialization cannot meet its privacy
// postcondition. The rollback refuses to overwrite concurrent edits.
func addFolderExclusion(repo, folder string) (excludeChange, error) {
	gd, err := resolveCommonGitDir(repo)
	if err != nil {
		return excludeChange{}, fmt.Errorf("resolve .git: %w", err)
	}
	exclude := filepath.Join(gd, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		return excludeChange{}, err
	}
	change := excludeChange{path: exclude, mode: 0o644}
	data, readErr := os.ReadFile(exclude)
	if readErr == nil {
		change.before = data
		change.existed = true
		if info, statErr := os.Stat(exclude); statErr == nil {
			change.mode = info.Mode()
		} else {
			return excludeChange{}, statErr
		}
	} else if !os.IsNotExist(readErr) {
		return excludeChange{}, readErr
	}
	if containsExcludeLine(string(change.before), folder) {
		return change, nil
	}

	addition := []byte(fmt.Sprintf("\n# ctx local-mode context (repo-local)\n%s\n", excludePattern(folder)))
	change.after = append(append([]byte(nil), change.before...), addition...)
	f, err := os.OpenFile(exclude, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return excludeChange{}, err
	}
	if !change.existed {
		change.changed = true // O_CREATE has already created the file.
		change.after = []byte{}
	}
	n, writeErr := f.Write(addition)
	if n > 0 {
		change.changed = true
		change.after = append(append([]byte(nil), change.before...), addition[:n]...)
	}
	closeErr := f.Close()
	if writeErr == nil && n != len(addition) {
		writeErr = fmt.Errorf("short write to %s", exclude)
	}
	if writeErr != nil {
		if closeErr != nil {
			writeErr = fmt.Errorf("%w; close exclude: %v", writeErr, closeErr)
		}
		return excludeChange{}, rollbackExcludeWrite(change, writeErr)
	}
	if closeErr != nil {
		return excludeChange{}, rollbackExcludeWrite(change, closeErr)
	}
	change.after = append(append([]byte(nil), change.before...), addition...)
	return change, nil
}

func rollbackExcludeWrite(change excludeChange, cause error) error {
	if !change.changed {
		return cause
	}
	if err := change.rollback(); err != nil {
		return fmt.Errorf("%w; also failed to roll back Git exclusion: %v", cause, err)
	}
	return cause
}

func hasFolderExclusion(repo, folder string) (bool, error) {
	gd, err := resolveCommonGitDir(repo)
	if err != nil {
		return false, err
	}
	b, err := os.ReadFile(filepath.Join(gd, "info", "exclude"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return containsExcludeLine(string(b), folder), nil
}

// gitCheckIgnored asks Git whether relPath is effectively ignored, including
// repository, info/exclude, global, and worktree rules. --no-index makes Git
// evaluate ignore rules without consulting tracked state; callers check the
// index separately where tracked files matter.
func gitCheckIgnored(repo, relPath string) (bool, error) {
	cmd := gitCommand(repo, "check-ignore", "-q", "--no-index", "--", filepath.ToSlash(relPath))
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git check-ignore %s: %w", relPath, err)
}

// gitTrackedFiles returns index entries at or below relPath. Ignore rules do
// not make tracked files private, so callers must check both states.
func gitTrackedFiles(repo, relPath string) ([]string, error) {
	pathspec := ":(literal)" + filepath.ToSlash(relPath)
	cmd := gitCommand(repo, "ls-files", "-z", "--", pathspec)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files %s: %w", relPath, err)
	}
	var paths []string
	for _, item := range bytes.Split(out, []byte{0}) {
		if len(item) > 0 {
			paths = append(paths, string(item))
		}
	}
	return paths, nil
}

func gitWorktreePaths(repo string) ([]string, error) {
	cmd := gitCommand(repo, "worktree", "list", "--porcelain", "-z")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	var paths []string
	for _, field := range bytes.Split(out, []byte{0}) {
		if path, ok := strings.CutPrefix(string(field), "worktree "); ok {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

var gitRepositoryEnv = map[string]bool{
	"GIT_COMMON_DIR":         true,
	"GIT_DIR":                true,
	"GIT_IMPLICIT_WORK_TREE": true,
	"GIT_INDEX_FILE":         true,
	"GIT_OBJECT_DIRECTORY":   true,
	"GIT_PREFIX":             true,
	"GIT_WORK_TREE":          true,
}

func gitCommand(repo string, args ...string) *exec.Cmd {
	cmdArgs := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", cmdArgs...)
	cmd.Env = make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !gitRepositoryEnv[strings.ToUpper(key)] {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	return cmd
}
