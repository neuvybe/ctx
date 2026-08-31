package ctx

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// AddResult describes the add-ons and files installed by Add.
type AddResult struct {
	Addons []string
	Files  []string
}

// Add installs one or more optional document sets into an existing configured
// layout-v2 scaffold. Add-on IDs may be repeated or comma-separated. The
// operation refuses every existing output and publishes new documents, the
// INDEX routing block, and config.json as one rollback-capable transaction.
func Add(repo, folder string, addons []string) (AddResult, error) {
	if folder == "" {
		folder = ".ctx"
	}
	if err := validateExistingFolderPath(folder); err != nil {
		return AddResult{}, err
	}
	requested, err := normalizeAddonNames(addons)
	if err != nil {
		return AddResult{}, err
	}
	if len(requested) == 0 {
		return AddResult{}, fmt.Errorf("no add-ons requested")
	}
	if _, err := os.Stat(repo); err != nil {
		return AddResult{}, fmt.Errorf("target repo: %w", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		return AddResult{}, fmt.Errorf("target %s is not a git repository (no .git)", repo)
	}

	releaseLock, err := acquireLifecycleLock(repo)
	if err != nil {
		return AddResult{}, err
	}
	defer func() { _ = releaseLock() }()

	dest, err := validateScaffoldPath(repo, folder)
	if err != nil {
		return AddResult{}, err
	}
	configPath := filepath.Join(dest, configFileName)
	configOutput, err := inspectUpdateOutput(configPath, true)
	if err != nil {
		return AddResult{}, fmt.Errorf("preflight %s: %w", configFileName, err)
	}
	if !configOutput.exists {
		return AddResult{}, fmt.Errorf("ctx add requires a configured layout-v2 scaffold; legacy scaffolds are not migrated automatically")
	}
	cfg, err := parseConfig(configOutput.data)
	if err != nil {
		return AddResult{}, fmt.Errorf("parse %s: %w", configFileName, err)
	}
	if cfg.SchemaVersion != currentSchemaVersion || cfg.LayoutVersion != CurrentLayoutVersion {
		return AddResult{}, fmt.Errorf("ctx add requires schema %d layout v%d; found schema %d layout v%d (no automatic migration)", currentSchemaVersion, CurrentLayoutVersion, cfg.SchemaVersion, cfg.LayoutVersion)
	}
	revisionComparison, err := compareTemplateRevisions(cfg.TemplateRevision, CurrentTemplateRevision)
	if err != nil {
		return AddResult{}, fmt.Errorf("unsupported installed template revision %q: %w", cfg.TemplateRevision, err)
	}
	if revisionComparison < 0 {
		return AddResult{}, fmt.Errorf("template revision %q is older than %q; run `ctx update --folder %q` before adding documents", cfg.TemplateRevision, CurrentTemplateRevision, folder)
	}
	if revisionComparison > 0 {
		return AddResult{}, fmt.Errorf("template revision %q is newer than this CLI supports (%q); upgrade ctx before adding documents", cfg.TemplateRevision, CurrentTemplateRevision)
	}
	if err := validateAddScaffoldBaseline(dest, cfg); err != nil {
		return AddResult{}, err
	}

	installed := make(map[string]bool, len(cfg.Addons))
	for _, id := range cfg.Addons {
		installed[id] = true
	}
	for _, id := range requested {
		if installed[id] {
			return AddResult{}, fmt.Errorf("add-on %q is already installed", id)
		}
	}
	combined, err := normalizeAddonNames(append(append([]string(nil), cfg.Addons...), requested...))
	if err != nil {
		return AddResult{}, err
	}

	documents, err := addonDocuments(requested)
	if err != nil {
		return AddResult{}, err
	}
	parentPlans, err := preflightAddonParents(dest, documents)
	if err != nil {
		return AddResult{}, err
	}

	state := scaffoldState{Config: cfg}
	routes, err := addonRoutesMarkdown(combined)
	if err != nil {
		return AddResult{}, err
	}
	values := templateValues{
		Project:      cfg.Project,
		Date:         time.Now().Format("2006-01-02"),
		Folder:       folder,
		Mode:         state.modeLabel(),
		ContinuePath: state.continuePath(),
		AddonRoutes:  routes,
	}

	plans := make([]*updateOutput, 0, len(documents)+2)
	files := make([]string, 0, len(documents))
	for _, document := range documents {
		tracked, trackErr := gitTrackedFiles(repo, filepath.Join(folder, filepath.FromSlash(document.Path)))
		if trackErr != nil {
			return AddResult{}, fmt.Errorf("preflight tracked add-on output %s: %w", document.Path, trackErr)
		}
		if len(tracked) > 0 {
			return AddResult{}, fmt.Errorf("refusing add-on output %s because it already has tracked index entries", document.Path)
		}
		path := filepath.Join(dest, filepath.FromSlash(document.Path))
		existing, err := inspectUpdateOutput(path, true)
		if err != nil {
			return AddResult{}, fmt.Errorf("preflight %s: %w", document.Path, err)
		}
		if existing.exists {
			return AddResult{}, fmt.Errorf("refusing to overwrite existing add-on output %s", document.Path)
		}
		tmpl, err := readTemplateAsset(document.TemplatePath)
		if err != nil {
			return AddResult{}, fmt.Errorf("read embedded template %s: %w", document.TemplatePath, err)
		}
		content := []byte(renderTemplate(string(tmpl), values))
		if document.Managed {
			doc, parseErr := parseManagedDocument(string(content), managedMarkersNamed)
			if parseErr != nil {
				return AddResult{}, fmt.Errorf("embedded template %s has malformed managed markers: %w", document.TemplatePath, parseErr)
			}
			if err := validateManagedDocumentCatalog(document, doc, managedMarkersNamed); err != nil {
				return AddResult{}, fmt.Errorf("embedded template %s: %w", document.TemplatePath, err)
			}
		}
		plans = append(plans, &updateOutput{
			name:         document.Path,
			scaffoldRoot: dest,
			path:         path,
			content:      content,
			mode:         0o644,
		})
		files = append(files, document.Path)
	}

	indexDocument, err := catalogDocument(CurrentLayoutVersion, combined, "INDEX.md")
	if err != nil {
		return AddResult{}, err
	}
	indexPath := filepath.Join(dest, "INDEX.md")
	indexOutput, err := inspectUpdateOutput(indexPath, false)
	if err != nil {
		return AddResult{}, fmt.Errorf("preflight INDEX.md: %w", err)
	}
	indexTemplate, err := readTemplateAsset(indexDocument.TemplatePath)
	if err != nil {
		return AddResult{}, fmt.Errorf("read embedded template %s: %w", indexDocument.TemplatePath, err)
	}
	renderedIndex := renderTemplate(string(indexTemplate), values)
	indexTemplateDocument, err := parseManagedDocument(renderedIndex, managedMarkersNamed)
	if err != nil {
		return AddResult{}, fmt.Errorf("embedded template %s has malformed managed markers: %w", indexDocument.TemplatePath, err)
	}
	if err := validateManagedDocumentCatalog(indexDocument, indexTemplateDocument, managedMarkersNamed); err != nil {
		return AddResult{}, fmt.Errorf("embedded template %s: %w", indexDocument.TemplatePath, err)
	}
	updatedIndex, err := replaceNamedManagedBlock(string(indexOutput.data), renderedIndex, indexDocument.ManagedMarkerID)
	if err != nil {
		return AddResult{}, fmt.Errorf("preflight INDEX.md: %w", err)
	}
	plans = append(plans, &updateOutput{
		name:         "INDEX.md",
		scaffoldRoot: dest,
		path:         indexPath,
		content:      []byte(updatedIndex),
		mode:         indexOutput.info.Mode().Perm(),
		existed:      true,
		original:     indexOutput.data,
		originalInfo: indexOutput.info,
	})

	nextConfig := cfg
	nextConfig.Addons = combined
	configContent, err := marshalConfig(nextConfig)
	if err != nil {
		return AddResult{}, fmt.Errorf("render %s: %w", configFileName, err)
	}
	// Configuration is deliberately last: it must never advertise an add-on
	// whose document or routing entry failed to publish.
	plans = append(plans, &updateOutput{
		name:         configFileName,
		scaffoldRoot: dest,
		path:         configPath,
		content:      configContent,
		mode:         configOutput.info.Mode().Perm(),
		existed:      true,
		original:     configOutput.data,
		originalInfo: configOutput.info,
	})

	visibilityPaths, err := addonVisibilityPaths(dest, combined)
	if err != nil {
		return AddResult{}, err
	}
	if err := verifyAddonVisibility(repo, dest, folder, nextConfig, visibilityPaths); err != nil {
		return AddResult{}, fmt.Errorf("preflight add-on Git visibility: %w", err)
	}

	createdDirs, err := prepareAddonParents(parentPlans)
	if err != nil {
		return AddResult{}, err
	}
	keepDirs := false
	defer func() {
		if !keepDirs {
			removeEmptyAddonParents(createdDirs)
		}
	}()
	if err := stageAddonOutputs(plans, parentPlans); err != nil {
		cleanupUpdateTemps(plans)
		return AddResult{}, err
	}
	defer cleanupUpdateTemps(plans)
	if err := validateAddonParentsUnchanged(parentPlans); err != nil {
		return AddResult{}, err
	}
	if err := validateUpdateOutputsUnchanged(plans); err != nil {
		return AddResult{}, err
	}
	if err := publishAddonOutputs(plans, parentPlans); err != nil {
		return AddResult{}, err
	}
	if err := verifyAddonVisibility(repo, dest, folder, nextConfig, visibilityPaths); err != nil {
		return AddResult{}, rollbackPublishedUpdates(plans, fmt.Errorf("verify installed add-on Git visibility: %w", err))
	}

	keepDirs = true
	sort.Strings(files)
	return AddResult{Addons: append([]string(nil), requested...), Files: files}, nil
}

func addonVisibilityPaths(dest string, addons []string) ([]string, error) {
	existing, err := teamSharedPaths(dest)
	if err != nil {
		return nil, err
	}
	catalog, err := teamVisibleOutputs(CurrentLayoutVersion, addons)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(existing)+len(catalog))
	paths := make([]string, 0, len(existing)+len(catalog))
	for _, group := range [][]string{existing, catalog} {
		for _, name := range group {
			name = filepath.ToSlash(name)
			if seen[name] {
				continue
			}
			seen[name] = true
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func validateAddScaffoldBaseline(dest string, cfg Config) error {
	documents, err := DocumentsForLayout(cfg.LayoutVersion, cfg.Addons)
	if err != nil {
		return err
	}
	for _, document := range documents {
		if _, err := inspectScaffoldOutput(dest, document.Path, false); err != nil {
			return fmt.Errorf("configured scaffold output %s is missing or invalid; run ctx doctor before adding documents: %w", document.Path, err)
		}
	}
	if _, err := inspectScaffoldOutput(dest, ".gitignore", false); err != nil {
		return fmt.Errorf("configured scaffold output .gitignore is missing or invalid; run ctx doctor before adding documents: %w", err)
	}
	orphans, err := undeclaredAddonOutputs(dest, cfg.Addons)
	if err != nil {
		return fmt.Errorf("inspect existing add-on catalog state: %w", err)
	}
	if len(orphans) > 0 {
		return fmt.Errorf("undeclared add-on outputs block installation: %s; run ctx doctor for recovery guidance", strings.Join(orphans, ", "))
	}
	return nil
}

func addonDocuments(addons []string) ([]DocumentSpec, error) {
	core, err := DocumentsForLayout(CurrentLayoutVersion, nil)
	if err != nil {
		return nil, err
	}
	corePaths := make(map[string]bool, len(core))
	for _, document := range core {
		corePaths[document.Path] = true
	}
	all, err := DocumentsForLayout(CurrentLayoutVersion, addons)
	if err != nil {
		return nil, err
	}
	var documents []DocumentSpec
	for _, document := range all {
		if !corePaths[document.Path] {
			documents = append(documents, document)
		}
	}
	sort.Slice(documents, func(i, j int) bool { return documents[i].Path < documents[j].Path })
	if len(documents) == 0 {
		return nil, fmt.Errorf("selected add-ons contain no documents")
	}
	return documents, nil
}

func catalogDocument(layoutVersion int, addons []string, path string) (DocumentSpec, error) {
	documents, err := DocumentsForLayout(layoutVersion, addons)
	if err != nil {
		return DocumentSpec{}, err
	}
	for _, document := range documents {
		if document.Path == path {
			return document, nil
		}
	}
	return DocumentSpec{}, fmt.Errorf("layout v%d has no catalog document %s", layoutVersion, path)
}

func replaceNamedManagedBlock(existing, template, id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("managed marker ID must not be empty")
	}
	existingDoc, err := parseManagedDocument(existing, managedMarkersNamed)
	if err != nil {
		return "", fmt.Errorf("target: %w", err)
	}
	templateDoc, err := parseManagedDocument(template, managedMarkersNamed)
	if err != nil {
		return "", fmt.Errorf("template: %w", err)
	}
	if len(existingDoc.blocks) != 1 || existingDoc.blocks[0].id != id {
		return "", fmt.Errorf("target managed block set must be exactly [%s]", id)
	}
	if len(templateDoc.blocks) != 1 || templateDoc.blocks[0].id != id {
		return "", fmt.Errorf("template managed block set must be exactly [%s]", id)
	}
	return updateManagedContentStrict(existing, template, managedMarkersNamed)
}

type addonParentPlan struct {
	path    string
	existed bool
	info    os.FileInfo
}

func preflightAddonParents(dest string, documents []DocumentSpec) ([]*addonParentPlan, error) {
	destInfo, err := os.Lstat(dest)
	if err != nil {
		return nil, fmt.Errorf("inspect scaffold parent %s: %w", dest, err)
	}
	if destInfo.Mode()&os.ModeSymlink != 0 || !destInfo.IsDir() {
		return nil, fmt.Errorf("scaffold parent %s is not a real directory", dest)
	}
	byPath := map[string]*addonParentPlan{
		dest: {path: dest, existed: true, info: destInfo},
	}
	for _, document := range documents {
		rel, err := cleanAddonOutputPath(document.Path)
		if err != nil {
			return nil, err
		}
		parentRel := filepath.Dir(rel)
		if parentRel == "." {
			continue
		}
		current := dest
		for _, part := range strings.Split(parentRel, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			if _, seen := byPath[current]; seen {
				continue
			}
			info, statErr := os.Lstat(current)
			if os.IsNotExist(statErr) {
				byPath[current] = &addonParentPlan{path: current}
				continue
			}
			if statErr != nil {
				return nil, fmt.Errorf("inspect add-on parent %s: %w", current, statErr)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("refusing add-on output through symbolic-link parent %s", current)
			}
			if !info.IsDir() {
				return nil, fmt.Errorf("add-on output parent %s is not a directory", current)
			}
			byPath[current] = &addonParentPlan{path: current, existed: true, info: info}
		}
	}
	plans := make([]*addonParentPlan, 0, len(byPath))
	for _, plan := range byPath {
		plans = append(plans, plan)
	}
	sort.Slice(plans, func(i, j int) bool {
		depthI := strings.Count(filepath.Clean(plans[i].path), string(filepath.Separator))
		depthJ := strings.Count(filepath.Clean(plans[j].path), string(filepath.Separator))
		if depthI != depthJ {
			return depthI < depthJ
		}
		return plans[i].path < plans[j].path
	})
	return plans, nil
}

func cleanAddonOutputPath(path string) (string, error) {
	rel := filepath.Clean(filepath.FromSlash(path))
	if path == "" || filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid add-on output path %q", path)
	}
	return rel, nil
}

func prepareAddonParents(plans []*addonParentPlan) ([]string, error) {
	var created []string
	for _, plan := range plans {
		if plan.existed {
			continue
		}
		if err := os.Mkdir(plan.path, 0o755); err != nil {
			removeEmptyAddonParents(created)
			return nil, fmt.Errorf("create add-on parent %s: %w", plan.path, err)
		}
		info, err := os.Lstat(plan.path)
		if err != nil {
			removeEmptyAddonParents(append(created, plan.path))
			return nil, fmt.Errorf("inspect created add-on parent %s: %w", plan.path, err)
		}
		plan.info = info
		created = append(created, plan.path)
	}
	if err := validateAddonParentsUnchanged(plans); err != nil {
		removeEmptyAddonParents(created)
		return nil, err
	}
	return created, nil
}

func validateAddonParentsUnchanged(plans []*addonParentPlan) error {
	for _, plan := range plans {
		info, err := os.Lstat(plan.path)
		if err != nil {
			return fmt.Errorf("reinspect add-on parent %s: %w", plan.path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || plan.info == nil || !os.SameFile(plan.info, info) {
			return fmt.Errorf("add-on parent %s changed during installation", plan.path)
		}
	}
	return nil
}

func removeEmptyAddonParents(paths []string) {
	for i := len(paths) - 1; i >= 0; i-- {
		_ = os.Remove(paths[i])
	}
}

func stageAddonOutputs(plans []*updateOutput, parentPlans []*addonParentPlan) error {
	for _, plan := range plans {
		if err := validateAddonParentsUnchanged(parentPlans); err != nil {
			return err
		}
		dir := filepath.Dir(plan.path)
		path, info, err := stageUpdateFile(dir, filepath.Base(plan.name), plan.content, plan.mode)
		if err != nil {
			return fmt.Errorf("stage %s: %w", plan.name, err)
		}
		plan.stagedPath = path
		plan.stagedInfo = info
		if !plan.existed {
			continue
		}
		backup, _, err := stageUpdateFile(dir, filepath.Base(plan.name)+"-backup", plan.original, plan.originalInfo.Mode().Perm())
		if err != nil {
			return fmt.Errorf("stage rollback for %s: %w", plan.name, err)
		}
		plan.backupPath = backup
	}
	return nil
}

func publishAddonOutputs(plans []*updateOutput, parentPlans []*addonParentPlan) error {
	for _, plan := range plans {
		if err := validateAddonParentsUnchanged(parentPlans); err != nil {
			return rollbackPublishedUpdates(plans, fmt.Errorf("publish %s: %w", plan.name, err))
		}
		if err := validateUpdateOutputUnchanged(plan); err != nil {
			return rollbackPublishedUpdates(plans, fmt.Errorf("publish %s: %w", plan.name, err))
		}
		var err error
		if plan.existed {
			err = atomicReplace(plan.stagedPath, plan.path)
		} else {
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

func verifyAddonVisibility(repo, dest, folder string, cfg Config, paths []string) error {
	paths = append([]string(nil), paths...)
	sort.Strings(paths)
	if cfg.Mode == ModeTeam {
		if err := verifyTeamPathsVisible(repo, folder, paths); err != nil {
			return err
		}
		return verifyTeamLocalPrivate(repo, dest, folder, scaffoldState{Config: cfg})
	}
	if cfg.Mode != ModeLocal {
		return fmt.Errorf("unsupported scaffold mode %q", cfg.Mode)
	}
	hasExclude, err := hasFolderExclusion(repo, folder)
	if err != nil {
		return err
	}
	if !hasExclude {
		return fmt.Errorf("repo-local folder exclusion is missing for %s", folder)
	}
	tracked, err := gitTrackedFiles(repo, folder)
	if err != nil {
		return err
	}
	if len(tracked) > 0 {
		return fmt.Errorf("local mode cannot hide tracked files: %s", strings.Join(tracked, ", "))
	}
	ignored, err := gitCheckIgnored(repo, folder)
	if err != nil {
		return err
	}
	if !ignored {
		return fmt.Errorf("%s/ is visible to Git", filepath.ToSlash(folder))
	}
	for _, path := range paths {
		rel := filepath.Join(folder, filepath.FromSlash(path))
		ignored, err := gitCheckIgnored(repo, rel)
		if err != nil {
			return err
		}
		if !ignored {
			return fmt.Errorf("%s is visible to Git", filepath.ToSlash(rel))
		}
	}
	return nil
}
