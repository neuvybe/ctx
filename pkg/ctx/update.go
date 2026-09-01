package ctx

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type updateOutput struct {
	name         string
	scaffoldRoot string
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

// Update refreshes the managed documents declared by the scaffold's persisted
// layout and add-on catalog. V1 uses its frozen unnamed-marker templates and
// .ctx-version stamp. V2 uses named IDs and atomically advances the config's
// templateRevision only after every managed output passes preflight.
func Update(repo, folder string) ([]string, error) {
	if err := validateExistingFolderPath(folder); err != nil {
		return nil, err
	}
	releaseLock, err := acquireLifecycleLock(repo)
	if err != nil {
		return nil, err
	}
	defer func() { _ = releaseLock() }()
	dest, err := validateScaffoldPath(repo, folder)
	if err != nil {
		return nil, err
	}
	state, configSnapshot, err := inspectUpdateScaffoldState(dest)
	if err != nil {
		return nil, err
	}
	layout, ok := LayoutForVersion(state.layoutVersion())
	if !ok {
		return nil, fmt.Errorf("unsupported layoutVersion %d", state.layoutVersion())
	}
	if layout.Version != LegacyLayoutVersion {
		comparison, err := compareTemplateRevisions(state.templateRevision(), layout.TemplateRevision)
		if err != nil {
			return nil, fmt.Errorf("unsupported installed template revision %q: %w", state.templateRevision(), err)
		}
		if comparison > 0 {
			return nil, fmt.Errorf("installed template revision %q is newer than this CLI supports (%q); upgrade ctx instead of downgrading the scaffold", state.templateRevision(), layout.TemplateRevision)
		}
	}
	managedDocuments, err := ManagedDocuments(layout.Version, state.Config.Addons)
	if err != nil {
		return nil, err
	}
	markerFormat, err := managedMarkerFormatFor(layout.MarkerFormat)
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return nil, err
	}
	project := state.Config.Project
	if project == "" {
		project = filepath.Base(abs)
	}
	addonRoutes, err := addonRoutesMarkdown(state.Config.Addons)
	if err != nil {
		return nil, err
	}
	values := templateValues{
		Project:      project,
		Date:         time.Now().Format("2006-01-02"),
		Folder:       folder,
		Mode:         state.modeLabel(),
		ContinuePath: state.continuePath(),
		AddonRoutes:  addonRoutes,
	}

	// Preflight every managed output before creating a temp file or changing an
	// intended output. V1 retains its markerless/missing user-ownership behavior.
	// V2's named ID set is structural, so missing markers or files fail closed.
	var plans []*updateOutput
	var touched []string
	for _, document := range managedDocuments {
		name := document.Path
		path := filepath.Join(dest, name)
		existing, err := inspectScaffoldOutput(dest, name, layout.Version == LegacyLayoutVersion)
		if err != nil {
			return nil, fmt.Errorf("preflight %s: %w", name, err)
		}
		if !existing.exists {
			continue
		}
		existingDoc, err := parseManagedDocument(string(existing.data), markerFormat)
		if err != nil {
			return nil, fmt.Errorf("preflight %s: malformed managed-marker grammar: %w", name, err)
		}
		if layout.Version == LegacyLayoutVersion && len(existingDoc.blocks) == 0 {
			continue
		}

		tmpl, err := readTemplateAsset(document.TemplatePath)
		if err != nil {
			return nil, fmt.Errorf("read embedded template %s: %w", name, err)
		}
		tmplStr := renderTemplate(string(tmpl), values)
		templateDoc, err := parseManagedDocument(tmplStr, markerFormat)
		if err != nil {
			return nil, fmt.Errorf("embedded template %s has malformed managed-marker grammar: %w", name, err)
		}
		if len(templateDoc.blocks) == 0 {
			return nil, fmt.Errorf("embedded template %s has no managed blocks", name)
		}
		if err := validateManagedDocumentCatalog(document, templateDoc, markerFormat); err != nil {
			return nil, fmt.Errorf("embedded template %s does not match the layout catalog: %w", name, err)
		}
		updated, err := updateManagedContentStrict(string(existing.data), tmplStr, markerFormat)
		if err != nil {
			return nil, fmt.Errorf("preflight %s: %w", name, err)
		}
		touched = append(touched, name)
		plans = append(plans, &updateOutput{
			name:         name,
			scaffoldRoot: dest,
			path:         path,
			content:      []byte(updated),
			mode:         existing.info.Mode().Perm(),
			existed:      true,
			original:     existing.data,
			originalInfo: existing.info,
		})
	}

	if layout.Version == LegacyLayoutVersion {
		versionPath := filepath.Join(dest, ".ctx-version")
		version, err := inspectScaffoldOutput(dest, ".ctx-version", true)
		if err != nil {
			return nil, fmt.Errorf("preflight .ctx-version: %w", err)
		}
		versionMode := os.FileMode(0o644)
		if version.exists {
			versionMode = version.info.Mode().Perm()
		}
		plans = append(plans, &updateOutput{
			name:         ".ctx-version",
			scaffoldRoot: dest,
			path:         versionPath,
			content:      []byte(Version + "\n"),
			mode:         versionMode,
			existed:      version.exists,
			original:     version.data,
			originalInfo: version.info,
		})
	} else {
		if !configSnapshot.exists {
			return nil, fmt.Errorf("preflight %s: missing from layout v%d scaffold", configFileName, layout.Version)
		}
		updatedConfig := state.Config
		updatedConfig.TemplateRevision = layout.TemplateRevision
		configData, err := marshalConfig(updatedConfig)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", configFileName, err)
		}
		plans = append(plans, &updateOutput{
			name:         configFileName,
			scaffoldRoot: dest,
			path:         filepath.Join(dest, configFileName),
			content:      configData,
			mode:         configSnapshot.info.Mode().Perm(),
			existed:      true,
			original:     configSnapshot.data,
			originalInfo: configSnapshot.info,
		})
	}

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

