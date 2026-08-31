package ctx

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Check is one doctor finding.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

func pass(name, detail string) Check { return Check{Name: name, OK: true, Detail: detail} }
func fail(name, detail string) Check { return Check{Name: name, OK: false, Detail: detail} }

var durableExpectedFiles = []string{
	"README.md", "OPERATING.md", "INDEX.md", "REVIEW.md",
	"context/overview.md", "context/architecture.md", "context/format.md",
	"context/extending.md", "context/known-issues.md", "context/glossary.md",
}

func teamVisibleFiles() []string {
	files := append([]string(nil), durableExpectedFiles...)
	return append(files, ".gitignore", configFileName, ".ctx-version")
}

func expectedFilesFor(state scaffoldState) []string {
	files := append([]string(nil), durableExpectedFiles...)
	if state.Legacy {
		return append(files, "CONTINUE.md")
	}
	return append(files, ".gitignore", configFileName, state.continuePath())
}

// Doctor validates scaffold integrity and verifies that Git visibility matches
// the configured audience. Config-less legacy scaffolds remain local and keep a
// root CONTINUE.md; doctor never converts them.
func Doctor(repo, folder string) ([]Check, error) {
	if err := validateExistingFolderPath(folder); err != nil {
		return nil, err
	}
	dest := filepath.Join(repo, folder)
	var checks []Check

	// 1. folder exists
	if info, err := os.Lstat(dest); err != nil || !info.IsDir() {
		return []Check{fail(folder+"/ folder exists", fmt.Sprintf("no %s; run `ctx init --folder %q` in the target repo", dest, folder))}, nil
	}
	checks = append(checks, pass(folder+"/ folder exists", dest))

	// 2. scaffold configuration and mode
	state, stateErr := loadScaffoldState(dest)
	if stateErr != nil {
		checks = append(checks, fail("scaffold config", stateErr.Error()))
		state = scaffoldState{Config: Config{SchemaVersion: currentSchemaVersion, Mode: ModeTeam}}
	} else if state.Legacy {
		checks = append(checks, pass("scaffold mode", state.modeLabel()))
	} else {
		checks = append(checks, pass("scaffold mode", fmt.Sprintf("%s (schema %d)", state.Config.Mode, state.Config.SchemaVersion)))
	}

	// 3. CLI version stamp
	if v, err := os.ReadFile(filepath.Join(dest, ".ctx-version")); err == nil {
		checks = append(checks, pass(".ctx-version stamp", strings.TrimSpace(string(v))))
	} else {
		checks = append(checks, fail(".ctx-version stamp", "missing or unreadable"))
	}

	// 4. configured Git visibility
	checks = append(checks, visibilityChecks(repo, folder, state)...)

	// 5. no leftover init-substituted placeholders in Markdown files
	var leftover []string
	walkErr := fs.WalkDir(os.DirFS(dest), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		b, err := os.ReadFile(filepath.Join(dest, path))
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "{{PROJECT}}") || strings.Contains(string(b), "{{DATE}}") ||
			strings.Contains(string(b), "{{FOLDER}}") || strings.Contains(string(b), "{{MODE}}") ||
			strings.Contains(string(b), "{{CONTINUE_PATH}}") {
			leftover = append(leftover, path)
		}
		return nil
	})
	if walkErr != nil {
		checks = append(checks, fail("read Markdown files", walkErr.Error()))
	} else if len(leftover) == 0 {
		checks = append(checks, pass("no init-placeholder leftovers", ""))
	} else {
		checks = append(checks, fail("no init-placeholder leftovers", strings.Join(leftover, ", ")))
	}

	// 6. managed marker grammar in platform-managed files
	for _, name := range managedFiles {
		b, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			checks = append(checks, fail("markers balanced in "+name, err.Error()))
			continue
		}
		if !markersBalanced(string(b)) {
			checks = append(checks, fail("markers balanced in "+name, "unbalanced <!-- ctx:managed --> begin/end"))
		} else {
			checks = append(checks, pass("markers balanced in "+name, ""))
		}
	}

	// 7. expected paths are present as regular files
	expectedFiles := expectedFilesFor(state)
	var invalid []string
	for _, name := range expectedFiles {
		info, err := os.Stat(filepath.Join(dest, name))
		if err != nil || !info.Mode().IsRegular() {
			invalid = append(invalid, name)
		}
	}
	if len(invalid) == 0 {
		checks = append(checks, pass("expected files present", fmt.Sprintf("%d regular files", len(expectedFiles))))
	} else {
		detail := "missing or invalid: " + strings.Join(invalid, ", ")
		if !state.Legacy && state.Config.Mode == ModeTeam && containsString(invalid, state.continuePath()) {
			detail += fmt.Sprintf("; run `ctx init --folder %q` to hydrate local state", folder)
		}
		checks = append(checks, fail("expected files present", detail))
	}

	return checks, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func visibilityChecks(repo, folder string, state scaffoldState) []Check {
	if state.Config.Mode == ModeTeam && !state.Legacy {
		var checks []Check
		sharedFiles, walkErr := teamSharedPaths(filepath.Join(repo, folder))
		ignoredShared := make([]string, 0)
		if walkErr != nil {
			checks = append(checks, fail("shared context Git-visible", walkErr.Error()))
			sharedFiles = nil
			ignoredShared = nil
		}
		for _, name := range sharedFiles {
			sharedPath := filepath.Join(folder, name)
			ignored, err := gitCheckIgnored(repo, sharedPath)
			if err != nil {
				checks = append(checks, fail("shared context Git-visible", err.Error()))
				ignoredShared = nil
				break
			}
			if ignored {
				ignoredShared = append(ignoredShared, filepath.ToSlash(sharedPath))
			}
		}
		if ignoredShared != nil {
			if len(ignoredShared) > 0 {
				checks = append(checks, fail("shared context Git-visible", "ignored: "+strings.Join(ignoredShared, ", ")))
			} else {
				checks = append(checks, pass("shared context Git-visible", folder+"/"))
			}
		}

		if err := verifyTeamLocalPrivate(repo, filepath.Join(repo, folder), folder, state); err != nil {
			checks = append(checks, fail("local state private", err.Error()))
		} else {
			checks = append(checks, pass("local state private", filepath.ToSlash(filepath.Join(folder, "local"))+"/ is ignored and untracked"))
		}
		return checks
	}

	var checks []Check
	hasExclude, err := hasFolderExclusion(repo, folder)
	if err != nil {
		checks = append(checks, fail("repo-local folder exclusion", err.Error()))
	} else if hasExclude {
		checks = append(checks, pass("repo-local folder exclusion", excludePattern(folder)))
	} else {
		checks = append(checks, fail("repo-local folder exclusion", excludePattern(folder)+" is missing from Git's common info/exclude"))
	}

	tracked, trackErr := gitTrackedFiles(repo, folder)
	if trackErr != nil {
		checks = append(checks, fail("local context untracked", trackErr.Error()))
	} else if len(tracked) > 0 {
		checks = append(checks, fail("local context untracked", "tracked: "+strings.Join(tracked, ", ")))
	} else {
		checks = append(checks, pass("local context untracked", folder+"/"))
	}

	paths, pathsErr := privacyPaths(filepath.Join(repo, folder), ".", append(expectedFilesFor(state), ".ctx-version", ".ctx-local-mode-probe")...)
	if pathsErr != nil {
		checks = append(checks, fail("local context ignored", pathsErr.Error()))
		return checks
	}
	var visible []string
	ignoredRoot, rootErr := gitCheckIgnored(repo, folder)
	if rootErr != nil {
		checks = append(checks, fail("local context ignored", rootErr.Error()))
		return checks
	}
	if !ignoredRoot {
		visible = append(visible, filepath.ToSlash(folder)+"/ (directory)")
	}
	for _, name := range paths {
		relPath := filepath.Join(folder, name)
		ignored, checkErr := gitCheckIgnored(repo, relPath)
		if checkErr != nil {
			checks = append(checks, fail("local context ignored", checkErr.Error()))
			return checks
		}
		if !ignored {
			visible = append(visible, filepath.ToSlash(relPath))
		}
	}
	if len(visible) > 0 {
		checks = append(checks, fail("local context ignored", "visible: "+strings.Join(visible, ", ")))
	} else {
		checks = append(checks, pass("local context ignored", folder+"/"))
	}
	return checks
}

