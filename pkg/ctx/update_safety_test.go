package ctx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpdateRejectsMalformedMarkersWithoutPartialMutation(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "dangling begin", content: managedBegin + "\nunterminated\n"},
		{name: "nested begin", content: managedBegin + "\nouter\n" + managedBegin + "\ninner\n" + managedEnd + "\n" + managedEnd + "\n"},
		{name: "end before begin", content: managedEnd + "\n" + managedBegin + "\ncontent\n" + managedEnd + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, dest := newUpdateSafetyRepo(t)
			readmePath := filepath.Join(dest, "README.md")
			reviewPath := filepath.Join(dest, "REVIEW.md")
			versionPath := filepath.Join(dest, ".ctx-version")
			readmeBefore := makeManagedOutputStale(t, readmePath)
			writeUpdateSafetyFile(t, reviewPath, []byte(tt.content), 0o644)
			versionBefore := []byte("preflight-sentinel\n")
			writeUpdateSafetyFile(t, versionPath, versionBefore, 0o644)

			touched, err := Update(repo, ".ctx")
			if err == nil || !strings.Contains(err.Error(), "malformed managed-marker grammar") {
				t.Fatalf("Update error = %v, want malformed-marker rejection", err)
			}
			if len(touched) != 0 {
				t.Fatalf("failed Update reported touched outputs: %v", touched)
			}
			requireUpdateSafetyContents(t, readmePath, readmeBefore)
			requireUpdateSafetyContents(t, reviewPath, []byte(tt.content))
			requireUpdateSafetyContents(t, versionPath, versionBefore)
			requireNoUpdateTemps(t, dest)
		})
	}
}

func TestUpdateRejectsSymlinkedScaffold(t *testing.T) {
	repo, dest := newUpdateSafetyRepo(t)
	realDest := filepath.Join(repo, ".ctx-real")
	if err := os.Rename(dest, realDest); err != nil {
		t.Fatal(err)
	}
	requireUpdateSafetySymlink(t, filepath.Base(realDest), dest)
	readmePath := filepath.Join(realDest, "README.md")
	versionPath := filepath.Join(realDest, ".ctx-version")
	readmeBefore := readUpdateSafetyFile(t, readmePath)
	versionBefore := readUpdateSafetyFile(t, versionPath)

	if touched, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Update = (%v, %v), want scaffold-symlink rejection", touched, err)
	}
	requireUpdateSafetyContents(t, readmePath, readmeBefore)
	requireUpdateSafetyContents(t, versionPath, versionBefore)
	requireNoUpdateTemps(t, realDest)
}

func TestUpdateRejectsNestedScaffoldSymlinkOutsideRepo(t *testing.T) {
	repo, dest := newUpdateSafetyRepo(t)
	outsideRoot := t.TempDir()
	outsideDest := filepath.Join(outsideRoot, "context")
	if err := os.Rename(dest, outsideDest); err != nil {
		t.Skipf("cannot move scaffold for nested-symlink test: %v", err)
	}
	linkedParent := filepath.Join(repo, "linked")
	requireUpdateSafetySymlink(t, outsideRoot, linkedParent)
	readmePath := filepath.Join(outsideDest, "README.md")
	readmeBefore := readUpdateSafetyFile(t, readmePath)

	folder := filepath.Join("linked", "context")
	if touched, err := Update(repo, folder); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Update = (%v, %v), want nested-symlink rejection", touched, err)
	}
	requireUpdateSafetyContents(t, readmePath, readmeBefore)
	requireNoUpdateTemps(t, outsideDest)
}

func TestUpdateRejectsSymlinkedManagedOutputWithoutOutsideWrite(t *testing.T) {
	repo, dest := newUpdateSafetyRepo(t)
	readmePath := filepath.Join(dest, "README.md")
	reviewPath := filepath.Join(dest, "REVIEW.md")
	versionPath := filepath.Join(dest, ".ctx-version")
	readmeBefore := makeManagedOutputStale(t, readmePath)
	versionBefore := []byte("managed-symlink-sentinel\n")
	writeUpdateSafetyFile(t, versionPath, versionBefore, 0o644)
	outsidePath := filepath.Join(t.TempDir(), "outside-review.md")
	outsideBefore := []byte("outside must not change\n")
	writeUpdateSafetyFile(t, outsidePath, outsideBefore, 0o644)
	if err := os.Remove(reviewPath); err != nil {
		t.Fatal(err)
	}
	requireUpdateSafetySymlink(t, outsidePath, reviewPath)

	if touched, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "symbolic-link output") {
		t.Fatalf("Update = (%v, %v), want managed-output symlink rejection", touched, err)
	}
	requireUpdateSafetyContents(t, readmePath, readmeBefore)
	requireUpdateSafetyContents(t, versionPath, versionBefore)
	requireUpdateSafetyContents(t, outsidePath, outsideBefore)
	requireNoUpdateTemps(t, dest)
}

