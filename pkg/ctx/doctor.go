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

func expectedFilesFor(state scaffoldState) []string {
	files, _ := RequiredOutputs(state.layoutVersion(), state.Config.Addons, state.Legacy)
	return files
}

// Doctor validates scaffold integrity and verifies that Git visibility matches
// the configured audience. Config-less legacy scaffolds remain local and keep a
// root CONTINUE.md; doctor never converts them.
func Doctor(repo, folder string) ([]Check, error) {
	if err := validateExistingFolderPath(folder); err != nil {
		return nil, err
	}
	releaseLock, err := acquireLifecycleLock(repo)
	if err != nil {
		return nil, err
	}
	defer func() { _ = releaseLock() }()
	dest := filepath.Join(repo, folder)
	var checks []Check

	// 1. folder exists
	validatedDest, pathErr := validateScaffoldPath(repo, folder)
	if pathErr != nil {
		return []Check{fail(folder+"/ folder exists", pathErr.Error())}, nil
	}
	dest = validatedDest
	checks = append(checks, pass(folder+"/ folder exists", dest))

	// 2. scaffold configuration and mode
	state, stateErr := inspectDoctorScaffoldState(dest)
	if stateErr != nil {
		checks = append(checks, fail("scaffold config", stateErr.Error()))
		currentLayout, _ := LayoutForVersion(CurrentLayoutVersion)
		state = scaffoldState{Config: Config{
			SchemaVersion:    currentSchemaVersion,
			LayoutVersion:    CurrentLayoutVersion,
			TemplateRevision: currentLayout.TemplateRevision,
			Project:          "unknown",
			Mode:             ModeTeam,
		}}
	} else if state.Legacy {
		checks = append(checks, pass("scaffold mode", state.modeLabel()))
	} else {
		checks = append(checks, pass("scaffold mode", fmt.Sprintf("%s (schema %d, layout v%d)", state.Config.Mode, state.Config.SchemaVersion, state.layoutVersion())))
	}
	layout, layoutOK := LayoutForVersion(state.layoutVersion())
	if !layoutOK {
		checks = append(checks, fail("layout catalog", fmt.Sprintf("unsupported layoutVersion %d", state.layoutVersion())))
		layout, _ = LayoutForVersion(CurrentLayoutVersion)
	}
	managedDocuments, managedErr := ManagedDocuments(layout.Version, state.Config.Addons)
	if managedErr != nil {
		checks = append(checks, fail("layout catalog", managedErr.Error()))
		managedDocuments = nil
	}
	expectedFiles, expectedErr := RequiredOutputs(layout.Version, state.Config.Addons, state.Legacy)
	if expectedErr != nil {
		checks = append(checks, fail("layout catalog", expectedErr.Error()))
		expectedFiles = nil
	}
	if stateErr == nil && !state.Legacy && layout.Version == CurrentLayoutVersion {
		orphans, orphanErr := undeclaredAddonOutputs(dest, state.Config.Addons)
		if orphanErr != nil {
			checks = append(checks, fail("add-on catalog consistency", orphanErr.Error()))
		} else if len(orphans) > 0 {
			checks = append(checks, fail("add-on catalog consistency", "undeclared outputs: "+strings.Join(orphans, ", ")+"; move/remove the orphan before running ctx add, or restore its ID in config.json and then run ctx update"))
		} else {
			checks = append(checks, pass("add-on catalog consistency", "no undeclared catalog outputs"))
		}
	}

	// 3. layout-aware template revision metadata
	if layout.Version == LegacyLayoutVersion {
		if v, err := inspectDoctorFile(dest, ".ctx-version"); err == nil {
			stamp := strings.TrimSpace(string(v))
			if stamp == "" {
				checks = append(checks, fail(".ctx-version stamp", "empty legacy release stamp"))
			} else {
				checks = append(checks, pass(".ctx-version stamp", stamp+" (v1 compatibility)"))
			}
		} else {
			checks = append(checks, fail(".ctx-version stamp", err.Error()))
		}
	} else {
		installed := state.templateRevision()
		comparison, revisionErr := compareTemplateRevisions(installed, layout.TemplateRevision)
		if revisionErr != nil {
			checks = append(checks, fail("template revision", fmt.Sprintf("unsupported installed revision %q: %v", installed, revisionErr)))
		} else if comparison > 0 {
			checks = append(checks, fail("template revision", fmt.Sprintf("installed %q is newer than this CLI supports (%q); upgrade ctx", installed, layout.TemplateRevision)))
		} else if comparison == 0 {
			checks = append(checks, pass("template revision", installed))
		} else {
			checks = append(checks, fail("template revision", fmt.Sprintf("installed %q, current %q; run `ctx update --folder %q`", installed, layout.TemplateRevision, folder)))
		}
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
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			// Required paths receive a specific failure below. Other special or
			// symlinked Markdown entries are never followed by Doctor.
			return nil
		}
		b, err := os.ReadFile(filepath.Join(dest, path))
		if err != nil {
			return err
		}
		if strings.Contains(string(b), "{{PROJECT}}") || strings.Contains(string(b), "{{DATE}}") ||
			strings.Contains(string(b), "{{FOLDER}}") || strings.Contains(string(b), "{{MODE}}") ||
			strings.Contains(string(b), "{{CONTINUE_PATH}}") || strings.Contains(string(b), "{{ADDON_ROUTES}}") {
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

	// 6. layout-specific managed marker grammar and identity
	markerFormat, markerFormatErr := managedMarkerFormatFor(layout.MarkerFormat)
	if markerFormatErr != nil {
		checks = append(checks, fail("managed marker format", markerFormatErr.Error()))
	}
	for _, document := range managedDocuments {
		name := document.Path
		b, err := inspectDoctorFile(dest, name)
		if err != nil {
			checks = append(checks, fail("markers balanced in "+name, err.Error()))
			continue
		}
		template, err := readTemplateAsset(document.TemplatePath)
		if err != nil {
			checks = append(checks, fail("markers balanced in "+name, fmt.Sprintf("read embedded template: %v", err)))
			continue
		}
		templateDoc, err := parseManagedDocument(string(template), markerFormat)
		if err == nil {
			err = validateManagedDocumentCatalog(document, templateDoc, markerFormat)
		}
		if err == nil {
			err = validateManagedCompatibility(string(b), string(template), markerFormat)
		}
		if err != nil {
			checks = append(checks, fail("markers balanced in "+name, err.Error()))
			continue
		}
		checks = append(checks, pass("markers balanced in "+name, string(layout.MarkerFormat)))
	}

	// 7. expected paths are present as regular files
	var invalid []string
	for _, name := range expectedFiles {
		if _, err := inspectDoctorFile(dest, name); err != nil {
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

func undeclaredAddonOutputs(dest string, configured []string) ([]string, error) {
	selected, err := normalizeAddonSelection(configured)
	if err != nil {
		return nil, err
	}
	var orphans []string
	for _, spec := range addonCatalog {
		if selected[spec.ID] {
			continue
		}
		exists, err := scaffoldEntryExists(dest, spec.Document.Path)
		if err != nil {
			return nil, fmt.Errorf("inspect add-on %q output: %w", spec.ID, err)
		}
		if exists {
			orphans = append(orphans, fmt.Sprintf("%s (%s)", spec.Document.Path, spec.ID))
		}
	}
	return orphans, nil
}

// scaffoldEntryExists checks a catalog path without following any parent
// symlink. A missing parent means the output is absent; any final entry counts
// as present because it would block ctx add regardless of its file type.
func scaffoldEntryExists(dest, name string) (bool, error) {
	name = filepath.FromSlash(name)
	clean := filepath.Clean(name)
	if name == "" || filepath.IsAbs(name) || clean == "." || clean == ".." || clean != name || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false, fmt.Errorf("invalid scaffold output path %q", name)
	}
	current := dest
	parts := strings.Split(clean, string(filepath.Separator))
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if i == len(parts)-1 {
			return true, nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("output parent %s is a symbolic link", filepath.ToSlash(strings.Join(parts[:i+1], string(filepath.Separator))))
		}
		if !info.IsDir() {
			return false, fmt.Errorf("output parent %s is not a directory", filepath.ToSlash(strings.Join(parts[:i+1], string(filepath.Separator))))
		}
	}
	return false, nil
}

// inspectDoctorFile applies the same non-symlink, regular-file policy as
// Update, including every directory component below the scaffold root. It then
// reads through Update's descriptor/entry identity check so a concurrent swap
// cannot turn the inspection into a symlink traversal.
func inspectDoctorFile(dest, name string) ([]byte, error) {
	name = filepath.FromSlash(name)
	clean := filepath.Clean(name)
	if name == "" || filepath.IsAbs(name) || clean == "." || clean == ".." || clean != name || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("invalid scaffold output path %q", name)
	}
	current := dest
	parts := strings.Split(clean, string(filepath.Separator))
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", filepath.ToSlash(name), err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("required output %s is a symbolic link", filepath.ToSlash(name))
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, fmt.Errorf("required output parent %s is not a directory", filepath.ToSlash(strings.Join(parts[:i+1], string(filepath.Separator))))
		}
	}
	output, err := inspectUpdateOutput(filepath.Join(dest, clean), false)
	if err != nil {
		return nil, err
	}
	return output.data, nil
}

