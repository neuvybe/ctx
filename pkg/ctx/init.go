package ctx

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Init preserves the original API's whole-folder-local behavior, safe contained
// custom paths, and legacy layout: a root CONTINUE.md, no config.json or
// scaffold .gitignore, and a whole-folder rule in the repository-local Git
// exclude file. The CLI uses InitWithOptions and defaults to team mode.
//
// Deprecated: use InitWithOptions and choose ModeTeam or ModeLocal explicitly.
func Init(repo, folder string) error {
	if err := validateExistingFolderPath(folder); err != nil {
		return err
	}
	return initLegacy(repo, folder)
}

// initLegacy is the compatibility creation path for the original exported
// Init API. It intentionally emits a config-less scaffold so Update and Doctor
// continue to recognize it as a pre-schema local scaffold.
func initLegacy(repo, folder string) error {
	if _, err := os.Stat(repo); err != nil {
		return fmt.Errorf("target repo: %w", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return fmt.Errorf("target %s is not a git repository (no .git)", repo)
	}
	releaseLock, err := acquireInitLock(repo)
	if err != nil {
		return err
	}
	defer func() { _ = releaseLock() }()

	dest := filepath.Join(repo, folder)
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("%s already exists — refusing to overwrite", dest)
	} else if !os.IsNotExist(err) {
		return err
	}

	tracked, err := gitTrackedFiles(repo, folder)
	if err != nil {
		return err
	}
	if len(tracked) > 0 {
		return fmt.Errorf("%s has tracked index entries but is absent from the worktree; restore or remove them before initializing", folder)
	}
	if err := rejectTeamModeInOtherWorktrees(repo, folder); err != nil {
		return err
	}
	parent, createdParents, err := prepareLegacyParentPath(repo, folder)
	if err != nil {
		return err
	}
	keepParents := false
	defer func() {
		if !keepParents {
			removeEmptyLegacyParents(createdParents)
		}
	}()

	abs, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	state := scaffoldState{Config: Config{Mode: ModeLocal}, Legacy: true}
	values := templateValues{
		Project:      filepath.Base(abs),
		Date:         time.Now().Format("2006-01-02"),
		Folder:       folder,
		Mode:         state.modeLabel(),
		ContinuePath: state.continuePath(),
	}

	// Stage beside the final destination so publication remains an atomic rename
	// even when a legacy nested parent is a separate mounted filesystem.
	stageRoot, err := os.MkdirTemp(parent, ".ctx-init-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(stageRoot) }()
	// Keep an interrupted staging namespace invisible to Git, just like current
	// InitWithOptions creation.
	if err := os.WriteFile(filepath.Join(stageRoot, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		return err
	}
	stage := filepath.Join(stageRoot, "scaffold")
	if err := os.Mkdir(stage, 0o755); err != nil {
		return err
	}

	tfs := TemplateFS()
	if err := fs.WalkDir(tfs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		outPath := path
		if filepath.ToSlash(path) == "local/CONTINUE.md" {
			outPath = "CONTINUE.md"
		} else if path == configFileName || path == ".gitignore" || strings.HasPrefix(filepath.ToSlash(path), "local/") {
			// Schema metadata and the local/ namespace belong only to the new
			// InitWithOptions layout.
			return nil
		}

		data, readErr := fs.ReadFile(tfs, path)
		if readErr != nil {
			return readErr
		}
		out := filepath.Join(stage, filepath.FromSlash(outPath))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, []byte(renderTemplate(string(data), values)), 0o644)
	}); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stage, ".ctx-version"), []byte(Version+"\n"), 0o644); err != nil {
		return err
	}

	exclusion, err := addFolderExclusion(repo, folder)
	if err != nil {
		return err
	}
	if err := rejectTeamModeInOtherWorktrees(repo, folder); err != nil {
		return withExcludeRollback(err, exclusion)
	}
	if err := verifyLocalDestinationDirectoryPrivate(repo, dest, folder); err != nil {
		return withExcludeRollback(err, exclusion)
	}
	if err := verifyLocalScaffoldPrivate(repo, folder, state); err != nil {
		return withExcludeRollback(err, exclusion)
	}
	if err := validateLegacyParentPath(repo, folder); err != nil {
		return withExcludeRollback(err, exclusion)
	}
	if err := os.Rename(stage, dest); err != nil {
		return withExcludeRollback(fmt.Errorf("publish scaffold: %w", err), exclusion)
	}
	keepParents = true
	return nil
}

