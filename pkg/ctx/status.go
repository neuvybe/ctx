package ctx

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ContentState is the outcome of one content-readiness check.
type ContentState string

const (
	// ContentReady means a document satisfies its readiness contract.
	ContentReady ContentState = "ready"
	// ContentNotReady means a document must be completed or reverified.
	ContentNotReady ContentState = "not-ready"
	// ContentWarning is non-fatal guidance, such as a document-size warning.
	ContentWarning ContentState = "warning"
)

// ContentCheck is one content-readiness finding. Path is relative to the
// scaffold folder; layout-wide checks use "layout".
type ContentCheck struct {
	Path   string
	State  ContentState
	Detail string
}

// ContentStatus is the complete readiness report for a scaffold.
type ContentStatus struct {
	LayoutVersion int
	Checks        []ContentCheck
}

// Ready reports whether the scaffold has no not-ready findings. Warnings do
// not make content unready.
func (s ContentStatus) Ready() bool {
	for _, check := range s.Checks {
		if check.State == ContentNotReady {
			return false
		}
	}
	return true
}

// Status evaluates project-fact documents without modifying the repository.
// Layout-v2 catalog facts and every Markdown file nested below context/ are
// checked. Layout v1 remains supported by init/update/doctor, but it does not
// carry the metadata required to certify content readiness.
func Status(repo, folder string) (ContentStatus, error) {
	if err := validateExistingFolderPath(folder); err != nil {
		return ContentStatus{}, err
	}
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return ContentStatus{}, fmt.Errorf("resolve target repository: %w", err)
	}
	dest, err := validateScaffoldPath(absRepo, folder)
	if err != nil {
		return ContentStatus{}, fmt.Errorf("inspect context folder: %w", err)
	}

	state, err := loadScaffoldState(dest)
	if err != nil {
		return ContentStatus{}, err
	}
	layoutVersion := state.layoutVersion()
	report := ContentStatus{LayoutVersion: layoutVersion}
	if layoutVersion != CurrentLayoutVersion {
		report.Checks = []ContentCheck{{
			Path:   "layout",
			State:  ContentNotReady,
			Detail: fmt.Sprintf("layout v%d has no content-readiness metadata; status requires layout v%d", layoutVersion, CurrentLayoutVersion),
		}}
		return report, nil
	}

	required, err := ProjectFactDocuments(layoutVersion, state.Config.Addons)
	if err != nil {
		return ContentStatus{}, err
	}
	paths := make(map[string]bool, len(required))
	for _, document := range required {
		paths[filepath.ToSlash(document.Path)] = true
	}
	discovered, discoveryChecks, err := discoverContextDocuments(dest)
	if err != nil {
		return ContentStatus{}, err
	}
	for _, documentPath := range discovered {
		paths[documentPath] = true
	}

	ordered := make([]string, 0, len(paths))
	for documentPath := range paths {
		ordered = append(ordered, documentPath)
	}
	sort.Strings(ordered)
	for _, documentPath := range ordered {
		checks, err := checkContentDocument(absRepo, dest, folder, documentPath)
		if err != nil {
			return ContentStatus{}, err
		}
		report.Checks = append(report.Checks, checks...)
	}
	report.Checks = append(report.Checks, mechanicsSizeWarnings(dest, state.continuePath())...)
	report.Checks = append(report.Checks, discoveryChecks...)
	sort.SliceStable(report.Checks, func(i, j int) bool {
		if report.Checks[i].Path != report.Checks[j].Path {
			return report.Checks[i].Path < report.Checks[j].Path
		}
		return contentStateOrder(report.Checks[i].State) < contentStateOrder(report.Checks[j].State)
	})
	return report, nil
}

func contentStateOrder(state ContentState) int {
	switch state {
	case ContentNotReady:
		return 0
	case ContentReady:
		return 1
	default:
		return 2
	}
}