func inspectDoctorScaffoldState(dest string) (scaffoldState, error) {
	configPath := filepath.Join(dest, configFileName)
	if _, err := os.Lstat(configPath); err == nil {
		data, err := inspectDoctorFile(dest, configFileName)
		if err != nil {
			return scaffoldState{}, fmt.Errorf("inspect %s: %w", configFileName, err)
		}
		cfg, err := parseConfig(data)
		if err != nil {
			return scaffoldState{}, fmt.Errorf("parse %s: %w", configFileName, err)
		}
		return scaffoldState{Config: cfg}, nil
	} else if !os.IsNotExist(err) {
		return scaffoldState{}, fmt.Errorf("inspect %s: %w", configFileName, err)
	}

	if _, err := os.Lstat(filepath.Join(dest, "local", "CONTINUE.md")); err == nil {
		return scaffoldState{}, fmt.Errorf("missing %s in a new-layout scaffold", configFileName)
	} else if !os.IsNotExist(err) {
		return scaffoldState{}, fmt.Errorf("inspect local continuation: %w", err)
	}
	if _, err := inspectDoctorFile(dest, "CONTINUE.md"); err != nil {
		return scaffoldState{}, fmt.Errorf("missing %s and no valid legacy CONTINUE.md: %w", configFileName, err)
	}
	legacyLayout, _ := LayoutForVersion(LegacyLayoutVersion)
	return scaffoldState{Config: Config{
		LayoutVersion:    LegacyLayoutVersion,
		TemplateRevision: legacyLayout.TemplateRevision,
		Mode:             ModeLocal,
	}, Legacy: true}, nil
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
		catalogShared, catalogErr := teamVisibleOutputs(state.layoutVersion(), state.Config.Addons)
		if catalogErr != nil {
			checks = append(checks, fail("shared context Git-visible", catalogErr.Error()))
		} else {
			seen := make(map[string]bool, len(sharedFiles)+len(catalogShared))
			for _, name := range sharedFiles {
				seen[filepath.ToSlash(name)] = true
			}
			for _, name := range catalogShared {
				name = filepath.ToSlash(name)
				if !seen[name] {
					seen[name] = true
					sharedFiles = append(sharedFiles, name)
				}
			}
		}
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

	paths, pathsErr := privacyPaths(filepath.Join(repo, folder), ".", append(expectedFilesFor(state), ".ctx-local-mode-probe")...)
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
	err := fs.WalkDir(os.DirFS(dest), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.ToSlash(path) == "local" {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
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
	b, err := inspectDoctorFile(dest, ".gitignore")
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