// prepareLegacyParentPath creates only missing directory components below repo
// for the deprecated Init API. Existing symlinks and non-directories are
// rejected so a historically accepted nested path cannot escape the repository.
func prepareLegacyParentPath(repo, folder string) (string, []string, error) {
	parentRel := filepath.Dir(folder)
	if parentRel == "." {
		return repo, nil, nil
	}

	current := repo
	var created []string
	for _, part := range strings.Split(parentRel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0o755); err != nil {
				removeEmptyLegacyParents(created)
				return "", nil, fmt.Errorf("create legacy scaffold parent %s: %w", current, err)
			}
			created = append(created, current)
			continue
		}
		if err != nil {
			removeEmptyLegacyParents(created)
			return "", nil, fmt.Errorf("inspect legacy scaffold parent %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			removeEmptyLegacyParents(created)
			return "", nil, fmt.Errorf("refusing legacy scaffold through symbolic-link parent %s", current)
		}
		if !info.IsDir() {
			removeEmptyLegacyParents(created)
			return "", nil, fmt.Errorf("legacy scaffold parent %s is not a directory", current)
		}
	}
	return current, created, nil
}

func validateLegacyParentPath(repo, folder string) error {
	parentRel := filepath.Dir(folder)
	if parentRel == "." {
		return nil
	}
	current := repo
	for _, part := range strings.Split(parentRel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("reinspect legacy scaffold parent %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing legacy scaffold through symbolic-link parent %s", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("legacy scaffold parent %s is not a directory", current)
		}
	}
	return nil
}

func removeEmptyLegacyParents(paths []string) {
	for i := len(paths) - 1; i >= 0; i-- {
		// Remove only empty directories created by this invocation. A concurrent
		// writer makes Remove fail safely; user content is never removed.
		_ = os.Remove(paths[i])
	}
}

// InitWithOptions scaffolds <repo>/<folder>/ from the embedded templates. Team
// mode keeps durable docs visible to Git and ignores local/ through the
// scaffold's .gitignore. Local mode additionally excludes the whole folder via
// the repository-local Git exclude file. Creation is staged and published with
// a final rename so failed initialization does not expose a partial scaffold.
// On a fresh clone of a team scaffold, it also hydrates the missing ignored
// local/CONTINUE.md without changing any durable file.
func InitWithOptions(repo string, rawOpts InitOptions) error {
	opts, err := normalizeInitOptions(rawOpts)
	if err != nil {
		return err
	}
	if _, err := os.Stat(repo); err != nil {
		return fmt.Errorf("target repo: %w", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return fmt.Errorf("target %s is not a git repository (no .git)", repo)
	}
	releaseLock, err := acquireInitLock(repo)
	if err != nil {
		return err
	}
	defer func() { _ = releaseLock() }()

	dest := filepath.Join(repo, opts.Folder)
	if info, err := os.Lstat(dest); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s already exists and is not a directory", dest)
		}
		return hydrateTeamLocalState(repo, dest, opts)
	} else if !os.IsNotExist(err) {
		return err
	}

	tracked, err := gitTrackedFiles(repo, opts.Folder)
	if err != nil {
		return err
	}
	if len(tracked) > 0 {
		return fmt.Errorf("%s has tracked index entries but is absent from the worktree; restore or remove them before initializing", opts.Folder)
	}

	if opts.Mode == ModeTeam {
		if err := verifyTeamFilesVisible(repo, opts.Folder); err != nil {
			return err
		}
	} else {
		if err := rejectTeamModeInOtherWorktrees(repo, opts.Folder); err != nil {
			return err
		}
	}

	abs, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	project := filepath.Base(abs)
	date := time.Now().Format("2006-01-02")
	state := scaffoldState{Config: Config{SchemaVersion: currentSchemaVersion, Mode: opts.Mode}}

	stageRoot, err := os.MkdirTemp(repo, ".ctx-init-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(stageRoot) }()
	// Git does not record empty directories. Write this before any scaffold
	// content so a crash can leave only an ignored staging namespace behind.
	if err := os.WriteFile(filepath.Join(stageRoot, ".gitignore"), []byte("*\n"), 0o644); err != nil {
		return err
	}
	stage := filepath.Join(stageRoot, "scaffold")
	if err := os.Mkdir(stage, 0o755); err != nil {
		return err
	}

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
		content := renderTemplate(string(data), templateValues{
			Project:      project,
			Date:         date,
			Folder:       opts.Folder,
			Mode:         state.modeLabel(),
			ContinuePath: state.continuePath(),
		})
		out := filepath.Join(stage, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, []byte(content), 0o644)
	}); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(stage, ".gitignore"), []byte("# Machine-local ctx state. Durable team context remains visible to Git.\n/local/\n"), 0o644); err != nil {
		return err
	}
	if err := writeConfig(stage, opts.Mode); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stage, ".ctx-version"), []byte(Version+"\n"), 0o644); err != nil {
		return err
	}

	var exclusion excludeChange
	if opts.Mode == ModeLocal {
		exclusion, err = addFolderExclusion(repo, opts.Folder)
		if err != nil {
			return err
		}
		if err := rejectTeamModeInOtherWorktrees(repo, opts.Folder); err != nil {
			return withExcludeRollback(err, exclusion)
		}
		if err := verifyLocalDestinationDirectoryPrivate(repo, dest, opts.Folder); err != nil {
			return withExcludeRollback(err, exclusion)
		}
		state := scaffoldState{Config: Config{SchemaVersion: currentSchemaVersion, Mode: ModeLocal}}
		if err := verifyLocalScaffoldPrivate(repo, opts.Folder, state); err != nil {
			return withExcludeRollback(err, exclusion)
		}
	}
	if err := os.Rename(stage, dest); err != nil {
		publishErr := fmt.Errorf("publish scaffold: %w", err)
		if opts.Mode == ModeLocal {
			return withExcludeRollback(publishErr, exclusion)
		}
		return publishErr
	}
	return nil
}