// discoverContextDocuments deliberately uses WalkDir's no-follow behavior.
// A symbolic-link directory is reported as not ready because otherwise status
// could not make a claim about Markdown files hidden behind it.
func discoverContextDocuments(dest string) ([]string, []ContentCheck, error) {
	contextRoot := filepath.Join(dest, "context")
	rootInfo, err := os.Lstat(contextRoot)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("inspect context directory: %w", err)
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return nil, []ContentCheck{{Path: "context", State: ContentNotReady, Detail: "context/ must be a non-symbolic-link directory"}}, nil
	}

	var documents []string
	var checks []ContentCheck
	err = filepath.WalkDir(contextRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		rel, relErr := filepath.Rel(dest, current)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if walkErr != nil {
			checks = append(checks, ContentCheck{Path: rel, State: ContentNotReady, Detail: walkErr.Error()})
			if entry != nil && entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if current == contextRoot {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			checks = append(checks, ContentCheck{Path: rel, State: ContentNotReady, Detail: "symbolic links are not followed by status"})
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			checks = append(checks, ContentCheck{Path: rel, State: ContentNotReady, Detail: fmt.Sprintf("inspect document: %v", infoErr)})
			return nil
		}
		if !info.Mode().IsRegular() {
			checks = append(checks, ContentCheck{Path: rel, State: ContentNotReady, Detail: "document is not a regular file"})
			return nil
		}
		documents = append(documents, rel)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("discover context documents: %w", err)
	}
	return documents, checks, nil
}

func checkContentDocument(repo, dest, folder, documentPath string) ([]ContentCheck, error) {
	content, err := inspectDoctorFile(dest, documentPath)
	if err != nil {
		return []ContentCheck{{Path: documentPath, State: ContentNotReady, Detail: err.Error()}}, nil
	}

	metadata, found, metadataErr := parseDocumentMetadata(content)
	check := ContentCheck{Path: documentPath, State: ContentNotReady}
	switch {
	case metadataErr != nil:
		check.Detail = metadataErr.Error()
	case !found:
		check.Detail = "missing ctx:doc metadata"
	case metadata.Status == "draft":
		check.Detail = "draft"
	case metadata.Status == "not-applicable":
		check.State = ContentReady
		check.Detail = "not applicable"
	case metadata.Status == "verified":
		detail, validationErr := verifyDocumentMetadata(repo, folder, metadata)
		if validationErr != nil {
			return nil, validationErr
		}
		if detail == "" {
			check.State = ContentReady
			check.Detail = "verified at " + strings.TrimSpace(metadata.VerifiedAt)
		} else {
			check.Detail = detail
		}
	default:
		check.Detail = fmt.Sprintf("invalid ctx:doc status %q", metadata.Status)
	}

	checks := []ContentCheck{check}
	if warning, ok := contentSizeWarning(documentPath, content); ok {
		checks = append(checks, warning)
	}
	return checks, nil
}

func mechanicsSizeWarnings(dest, continuePath string) []ContentCheck {
	var checks []ContentCheck
	for _, documentPath := range []string{"INDEX.md", continuePath} {
		content, err := inspectDoctorFile(dest, documentPath)
		if err != nil {
			// Presence and file-type failures belong to Doctor. Status only adds
			// non-fatal size guidance for readable mechanics documents.
			continue
		}
		if warning, ok := contentSizeWarning(documentPath, content); ok {
			checks = append(checks, warning)
		}
	}
	return checks
}

func contentSizeWarning(documentPath string, content []byte) (ContentCheck, bool) {
	limit := documentWordWarningLimit(documentPath)
	count := documentWordCount(content)
	if limit == 0 || count <= limit {
		return ContentCheck{}, false
	}
	return ContentCheck{
		Path:   documentPath,
		State:  ContentWarning,
		Detail: fmt.Sprintf("%d words exceeds the %d-word routing guideline; split or route narrower context", count, limit),
	}, true
}

