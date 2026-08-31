package ctx

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// managedFiles are the files ctx update refreshes (managed-block swap). All
// other .ctx/ files are user-owned and never touched by update.
var managedFiles = []string{"README.md", "REVIEW.md"}

type updateOutput struct {
	name         string
	path         string
	content      []byte
	mode         os.FileMode
	existed      bool
	original     []byte
	originalInfo fs.FileInfo

	stagedPath string
	stagedInfo fs.FileInfo
	backupPath string
	published  bool
}

type inspectedOutput struct {
	exists bool
	data   []byte
	info   fs.FileInfo
}

// Update refreshes the managed blocks in <repo>/<folder>/ managedFiles from the
// embedded templates, preserves all user content, and bumps .ctx-version.
// Files without markers (user took ownership) or missing files are skipped.
// Every intended output is validated before staging begins, and each output is
// atomically replaced without opening the destination for writing.
func Update(repo, folder string) ([]string, error) {
	if err := validateExistingFolderPath(folder); err != nil {
		return nil, err
	}
	dest, err := validateUpdateScaffoldPath(repo, folder)
	if err != nil {
		return nil, err
	}
	state, err := loadScaffoldState(dest)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	values := templateValues{
		Project:      filepath.Base(abs),
		Date:         time.Now().Format("2006-01-02"),
		Folder:       folder,
		Mode:         state.modeLabel(),
		ContinuePath: state.continuePath(),
	}
	tfs := TemplateFS()

	// Preflight every managed output before creating a temp file or changing an
	// intended output. Missing and markerless files retain their skip semantics;
	// every other filesystem or marker-grammar problem is an error.
	var plans []*updateOutput
	var touched []string
	for _, name := range managedFiles {
		path := filepath.Join(dest, name)
		existing, err := inspectUpdateOutput(path, true)
		if err != nil {
			return nil, fmt.Errorf("preflight %s: %w", name, err)
		}
		if !existing.exists {
			continue
		}
		if !markersBalanced(string(existing.data)) {
			return nil, fmt.Errorf("preflight %s: malformed managed-marker grammar", name)
		}
		if !hasManaged(string(existing.data)) {
			continue
		}

		tmpl, err := fs.ReadFile(tfs, name)
		if err != nil {
			return nil, fmt.Errorf("read embedded template %s: %w", name, err)
		}
		tmplStr := renderTemplate(string(tmpl), values)
		if !markersBalanced(tmplStr) || !hasManaged(tmplStr) {
			return nil, fmt.Errorf("embedded template %s has malformed managed-marker grammar", name)
		}
		updated, existingN, newN := updateManagedContent(string(existing.data), tmplStr)
		label := name
		if existingN != newN {
			label = fmt.Sprintf("%s (managed blocks %d→%d; refreshed matching, left extras)", name, existingN, newN)
		}
		touched = append(touched, label)
		plans = append(plans, &updateOutput{
			name:         name,
			path:         path,
			content:      []byte(updated),
			mode:         existing.info.Mode().Perm(),
			existed:      true,
			original:     existing.data,
			originalInfo: existing.info,
		})
	}

	versionPath := filepath.Join(dest, ".ctx-version")
	version, err := inspectUpdateOutput(versionPath, true)
	if err != nil {
		return nil, fmt.Errorf("preflight .ctx-version: %w", err)
	}
	versionMode := os.FileMode(0o644)
	if version.exists {
		versionMode = version.info.Mode().Perm()
	}
	plans = append(plans, &updateOutput{
		name:         ".ctx-version",
		path:         versionPath,
		content:      []byte(Version + "\n"),
		mode:         versionMode,
		existed:      version.exists,
		original:     version.data,
		originalInfo: version.info,
	})

	if err := stageUpdateOutputs(dest, plans); err != nil {
		cleanupUpdateTemps(plans)
		return nil, err
	}
	defer cleanupUpdateTemps(plans)
	if err := validateUpdateOutputsUnchanged(plans); err != nil {
		return nil, err
	}
	if err := publishUpdateOutputs(plans); err != nil {
		return nil, err
	}
	return touched, nil
}

// validateUpdateScaffoldPath rejects a scaffold path that is itself a symlink
// or traverses a symlink below repo. Nested paths remain supported for older
// custom scaffolds, but updates never use them to escape the supplied repo.
func validateUpdateScaffoldPath(repo, folder string) (string, error) {
	current := repo
	for _, part := range strings.Split(folder, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("no %s here; run `ctx init --folder %q` first", filepath.Join(repo, folder), folder)
			}
			return "", fmt.Errorf("inspect scaffold path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to update scaffold through symbolic link %s", current)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("scaffold path %s is not a directory", current)
		}
	}
	return filepath.Join(repo, folder), nil
}