func hydrateTeamLocalState(repo, dest string, opts InitOptions) error {
	state, err := loadScaffoldState(dest)
	if err != nil {
		return fmt.Errorf("%s already exists and cannot be hydrated: %w", dest, err)
	}
	if opts.Mode != ModeTeam || state.Legacy || state.Config.Mode != ModeTeam {
		return fmt.Errorf("%s already exists in %s mode — refusing to overwrite or convert it", dest, state.modeLabel())
	}
	continuePath := filepath.Join(dest, filepath.FromSlash(state.continuePath()))
	if _, err := os.Lstat(continuePath); err == nil {
		return fmt.Errorf("%s already exists — refusing to overwrite", dest)
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Lstat(filepath.Join(dest, "CONTINUE.md")); err == nil {
		return fmt.Errorf("%s has an unexpected root CONTINUE.md; refusing ambiguous hydration", dest)
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, name := range teamVisibleFiles() {
		info, err := os.Lstat(filepath.Join(dest, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("cannot hydrate incomplete team scaffold: %s: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("cannot hydrate incomplete team scaffold: %s is not a regular file", name)
		}
	}
	sharedPaths, err := teamSharedPaths(dest)
	if err != nil {
		return err
	}
	if err := verifyTeamPathsVisible(repo, opts.Folder, sharedPaths); err != nil {
		return err
	}
	localDir := filepath.Join(dest, "local")
	createdLocalDir := false
	if info, err := os.Lstat(localDir); os.IsNotExist(err) {
		if err := os.Mkdir(localDir, 0o755); err != nil {
			return fmt.Errorf("create local state directory: %w", err)
		}
		createdLocalDir = true
	} else if err != nil {
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("cannot hydrate team scaffold: local is not a directory")
	}
	keepLocalDir := !createdLocalDir
	defer func() {
		if !keepLocalDir {
			_ = os.Remove(localDir)
		}
	}()
	if err := verifyTeamLocalPrivate(repo, dest, opts.Folder, state); err != nil {
		return err
	}

	tmpl, err := fs.ReadFile(TemplateFS(), "local/CONTINUE.md")
	if err != nil {
		return fmt.Errorf("read embedded local continuation: %w", err)
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	content := renderTemplate(string(tmpl), templateValues{
		Project:      filepath.Base(abs),
		Date:         time.Now().Format("2006-01-02"),
		Folder:       opts.Folder,
		Mode:         state.modeLabel(),
		ContinuePath: state.continuePath(),
	})
	tmp, err := os.CreateTemp(localDir, ".ctx-continue-*")
	if err != nil {
		return fmt.Errorf("stage local continuation: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Link publishes without replacing a file created by a concurrent process.
	if err := os.Link(tmpPath, continuePath); err != nil {
		return fmt.Errorf("publish local continuation: %w", err)
	}
	keepLocalDir = true
	_ = os.Remove(tmpPath)
	return nil
}

func verifyTeamFilesVisible(repo, folder string) error {
	return verifyTeamPathsVisible(repo, folder, teamVisibleFiles())
}

func verifyTeamPathsVisible(repo, folder string, paths []string) error {
	// A parent, repository, or global rule can hide a path; only Git can
	// evaluate the complete ignore stack reliably.
	for _, name := range paths {
		relPath := filepath.Join(folder, name)
		ignored, err := gitCheckIgnored(repo, relPath)
		if err != nil {
			return fmt.Errorf("verify team-mode Git visibility: %w", err)
		}
		if ignored {
			return fmt.Errorf("%s is ignored by Git; remove the matching ignore rule for team mode or use --mode local", filepath.ToSlash(relPath))
		}
	}
	return nil
}

func verifyLocalScaffoldPrivate(repo, folder string, state scaffoldState) error {
	tracked, err := gitTrackedFiles(repo, folder)
	if err != nil {
		return err
	}
	if len(tracked) > 0 {
		return fmt.Errorf("local mode cannot hide tracked files: %s", strings.Join(tracked, ", "))
	}
	paths := append(expectedFilesFor(state), ".ctx-version", ".ctx-local-mode-probe")
	for _, name := range paths {
		relPath := filepath.Join(folder, name)
		ignored, err := gitCheckIgnored(repo, relPath)
		if err != nil {
			return fmt.Errorf("verify local-mode Git privacy: %w", err)
		}
		if !ignored {
			return fmt.Errorf("local mode could not ignore %s; remove overriding Git ignore negations or use --mode team", filepath.ToSlash(relPath))
		}
	}
	return nil
}

func verifyLocalDestinationDirectoryPrivate(repo, dest, folder string) error {
	// Directory-only ignore rules cannot be evaluated reliably for an absent
	// path. Materialize an empty directory briefly; Git does not track empty
	// directories, and os.Remove refuses to delete it if anything appears inside.
	if err := os.Mkdir(dest, 0o755); err != nil {
		return fmt.Errorf("create local-mode privacy probe: %w", err)
	}
	ignored, checkErr := gitCheckIgnored(repo, folder)
	removeErr := os.Remove(dest)
	if removeErr != nil {
		return fmt.Errorf("remove local-mode privacy probe: %w", removeErr)
	}
	if checkErr != nil {
		return fmt.Errorf("verify local-mode Git privacy: %w", checkErr)
	}
	if !ignored {
		return fmt.Errorf("local mode could not ignore %s/ as a directory; remove overriding Git ignore negations or use --mode team", filepath.ToSlash(folder))
	}
	return nil
}

func withExcludeRollback(cause error, change excludeChange) error {
	if err := change.rollback(); err != nil {
		return fmt.Errorf("%w; also failed to roll back Git exclusion: %v", cause, err)
	}
	return cause
}

func rejectTeamModeInOtherWorktrees(repo, folder string) error {
	worktrees, err := gitWorktreePaths(repo)
	if err != nil {
		return err
	}
	current, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	for _, worktree := range worktrees {
		other, err := filepath.Abs(worktree)
		if err != nil {
			return err
		}
		if filepath.Clean(other) == filepath.Clean(current) {
			continue
		}
		tracked, trackErr := gitTrackedFiles(other, folder)
		if trackErr != nil {
			return fmt.Errorf("inspect sibling worktree index %s: %w", other, trackErr)
		}
		if len(tracked) > 0 {
			return fmt.Errorf("cannot use local mode for %s: sibling worktree %s tracks files in that folder; Git's repo-local exclude is shared by linked worktrees", folder, other)
		}
		otherDest := filepath.Join(other, folder)
		info, statErr := os.Lstat(otherDest)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return fmt.Errorf("inspect sibling worktree %s: %w", otherDest, statErr)
		}
		if !info.IsDir() {
			return fmt.Errorf("sibling worktree has non-directory path %s; cannot safely enable repository-wide local mode", otherDest)
		}
		state, stateErr := loadScaffoldState(otherDest)
		if stateErr != nil {
			return fmt.Errorf("inspect sibling worktree scaffold %s: %w", otherDest, stateErr)
		}
		if !state.Legacy && state.Config.Mode == ModeTeam {
			return fmt.Errorf("cannot use local mode for %s: sibling worktree %s has a team-mode scaffold; Git's repo-local exclude is shared by linked worktrees", folder, other)
		}
	}
	return nil
}

func substitute(s, project, date string) string {
	return renderTemplate(s, templateValues{Project: project, Date: date})
}

type templateValues struct {
	Project      string
	Date         string
	Folder       string
	Mode         string
	ContinuePath string
}

func renderTemplate(s string, values templateValues) string {
	replacer := strings.NewReplacer(
		"{{PROJECT}}", values.Project,
		"{{DATE}}", values.Date,
		"{{FOLDER}}", values.Folder,
		"{{MODE}}", values.Mode,
		"{{CONTINUE_PATH}}", values.ContinuePath,
	)
	return replacer.Replace(s)
}