func TestUpdateRejectsSymlinkedVersionWithoutManagedMutation(t *testing.T) {
	repo, dest := newUpdateSafetyRepo(t)
	readmePath := filepath.Join(dest, "README.md")
	versionPath := filepath.Join(dest, ".ctx-version")
	readmeBefore := makeManagedOutputStale(t, readmePath)
	outsidePath := filepath.Join(t.TempDir(), "outside-version")
	outsideBefore := []byte("outside-version-sentinel\n")
	writeUpdateSafetyFile(t, outsidePath, outsideBefore, 0o644)
	if err := os.Remove(versionPath); err != nil {
		t.Fatal(err)
	}
	requireUpdateSafetySymlink(t, outsidePath, versionPath)

	if touched, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "symbolic-link output") {
		t.Fatalf("Update = (%v, %v), want version symlink rejection", touched, err)
	}
	requireUpdateSafetyContents(t, readmePath, readmeBefore)
	requireUpdateSafetyContents(t, outsidePath, outsideBefore)
	requireNoUpdateTemps(t, dest)
}

func TestRollbackDoesNotFollowSymlinkedOutputParent(t *testing.T) {
	dest := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "review.md")
	outsideBefore := []byte("published-looking external content\n")
	writeUpdateSafetyFile(t, outsidePath, outsideBefore, 0o644)
	outsideInfo, err := os.Stat(outsidePath)
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(dest, "workflows")
	requireUpdateSafetySymlink(t, outsideDir, parent)
	plans := []*updateOutput{{
		name:         "workflows/review.md",
		scaffoldRoot: dest,
		path:         filepath.Join(parent, "review.md"),
		content:      outsideBefore,
		stagedInfo:   outsideInfo,
		published:    true,
	}}

	err = rollbackPublishedUpdates(plans, errors.New("trigger rollback"))
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("rollback error = %v, want parent-symlink refusal", err)
	}
	requireUpdateSafetyContents(t, outsidePath, outsideBefore)
}

func TestUpdateRejectsNonRegularOutputsWithoutPartialMutation(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{name: "managed directory", output: "REVIEW.md"},
		{name: "version directory", output: ".ctx-version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, dest := newUpdateSafetyRepo(t)
			readmePath := filepath.Join(dest, "README.md")
			readmeBefore := makeManagedOutputStale(t, readmePath)
			outputPath := filepath.Join(dest, tt.output)
			if err := os.Remove(outputPath); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(outputPath, 0o755); err != nil {
				t.Fatal(err)
			}
			var versionBefore []byte
			if tt.output != ".ctx-version" {
				versionPath := filepath.Join(dest, ".ctx-version")
				versionBefore = []byte("type-error-sentinel\n")
				writeUpdateSafetyFile(t, versionPath, versionBefore, 0o644)
			}

			if touched, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "not a regular file") {
				t.Fatalf("Update = (%v, %v), want non-regular-output rejection", touched, err)
			}
			requireUpdateSafetyContents(t, readmePath, readmeBefore)
			if tt.output != ".ctx-version" {
				requireUpdateSafetyContents(t, filepath.Join(dest, ".ctx-version"), versionBefore)
			}
			if info, err := os.Lstat(outputPath); err != nil || !info.IsDir() {
				t.Fatalf("non-regular output changed: info=%v err=%v", info, err)
			}
			requireNoUpdateTemps(t, dest)
		})
	}
}

func TestUpdateDoesNotTreatReadErrorAsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows permissions do not provide a deterministic owner-read denial")
	}
	repo, dest := newUpdateSafetyRepo(t)
	readmePath := filepath.Join(dest, "README.md")
	reviewPath := filepath.Join(dest, "REVIEW.md")
	versionPath := filepath.Join(dest, ".ctx-version")
	readmeBefore := makeManagedOutputStale(t, readmePath)
	reviewBefore := readUpdateSafetyFile(t, reviewPath)
	versionBefore := []byte("read-error-sentinel\n")
	writeUpdateSafetyFile(t, versionPath, versionBefore, 0o644)
	if err := os.Chmod(reviewPath, 0); err != nil {
		t.Fatal(err)
	}
	if f, err := os.Open(reviewPath); err == nil {
		_ = f.Close()
		_ = os.Chmod(reviewPath, 0o644)
		t.Skip("test process can read mode-000 files")
	}

	touched, updateErr := Update(repo, ".ctx")
	if err := os.Chmod(reviewPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if updateErr == nil || !strings.Contains(updateErr.Error(), "read") {
		t.Fatalf("Update = (%v, %v), want read error", touched, updateErr)
	}
	requireUpdateSafetyContents(t, readmePath, readmeBefore)
	requireUpdateSafetyContents(t, reviewPath, reviewBefore)
	requireUpdateSafetyContents(t, versionPath, versionBefore)
	requireNoUpdateTemps(t, dest)
}