func teamSharedPaths(dest string) ([]string, error) {
	seen := make(map[string]bool)
	var paths []string
	add := func(path string) {
		path = filepath.ToSlash(path)
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	for _, path := range teamVisibleFiles() {
		add(path)
	}
	err := fs.WalkDir(os.DirFS(dest), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.ToSlash(path) == "local" {
				return fs.SkipDir
			}
			return nil
		}
		add(path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inspect shared paths: %w", err)
	}
	return paths, nil
}

func verifyTeamLocalPrivate(repo, dest, folder string, state scaffoldState) error {
	b, err := os.ReadFile(filepath.Join(dest, ".gitignore"))
	if err != nil {
		return fmt.Errorf("read scaffold .gitignore: %w", err)
	}
	if !containsExcludeLine(string(b), "local") {
		return fmt.Errorf("scaffold .gitignore must contain an effective /local/ rule")
	}
	tracked, err := gitTrackedFiles(repo, filepath.Join(folder, "local"))
	if err != nil {
		return err
	}
	if len(tracked) > 0 {
		return fmt.Errorf("tracked local state: %s", strings.Join(tracked, ", "))
	}
	localDirPath := filepath.Join(dest, "local")
	localInfo, err := os.Lstat(localDirPath)
	if err != nil {
		return fmt.Errorf("inspect local state directory: %w", err)
	}
	if !localInfo.IsDir() {
		return fmt.Errorf("local state path is not a directory")
	}
	localDir := filepath.ToSlash(filepath.Join(folder, "local"))
	ignoredDir, err := gitCheckIgnored(repo, localDir)
	if err != nil {
		return err
	}
	if !ignoredDir {
		return fmt.Errorf("%s/ is visible to Git", localDir)
	}
	paths, err := privacyPaths(dest, "local", state.continuePath(), filepath.ToSlash(filepath.Join("local", ".ctx-ignore-probe")))
	if err != nil {
		return err
	}
	for _, name := range paths {
		relPath := filepath.Join(folder, filepath.FromSlash(name))
		ignored, err := gitCheckIgnored(repo, relPath)
		if err != nil {
			return err
		}
		if !ignored {
			return fmt.Errorf("%s is visible to Git", filepath.ToSlash(relPath))
		}
	}
	return nil
}

// privacyPaths returns seeded paths plus every existing non-directory path
// beneath walkRoot, relative to the scaffold. Probes cover future files while
// the walk catches path-specific negations and force-added local state.
func privacyPaths(dest, walkRoot string, seeds ...string) ([]string, error) {
	seen := make(map[string]bool)
	var paths []string
	add := func(path string) {
		path = filepath.ToSlash(path)
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	for _, seed := range seeds {
		add(seed)
	}
	root := filepath.Join(dest, filepath.FromSlash(walkRoot))
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if walkRoot == "." {
			add(path)
		} else {
			add(filepath.Join(walkRoot, path))
		}
		return nil
	})
	if os.IsNotExist(err) {
		return paths, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect private paths: %w", err)
	}
	return paths, nil
}

func containsExcludeLine(exclude, folder string) bool {
	folder = filepath.ToSlash(folder)
	matched := false
	for _, line := range strings.Split(exclude, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		negated := strings.HasPrefix(line, "!")
		line = strings.TrimPrefix(line, "!")
		line = strings.TrimPrefix(strings.TrimRight(line, "/"), "/")
		if line == folder {
			matched = !negated
		}
	}
	return matched
}