func verifyDocumentMetadata(repo, folder string, metadata documentMetadata) (string, error) {
	revision, _, detail := parseVerifiedAt(metadata.VerifiedAt)
	if detail != "" {
		return detail, nil
	}
	if len(metadata.Sources) == 0 {
		return "verified documents require at least one source path", nil
	}

	sources := make([]string, 0, len(metadata.Sources))
	seen := make(map[string]bool, len(metadata.Sources))
	for _, source := range metadata.Sources {
		normalized, err := validateStatusSourcePath(repo, folder, source)
		if err != nil {
			return err.Error(), nil
		}
		if !seen[normalized] {
			seen[normalized] = true
			sources = append(sources, normalized)
		}
	}
	sort.Strings(sources)

	commit, exists, err := resolveStatusRevision(repo, revision)
	if err != nil {
		return "", err
	}
	if !exists {
		return fmt.Sprintf("Git revision %q from verifiedAt does not exist", revision), nil
	}
	if !strings.HasPrefix(strings.ToLower(commit), strings.ToLower(revision)) {
		return fmt.Sprintf("Git revision %q did not resolve to the recorded commit hash", revision), nil
	}
	missing, recordedSymlinks, err := inspectStatusSourcesAtRevision(repo, commit, sources)
	if err != nil {
		return "", err
	}
	if len(missing) > 0 {
		return fmt.Sprintf("sources were not tracked at %s: %s", revision, strings.Join(missing, ", ")), nil
	}
	if len(recordedSymlinks) > 0 {
		return fmt.Sprintf("sources contain symbolic links at %s: %s", revision, strings.Join(recordedSymlinks, ", ")), nil
	}
	currentSymlinks, err := statusSourceSymlinksInWorktree(repo, sources)
	if err != nil {
		return "", err
	}
	if len(currentSymlinks) > 0 {
		return "sources contain symbolic links in the current worktree: " + strings.Join(currentSymlinks, ", "), nil
	}
	hidden, err := statusSourcesWithHiddenWorktreeChanges(repo, sources)
	if err != nil {
		return "", err
	}
	if len(hidden) > 0 {
		return "sources use assume-unchanged or skip-worktree flags that can hide changes: " + strings.Join(hidden, ", "), nil
	}
	changed, err := changedStatusSources(repo, commit, sources)
	if err != nil {
		return "", err
	}
	if len(changed) > 0 {
		return fmt.Sprintf("sources changed or are untracked since %s: %s", revision, strings.Join(changed, ", ")), nil
	}
	return "", nil
}

func statusSourcesWithHiddenWorktreeChanges(repo string, sources []string) ([]string, error) {
	pathspecs := make([]string, 0, len(sources))
	for _, source := range sources {
		pathspecs = append(pathspecs, ":(top,literal)"+source)
	}
	args := append([]string{"ls-files", "-v", "-z", "--"}, pathspecs...)
	output, err := gitCommand(repo, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("inspect verified-source index flags: %w", err)
	}
	var hidden []string
	for _, item := range bytes.Split(output, []byte{0}) {
		if len(item) < 3 || item[1] != ' ' {
			continue
		}
		tag := item[0]
		if tag == 'S' || (tag >= 'a' && tag <= 'z') {
			hidden = append(hidden, filepath.ToSlash(string(item[2:])))
		}
	}
	sort.Strings(hidden)
	return hidden, nil
}

func parseVerifiedAt(value string) (string, time.Time, string) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, " @ ")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", time.Time{}, "verified documents require verifiedAt in `<commit-hash> @ YYYY-MM-DD` form"
	}
	revision := parts[0]
	if len(revision) < 12 || len(revision) > 64 {
		return "", time.Time{}, "verifiedAt commit hash must be 12 to 64 hexadecimal characters (not a mutable ref such as HEAD)"
	}
	for _, char := range revision {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return "", time.Time{}, "verifiedAt commit hash must be hexadecimal (not a mutable ref such as HEAD)"
		}
	}
	const dateLayout = "2006-01-02"
	verifiedDate, err := time.Parse(dateLayout, parts[1])
	if err != nil || verifiedDate.Format(dateLayout) != parts[1] {
		return "", time.Time{}, "verifiedAt date must be a real calendar date in YYYY-MM-DD form"
	}
	return revision, verifiedDate, ""
}

func inspectStatusSourcesAtRevision(repo, revision string, sources []string) ([]string, []string, error) {
	var missing []string
	symlinks := make(map[string]bool)
	for _, source := range sources {
		pathspec := ":(top,literal)" + source
		output, err := gitCommand(repo, "ls-tree", "-r", "--full-tree", "-z", revision, "--", pathspec).Output()
		if err != nil {
			return nil, nil, fmt.Errorf("inspect verified source %s at %s: %w", source, revision, err)
		}
		if len(output) == 0 {
			missing = append(missing, source)
			continue
		}
		for _, record := range bytes.Split(output, []byte{0}) {
			if len(record) == 0 {
				continue
			}
			tab := bytes.IndexByte(record, '\t')
			if tab < 0 {
				return nil, nil, fmt.Errorf("inspect verified source %s at %s: malformed git ls-tree output", source, revision)
			}
			fields := bytes.Fields(record[:tab])
			if len(fields) < 3 {
				return nil, nil, fmt.Errorf("inspect verified source %s at %s: malformed git ls-tree metadata", source, revision)
			}
			if string(fields[0]) == "120000" {
				symlinks[filepath.ToSlash(string(record[tab+1:]))] = true
			}
		}
	}
	sort.Strings(missing)
	return missing, sortedStatusPaths(symlinks), nil
}