func TestUpdateSkipsMissingManagedAndCreatesMissingVersion(t *testing.T) {
	repo, dest := newUpdateSafetyRepo(t)
	readmePath := filepath.Join(dest, "README.md")
	reviewPath := filepath.Join(dest, "REVIEW.md")
	versionPath := filepath.Join(dest, ".ctx-version")
	readmeBefore := makeManagedOutputStale(t, readmePath)
	if err := os.Remove(reviewPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(versionPath); err != nil {
		t.Fatal(err)
	}

	touched, err := Update(repo, ".ctx")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(touched) != 1 || !strings.HasPrefix(touched[0], "README.md") {
		t.Fatalf("touched = %v, want README.md only", touched)
	}
	if bytes.Equal(readUpdateSafetyFile(t, readmePath), readmeBefore) {
		t.Fatal("managed README was not refreshed")
	}
	if _, err := os.Lstat(reviewPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing managed file was recreated: %v", err)
	}
	requireUpdateSafetyContents(t, versionPath, []byte(Version+"\n"))
	requireNoUpdateTemps(t, dest)
}

func TestUpdateAtomicallyReplacesManagedOutputAndPreservesMode(t *testing.T) {
	repo, dest := newUpdateSafetyRepo(t)
	readmePath := filepath.Join(dest, "README.md")
	makeManagedOutputStale(t, readmePath)
	if err := os.Chmod(readmePath, 0o600); err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Lstat(readmePath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Update(repo, ".ctx"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	afterInfo, err := os.Lstat(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(beforeInfo, afterInfo) {
		t.Fatal("managed output was modified in place instead of atomically replaced")
	}
	if runtime.GOOS != "windows" && afterInfo.Mode().Perm() != 0o600 {
		t.Fatalf("managed output mode = %o, want 600", afterInfo.Mode().Perm())
	}
	requireNoUpdateTemps(t, dest)
}

func TestPublishUpdateOutputsRollsBackEarlierReplacement(t *testing.T) {
	dest := t.TempDir()
	readmePath := filepath.Join(dest, "README.md")
	versionPath := filepath.Join(dest, ".ctx-version")
	readmeBefore := []byte("original readme\n")
	versionBefore := []byte("original version\n")
	writeUpdateSafetyFile(t, readmePath, readmeBefore, 0o644)
	writeUpdateSafetyFile(t, versionPath, versionBefore, 0o644)
	readme, err := inspectUpdateOutput(readmePath, false)
	if err != nil {
		t.Fatal(err)
	}
	version, err := inspectUpdateOutput(versionPath, false)
	if err != nil {
		t.Fatal(err)
	}
	plans := []*updateOutput{
		{
			name:         "README.md",
			path:         readmePath,
			content:      []byte("updated readme\n"),
			mode:         readme.info.Mode().Perm(),
			existed:      true,
			original:     readme.data,
			originalInfo: readme.info,
		},
		{
			name:         ".ctx-version",
			path:         versionPath,
			content:      []byte("updated version\n"),
			mode:         version.info.Mode().Perm(),
			existed:      true,
			original:     version.data,
			originalInfo: version.info,
		},
	}
	if err := stageUpdateOutputs(dest, plans); err != nil {
		t.Fatal(err)
	}
	defer cleanupUpdateTemps(plans)
	if err := validateUpdateOutputsUnchanged(plans); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(versionPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(versionPath, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := publishUpdateOutputs(plans); err == nil {
		t.Fatal("publish succeeded after the version output changed type")
	}
	requireUpdateSafetyContents(t, readmePath, readmeBefore)
	if info, err := os.Lstat(versionPath); err != nil || !info.IsDir() {
		t.Fatalf("concurrently changed version output was overwritten: info=%v err=%v", info, err)
	}
	cleanupUpdateTemps(plans)
	requireNoUpdateTemps(t, dest)
}

func newUpdateSafetyRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := mkRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return repo, filepath.Join(repo, ".ctx")
}

func makeManagedOutputStale(t *testing.T, path string) []byte {
	t.Helper()
	content := readUpdateSafetyFile(t, path)
	stale := bytes.Replace(content, []byte(managedBegin+"\n"), []byte(managedBegin+"\nSTALE-MANAGED-CONTENT\n"), 1)
	if bytes.Equal(stale, content) {
		t.Fatalf("could not make %s stale", path)
	}
	writeUpdateSafetyFile(t, path, stale, 0o644)
	return stale
}

func requireUpdateSafetySymlink(t *testing.T, target, path string) {
	t.Helper()
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
}

func readUpdateSafetyFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func writeUpdateSafetyFile(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func requireUpdateSafetyContents(t *testing.T, path string, want []byte) {
	t.Helper()
	if got := readUpdateSafetyFile(t, path); !bytes.Equal(got, want) {
		t.Fatalf("%s changed\n got: %q\nwant: %q", path, got, want)
	}
}

func requireNoUpdateTemps(t *testing.T, dest string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dest, ".ctx-update-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("leftover update temp files: %v", matches)
	}
}