// inspectUpdateScaffoldState parses configuration from the exact regular-file
// snapshot that Update will later validate and, for v2, atomically replace.
// This prevents a concurrent config edit from being silently overwritten.
func inspectUpdateScaffoldState(dest string) (scaffoldState, inspectedOutput, error) {
	config, err := inspectScaffoldOutput(dest, configFileName, true)
	if err != nil {
		return scaffoldState{}, inspectedOutput{}, fmt.Errorf("preflight %s: %w", configFileName, err)
	}
	if config.exists {
		cfg, err := parseConfig(config.data)
		if err != nil {
			return scaffoldState{}, inspectedOutput{}, fmt.Errorf("parse %s: %w", configFileName, err)
		}
		return scaffoldState{Config: cfg}, config, nil
	}

	localContinue := filepath.Join(dest, "local", "CONTINUE.md")
	if _, err := os.Lstat(localContinue); err == nil {
		return scaffoldState{}, inspectedOutput{}, fmt.Errorf("missing %s in a new-layout scaffold", configFileName)
	} else if !os.IsNotExist(err) {
		return scaffoldState{}, inspectedOutput{}, fmt.Errorf("inspect local continuation: %w", err)
	}
	if _, err := inspectUpdateOutput(filepath.Join(dest, "CONTINUE.md"), false); err != nil {
		return scaffoldState{}, inspectedOutput{}, fmt.Errorf("cannot determine scaffold mode: missing %s and no valid legacy CONTINUE.md: %w", configFileName, err)
	}
	legacyLayout, _ := LayoutForVersion(LegacyLayoutVersion)
	return scaffoldState{Config: Config{
		LayoutVersion:    LegacyLayoutVersion,
		TemplateRevision: legacyLayout.TemplateRevision,
		Mode:             ModeLocal,
	}, Legacy: true}, config, nil
}

func validateManagedDocumentCatalog(document DocumentSpec, template managedDocument, format managedMarkerFormat) error {
	if format == managedMarkersUnnamed {
		if document.ManagedMarkerID != "" {
			return fmt.Errorf("v1 document declares named marker ID %q", document.ManagedMarkerID)
		}
		return nil
	}
	if document.ManagedMarkerID == "" {
		return fmt.Errorf("v2 document has no managed marker ID")
	}
	if len(template.blocks) != 1 || template.blocks[0].id != document.ManagedMarkerID {
		var ids []string
		for _, block := range template.blocks {
			ids = append(ids, block.id)
		}
		return fmt.Errorf("catalog ID %q, template IDs %v", document.ManagedMarkerID, ids)
	}
	return nil
}