// inspectUpdateOutput distinguishes an allowed missing output from all other
// failures and reads through a descriptor whose identity still matches the
// regular, non-symlink directory entry inspected with Lstat.
func inspectUpdateOutput(path string, allowMissing bool) (inspectedOutput, error) {
	entryInfo, err := os.Lstat(path)
	if err != nil {
		if allowMissing && os.IsNotExist(err) {
			return inspectedOutput{}, nil
		}
		return inspectedOutput{}, fmt.Errorf("inspect %s: %w", path, err)
	}
	if entryInfo.Mode()&os.ModeSymlink != 0 {
		return inspectedOutput{}, fmt.Errorf("refusing symbolic-link output %s", path)
	}
	if !entryInfo.Mode().IsRegular() {
		return inspectedOutput{}, fmt.Errorf("output %s is not a regular file", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return inspectedOutput{}, fmt.Errorf("read %s: %w", path, err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
	}()
	openedInfo, err := f.Stat()
	if err != nil {
		return inspectedOutput{}, fmt.Errorf("inspect open output %s: %w", path, err)
	}
	currentInfo, err := os.Lstat(path)
	if err != nil {
		return inspectedOutput{}, fmt.Errorf("reinspect %s: %w", path, err)
	}
	if currentInfo.Mode()&os.ModeSymlink != 0 || !currentInfo.Mode().IsRegular() || !os.SameFile(openedInfo, currentInfo) {
		return inspectedOutput{}, fmt.Errorf("output %s changed during preflight", path)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return inspectedOutput{}, fmt.Errorf("read %s: %w", path, err)
	}
	closeErr := f.Close()
	closed = true
	if closeErr != nil {
		return inspectedOutput{}, fmt.Errorf("close %s after reading: %w", path, closeErr)
	}
	return inspectedOutput{exists: true, data: data, info: openedInfo}, nil
}

func stageUpdateOutputs(dest string, plans []*updateOutput) error {
	for _, plan := range plans {
		path, info, err := stageUpdateFile(dest, plan.name, plan.content, plan.mode)
		if err != nil {
			return fmt.Errorf("stage %s: %w", plan.name, err)
		}
		plan.stagedPath = path
		plan.stagedInfo = info
		if !plan.existed {
			continue
		}
		backup, _, err := stageUpdateFile(dest, plan.name+"-backup", plan.original, plan.originalInfo.Mode().Perm())
		if err != nil {
			return fmt.Errorf("stage rollback for %s: %w", plan.name, err)
		}
		plan.backupPath = backup
	}
	return nil
}

func stageUpdateFile(dest, name string, content []byte, mode os.FileMode) (string, fs.FileInfo, error) {
	prefix := ".ctx-update-" + strings.TrimPrefix(filepath.Base(name), ".") + "-*"
	f, err := os.CreateTemp(dest, prefix)
	if err != nil {
		return "", nil, err
	}
	path := f.Name()
	cleanup := func(cause error) (string, fs.FileInfo, error) {
		_ = f.Close()
		_ = os.Remove(path)
		return "", nil, cause
	}
	if err := f.Chmod(mode.Perm()); err != nil {
		return cleanup(err)
	}
	n, err := f.Write(content)
	if err != nil {
		return cleanup(err)
	}
	if n != len(content) {
		return cleanup(fmt.Errorf("short write: wrote %d of %d bytes", n, len(content)))
	}
	info, err := f.Stat()
	if err != nil {
		return cleanup(err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, err
	}
	return path, info, nil
}

func validateUpdateOutputsUnchanged(plans []*updateOutput) error {
	for _, plan := range plans {
		if err := validateUpdateOutputUnchanged(plan); err != nil {
			return fmt.Errorf("pre-publish %s: %w", plan.name, err)
		}
	}
	return nil
}

func validateUpdateOutputUnchanged(plan *updateOutput) error {
	current, err := inspectUpdateOutput(plan.path, !plan.existed)
	if err != nil {
		return err
	}
	if !plan.existed {
		if current.exists {
			return fmt.Errorf("output %s appeared during update", plan.path)
		}
		return nil
	}
	if !current.exists {
		return fmt.Errorf("output %s disappeared during update", plan.path)
	}
	if !os.SameFile(plan.originalInfo, current.info) || current.info.Mode() != plan.originalInfo.Mode() || !bytes.Equal(current.data, plan.original) {
		return fmt.Errorf("output %s changed during update", plan.path)
	}
	return nil
}

func publishUpdateOutputs(plans []*updateOutput) error {
	for _, plan := range plans {
		// Recheck each output immediately before atomically replacing its final
		// directory entry rather than opening or following the destination.
		if err := validateUpdateOutputUnchanged(plan); err != nil {
			return rollbackPublishedUpdates(plans, fmt.Errorf("publish %s: %w", plan.name, err))
		}
		var err error
		if plan.existed {
			err = atomicReplace(plan.stagedPath, plan.path)
		} else {
			// Link supplies no-replace publication for a missing version stamp.
			err = os.Link(plan.stagedPath, plan.path)
		}
		if err != nil {
			return rollbackPublishedUpdates(plans, fmt.Errorf("publish %s: %w", plan.name, err))
		}
		if plan.existed {
			plan.stagedPath = ""
		}
		plan.published = true
	}
	return nil
}

func rollbackPublishedUpdates(plans []*updateOutput, cause error) error {
	var rollbackErrors []string
	for i := len(plans) - 1; i >= 0; i-- {
		plan := plans[i]
		if !plan.published {
			continue
		}
		current, err := inspectUpdateOutput(plan.path, false)
		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("%s: %v", plan.name, err))
			continue
		}
		if !os.SameFile(plan.stagedInfo, current.info) || current.info.Mode() != plan.stagedInfo.Mode() || !bytes.Equal(plan.content, current.data) {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("%s: refusing to overwrite a concurrently changed output", plan.name))
			continue
		}
		if plan.existed {
			err = atomicReplace(plan.backupPath, plan.path)
			if err == nil {
				plan.backupPath = ""
			}
		} else {
			err = os.Remove(plan.path)
		}
		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("%s: %v", plan.name, err))
			continue
		}
		plan.published = false
	}
	if len(rollbackErrors) > 0 {
		return fmt.Errorf("%w; also failed to roll back update: %s", cause, strings.Join(rollbackErrors, "; "))
	}
	return cause
}

func cleanupUpdateTemps(plans []*updateOutput) {
	for _, plan := range plans {
		if plan.stagedPath != "" {
			_ = os.Remove(plan.stagedPath)
		}
		if plan.backupPath != "" {
			_ = os.Remove(plan.backupPath)
		}
	}
}