// statusSourceSymlinksInWorktree inspects every source component with Lstat,
// then walks real directory sources without following links. Missing or
// type-changed paths are left to changedStatusSources, which reports them as
// stale evidence rather than turning Status into an operational failure.
func statusSourceSymlinksInWorktree(repo string, sources []string) ([]string, error) {
	symlinks := make(map[string]bool)
	for _, source := range sources {
		parts := strings.Split(filepath.FromSlash(source), string(filepath.Separator))
		current := repo
		for index, part := range parts {
			current = filepath.Join(current, part)
			info, err := os.Lstat(current)
			if os.IsNotExist(err) {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("inspect verified source %s in current worktree: %w", source, err)
			}
			rel := filepath.ToSlash(filepath.Join(parts[:index+1]...))
			if info.Mode()&os.ModeSymlink != 0 {
				symlinks[rel] = true
				break
			}
			if index < len(parts)-1 {
				if !info.IsDir() {
					break
				}
				continue
			}
			if !info.IsDir() {
				break
			}
			if err := filepath.WalkDir(current, func(walkPath string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if walkPath == current || entry.Type()&os.ModeSymlink == 0 {
					return nil
				}
				rel, err := filepath.Rel(repo, walkPath)
				if err != nil {
					return err
				}
				symlinks[filepath.ToSlash(rel)] = true
				return nil
			}); err != nil {
				return nil, fmt.Errorf("inspect verified directory source %s in current worktree: %w", source, err)
			}
		}
	}
	return sortedStatusPaths(symlinks), nil
}

func sortedStatusPaths(paths map[string]bool) []string {
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	return ordered
}

func validateStatusSourcePath(repo, folder, source string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return "", fmt.Errorf("source paths must not be empty")
	}
	if strings.Contains(source, "\\") {
		return "", fmt.Errorf("source path %q must use forward slashes", source)
	}
	clean := path.Clean(source)
	native := filepath.FromSlash(clean)
	windowsDrive := len(clean) >= 2 && clean[1] == ':' && ((clean[0] >= 'a' && clean[0] <= 'z') || (clean[0] >= 'A' && clean[0] <= 'Z'))
	if path.IsAbs(source) || filepath.IsAbs(native) || filepath.VolumeName(native) != "" || windowsDrive || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("source path %q must be relative and contained in the repository", source)
	}
	absSource := filepath.Join(repo, native)
	rel, err := filepath.Rel(repo, absSource)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("source path %q escapes the repository", source)
	}
	cleanFolder := filepath.ToSlash(filepath.Clean(folder))
	if clean == cleanFolder || strings.HasPrefix(clean, cleanFolder+"/") {
		return "", fmt.Errorf("source path %q is inside the context folder", source)
	}
	return clean, nil
}

func resolveStatusRevision(repo, revision string) (string, bool, error) {
	cmd := gitCommand(repo, "rev-parse", "--verify", "--quiet", revision+"^{commit}")
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "", false, nil
	}
	return "", false, fmt.Errorf("resolve Git revision %q: %w", revision, err)
}

func changedStatusSources(repo, revision string, sources []string) ([]string, error) {
	pathspecs := make([]string, 0, len(sources))
	for _, source := range sources {
		pathspecs = append(pathspecs, ":(top,literal)"+source)
	}
	diffArgs := append([]string{"diff", "--no-ext-diff", "--no-textconv", "--name-only", "-z", revision, "--"}, pathspecs...)
	diffOutput, err := gitCommand(repo, diffArgs...).Output()
	if err != nil {
		return nil, fmt.Errorf("compare verified sources with %s: %w", revision, err)
	}
	otherArgs := append([]string{"ls-files", "--others", "-z", "--"}, pathspecs...)
	otherOutput, err := gitCommand(repo, otherArgs...).Output()
	if err != nil {
		return nil, fmt.Errorf("inspect untracked verified sources: %w", err)
	}
	changed := make(map[string]bool)
	for _, output := range [][]byte{diffOutput, otherOutput} {
		for _, item := range bytes.Split(output, []byte{0}) {
			if len(item) > 0 {
				changed[filepath.ToSlash(string(item))] = true
			}
		}
	}
	paths := make([]string, 0, len(changed))
	for changedPath := range changed {
		paths = append(paths, changedPath)
	}
	sort.Strings(paths)
	return paths, nil
}