// compareTemplateRevisions compares the deliberately layout-scoped numeric
// template revision format. It is separate from the ctx CLI release version.
func compareTemplateRevisions(left, right string) (int, error) {
	parse := func(value string) ([3]int, error) {
		var parsed [3]int
		parts := strings.Split(value, ".")
		if len(parts) != len(parsed) {
			return parsed, fmt.Errorf("want MAJOR.MINOR.PATCH")
		}
		for i, part := range parts {
			if part == "" {
				return parsed, fmt.Errorf("want MAJOR.MINOR.PATCH")
			}
			n, err := strconv.Atoi(part)
			if err != nil || n < 0 {
				return parsed, fmt.Errorf("want non-negative numeric MAJOR.MINOR.PATCH")
			}
			parsed[i] = n
		}
		return parsed, nil
	}
	a, err := parse(left)
	if err != nil {
		return 0, err
	}
	b, err := parse(right)
	if err != nil {
		return 0, fmt.Errorf("invalid catalog revision %q: %w", right, err)
	}
	for i := range a {
		if a[i] < b[i] {
			return -1, nil
		}
		if a[i] > b[i] {
			return 1, nil
		}
	}
	return 0, nil
}

// validateScaffoldPath rejects a scaffold path that is itself a symlink or
// traverses a symlink below repo. Nested paths remain supported for older
// custom scaffolds, but lifecycle operations never use them to escape the
// supplied repository.
func validateScaffoldPath(repo, folder string) (string, error) {
	current := repo
	cleanFolder := filepath.Clean(filepath.FromSlash(folder))
	for _, part := range strings.Split(cleanFolder, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("no %s here; run `ctx init --folder %q` first", filepath.Join(repo, folder), folder)
			}
			return "", fmt.Errorf("inspect scaffold path %s: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing scaffold through symbolic link %s", current)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("scaffold path %s is not a directory", current)
		}
	}
	return filepath.Join(repo, cleanFolder), nil
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

// inspectScaffoldOutput rejects symbolic-link or non-directory components
// between the scaffold root and an output before inspecting the output entry.
// The same check is repeated immediately before publication through each
// updateOutput's scaffoldRoot.
func inspectScaffoldOutput(dest, name string, allowMissing bool) (inspectedOutput, error) {
	native := filepath.FromSlash(name)
	clean := filepath.Clean(native)
	if name == "" || filepath.IsAbs(native) || clean == "." || clean == ".." || clean != native || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return inspectedOutput{}, fmt.Errorf("invalid scaffold output path %q", name)
	}

	current := dest
	rootInfo, err := os.Lstat(current)
	if err != nil {
		return inspectedOutput{}, fmt.Errorf("inspect scaffold root: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return inspectedOutput{}, fmt.Errorf("scaffold root %s is not a real directory", dest)
	}
	parts := strings.Split(clean, string(filepath.Separator))
	for i := 0; i < len(parts)-1; i++ {
		current = filepath.Join(current, parts[i])
		info, err := os.Lstat(current)
		if err != nil {
			return inspectedOutput{}, fmt.Errorf("inspect output parent %s: %w", filepath.ToSlash(strings.Join(parts[:i+1], string(filepath.Separator))), err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return inspectedOutput{}, fmt.Errorf("output parent %s is a symbolic link", filepath.ToSlash(strings.Join(parts[:i+1], string(filepath.Separator))))
		}
		if !info.IsDir() {
			return inspectedOutput{}, fmt.Errorf("output parent %s is not a directory", filepath.ToSlash(strings.Join(parts[:i+1], string(filepath.Separator))))
		}
	}
	return inspectUpdateOutput(filepath.Join(dest, clean), allowMissing)
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
	var current inspectedOutput
	var err error
	if plan.scaffoldRoot != "" {
		current, err = inspectScaffoldOutput(plan.scaffoldRoot, plan.name, !plan.existed)
	} else {
		current, err = inspectUpdateOutput(plan.path, !plan.existed)
	}
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
		var current inspectedOutput
		var err error
		if plan.scaffoldRoot != "" {
			current, err = inspectScaffoldOutput(plan.scaffoldRoot, plan.name, false)
		} else {
			current, err = inspectUpdateOutput(plan.path, false)
		}
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
