package ctx

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStatusFindsNestedDocumentsAndVerifiesSources(t *testing.T) {
	repo, revision := setupStatusRepo(t)
	writeStatusDocument(t, repo, "context/domain/terms.md", documentMetadata{
		Status:     "verified",
		VerifiedAt: revision + " @ 2026-08-31",
		Sources:    []string{"src/model.go"},
	}, "# Domain terms\n")

	report, err := Status(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready() {
		t.Fatalf("Status reported ready content as unready: %+v", report.Checks)
	}
	check := findStatusCheck(report, "context/domain/terms.md", ContentReady)
	if check == nil || !strings.Contains(check.Detail, "verified at") {
		t.Fatalf("nested document check = %+v", check)
	}
}

func TestStatusIncludesMissingRequiredProjectFacts(t *testing.T) {
	repo, _ := setupStatusRepo(t)
	missing := filepath.Join(repo, ".ctx", "context", "architecture.md")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}

	report, err := Status(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	check := findStatusCheck(report, "context/architecture.md", ContentNotReady)
	if check == nil || !strings.Contains(check.Detail, "no such file") {
		t.Fatalf("missing required fact check = %+v", check)
	}
}

func TestStatusMetadataStates(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantReady  bool
		wantDetail string
	}{
		{name: "missing metadata", content: "# Overview\n", wantDetail: "missing ctx:doc metadata"},
		{name: "malformed metadata", content: "<!-- ctx:doc not-json -->\n", wantDetail: "parse ctx:doc metadata"},
		{name: "draft", content: statusDocument(documentMetadata{Status: "draft"}, "# Overview\n"), wantDetail: "draft"},
		{name: "unknown status", content: statusDocument(documentMetadata{Status: "obsolete"}, "# Overview\n"), wantDetail: "invalid ctx:doc status"},
		{name: "not applicable", content: statusDocument(documentMetadata{Status: "not-applicable"}, "# Overview\n"), wantReady: true, wantDetail: "not applicable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, _ := setupStatusRepo(t)
			writeStatusFile(t, filepath.Join(repo, ".ctx", "context", "overview.md"), test.content)
			report, err := Status(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			state := ContentNotReady
			if test.wantReady {
				state = ContentReady
			}
			check := findStatusCheck(report, "context/overview.md", state)
			if check == nil || !strings.Contains(check.Detail, test.wantDetail) {
				t.Fatalf("overview check = %+v, want state %s containing %q", check, state, test.wantDetail)
			}
		})
	}
}

func TestStatusVerifiedMetadataRequirements(t *testing.T) {
	tests := []struct {
		name       string
		metadata   documentMetadata
		wantDetail string
	}{
		{name: "missing revision", metadata: documentMetadata{Status: "verified", Sources: []string{"src/model.go"}}, wantDetail: "require verifiedAt"},
		{name: "mutable revision", metadata: documentMetadata{Status: "verified", VerifiedAt: "HEAD @ 2026-08-31", Sources: []string{"src/model.go"}}, wantDetail: "not a mutable ref"},
		{name: "invalid date", metadata: documentMetadata{Status: "verified", VerifiedAt: "VALID @ 2026-02-30", Sources: []string{"src/model.go"}}, wantDetail: "real calendar date"},
		{name: "missing sources", metadata: documentMetadata{Status: "verified", VerifiedAt: "VALID @ 2026-08-31"}, wantDetail: "at least one source"},
		{name: "unknown revision", metadata: documentMetadata{Status: "verified", VerifiedAt: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef @ 2026-08-31", Sources: []string{"src/model.go"}}, wantDetail: "does not exist"},
		{name: "escaping source", metadata: documentMetadata{Status: "verified", VerifiedAt: "VALID @ 2026-08-31", Sources: []string{"../outside"}}, wantDetail: "contained in the repository"},
		{name: "drive source", metadata: documentMetadata{Status: "verified", VerifiedAt: "VALID @ 2026-08-31", Sources: []string{"C:/outside"}}, wantDetail: "contained in the repository"},
		{name: "self source", metadata: documentMetadata{Status: "verified", VerifiedAt: "VALID @ 2026-08-31", Sources: []string{".ctx/context/overview.md"}}, wantDetail: "inside the context folder"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, revision := setupStatusRepo(t)
			metadata := test.metadata
			metadata.VerifiedAt = strings.Replace(metadata.VerifiedAt, "VALID", revision, 1)
			writeStatusDocument(t, repo, "context/overview.md", metadata, "# Overview\n")
			report, err := Status(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			check := findStatusCheck(report, "context/overview.md", ContentNotReady)
			if check == nil || !strings.Contains(check.Detail, test.wantDetail) {
				t.Fatalf("overview check = %+v, want detail containing %q", check, test.wantDetail)
			}
		})
	}
}

func TestStatusFlagsTrackedAndUntrackedSources(t *testing.T) {
	t.Run("tracked change", func(t *testing.T) {
		repo, revision := setupStatusRepo(t)
		writeStatusDocument(t, repo, "context/overview.md", documentMetadata{
			Status: "verified", VerifiedAt: revision + " @ 2026-08-31", Sources: []string{"src/model.go"},
		}, "# Overview\n")
		writeStatusFile(t, filepath.Join(repo, "src", "model.go"), "package src\n\nconst Changed = true\n")

		report, err := Status(repo, ".ctx")
		if err != nil {
			t.Fatal(err)
		}
		check := findStatusCheck(report, "context/overview.md", ContentNotReady)
		if check == nil || !strings.Contains(check.Detail, "src/model.go") {
			t.Fatalf("tracked-source check = %+v", check)
		}
	})

	t.Run("untracked source, including ignored files", func(t *testing.T) {
		repo, revision := setupStatusRepo(t)
		writeStatusFile(t, filepath.Join(repo, ".gitignore"), "notes.txt\n")
		writeStatusFile(t, filepath.Join(repo, "notes.txt"), "new facts\n")
		writeStatusDocument(t, repo, "context/overview.md", documentMetadata{
			Status: "verified", VerifiedAt: revision + " @ 2026-08-31", Sources: []string{"notes.txt"},
		}, "# Overview\n")

		report, err := Status(repo, ".ctx")
		if err != nil {
			t.Fatal(err)
		}
		check := findStatusCheck(report, "context/overview.md", ContentNotReady)
		if check == nil || !strings.Contains(check.Detail, "notes.txt") {
			t.Fatalf("untracked-source check = %+v", check)
		}
	})
}

func TestStatusFailsClosedForIndexFlagsThatHideWorktreeChanges(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(strings.TrimPrefix(flag, "--"), func(t *testing.T) {
			repo, revision := setupStatusRepo(t)
			writeStatusDocument(t, repo, "context/overview.md", documentMetadata{
				Status:     "verified",
				VerifiedAt: revision + " @ 2026-08-31",
				Sources:    []string{"src/model.go"},
			}, "# Overview\n")
			runStatusGit(t, repo, "update-index", flag, "src/model.go")
			writeStatusFile(t, filepath.Join(repo, "src", "model.go"), "package src\n\nconst HiddenChange = true\n")

			report, err := Status(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			check := findStatusCheck(report, "context/overview.md", ContentNotReady)
			if check == nil || !strings.Contains(check.Detail, "can hide changes") || !strings.Contains(check.Detail, "src/model.go") {
				t.Fatalf("hidden-worktree-change check = %+v", check)
			}
		})
	}
}

func TestStatusRejectsSourceThatDidNotExistAtVerifiedRevision(t *testing.T) {
	repo, revision := setupStatusRepo(t)
	writeStatusDocument(t, repo, "context/overview.md", documentMetadata{
		Status: "verified", VerifiedAt: revision + " @ 2026-08-31", Sources: []string{"never-existed.md"},
	}, "# Overview\n")

	report, err := Status(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	check := findStatusCheck(report, "context/overview.md", ContentNotReady)
	if check == nil || !strings.Contains(check.Detail, "not tracked") || !strings.Contains(check.Detail, "never-existed.md") {
		t.Fatalf("never-tracked source check = %+v", check)
	}
}

func TestStatusDoesNotFollowSymbolicLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation is not generally available to unprivileged Windows tests")
	}
	repo, _ := setupStatusRepo(t)
	outside := filepath.Join(repo, "outside.md")
	writeStatusFile(t, outside, statusDocument(documentMetadata{Status: "not-applicable"}, "# Outside\n"))
	link := filepath.Join(repo, ".ctx", "context", "linked.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	report, err := Status(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	check := findStatusCheck(report, "context/linked.md", ContentNotReady)
	if check == nil || !strings.Contains(check.Detail, "not followed") {
		t.Fatalf("symbolic-link check = %+v", check)
	}
}

func TestStatusRejectsSymlinkedScaffoldParentOutsideRepo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation is not generally available to unprivileged Windows tests")
	}
	target := mkRepo(t)
	outside, _ := setupStatusRepo(t)
	if err := os.Symlink(outside, filepath.Join(target, "linked")); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if report, err := Status(target, "linked/.ctx"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Status = (%+v, %v), want nested scaffold symlink rejection", report, err)
	}
}

func TestStatusSizeWarningsAreNonFatal(t *testing.T) {
	repo, _ := setupStatusRepo(t)
	writeStatusDocument(t, repo, "context/large.md", documentMetadata{Status: "not-applicable"}, strings.Repeat("word ", 1601))

	report, err := Status(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready() {
		t.Fatalf("size warning made report unready: %+v", report.Checks)
	}
	if check := findStatusCheck(report, "context/large.md", ContentWarning); check == nil {
		t.Fatalf("missing size warning: %+v", report.Checks)
	}
	var output bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"status", repo})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ctx status failed for warning-only content: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "! context/large.md") {
		t.Fatalf("status output missing warning marker:\n%s", output.String())
	}
}

func TestStatusWarnsAboutLargeIndexAndContinuationWithoutMetadata(t *testing.T) {
	repo, _ := setupStatusRepo(t)
	writeStatusFile(t, filepath.Join(repo, ".ctx", "INDEX.md"), strings.Repeat("route ", 501))
	writeStatusFile(t, filepath.Join(repo, ".ctx", "local", "CONTINUE.md"), strings.Repeat("session ", 601))

	report, err := Status(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Ready() {
		t.Fatalf("mechanics size warnings made report unready: %+v", report.Checks)
	}
	for _, documentPath := range []string{"INDEX.md", "local/CONTINUE.md"} {
		if check := findStatusCheck(report, documentPath, ContentWarning); check == nil {
			t.Errorf("missing size warning for %s: %+v", documentPath, report.Checks)
		}
		if check := findStatusCheck(report, documentPath, ContentNotReady); check != nil {
			t.Errorf("mechanics document %s received metadata failure: %+v", documentPath, check)
		}
	}
}

func TestStatusLayoutV1IsInformativelyNotReady(t *testing.T) {
	repo := mkRepo(t)
	dest := filepath.Join(repo, ".ctx")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeScaffoldConfig(dest, Config{SchemaVersion: 1, Mode: ModeLocal}); err != nil {
		t.Fatal(err)
	}

	report, err := Status(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready() || len(report.Checks) != 1 || !strings.Contains(report.Checks[0].Detail, "layout v1") {
		t.Fatalf("v1 report = %+v", report)
	}
}

func TestStatusCommandPrintsFindingsAndFailsForUnreadyContent(t *testing.T) {
	repo, _ := setupStatusRepo(t)
	writeStatusDocument(t, repo, "context/overview.md", documentMetadata{Status: "draft"}, "# Overview\n")
	var output bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"status", repo})

	if err := cmd.Execute(); err == nil {
		t.Fatal("ctx status succeeded for draft content")
	}
	if !strings.Contains(output.String(), "✗ context/overview.md") || !strings.Contains(output.String(), "✓ context/architecture.md") {
		t.Fatalf("status output missing concise findings:\n%s", output.String())
	}
}

func setupStatusRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := mkRepo(t)
	dest := filepath.Join(repo, ".ctx")
	if err := os.MkdirAll(filepath.Join(dest, "context"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeScaffoldConfig(dest, Config{
		SchemaVersion:    currentSchemaVersion,
		LayoutVersion:    CurrentLayoutVersion,
		TemplateRevision: CurrentTemplateRevision,
		Project:          "status-test",
		Mode:             ModeTeam,
	}); err != nil {
		t.Fatal(err)
	}
	facts, err := ProjectFactDocuments(CurrentLayoutVersion, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, fact := range facts {
		writeStatusDocument(t, repo, fact.Path, documentMetadata{Status: "not-applicable"}, "# Fact\n")
	}
	writeStatusFile(t, filepath.Join(repo, "src", "model.go"), "package src\n")
	runStatusGit(t, repo, "add", "src/model.go")
	runStatusGit(t, repo, "-c", "user.name=ctx-test", "-c", "user.email=ctx@example.invalid", "commit", "-qm", "status baseline")
	return repo, strings.TrimSpace(runStatusGit(t, repo, "rev-parse", "HEAD"))
}

func writeStatusDocument(t *testing.T, repo, documentPath string, metadata documentMetadata, body string) {
	t.Helper()
	writeStatusFile(t, filepath.Join(repo, ".ctx", filepath.FromSlash(documentPath)), statusDocument(metadata, body))
}

func statusDocument(metadata documentMetadata, body string) string {
	data, err := json.Marshal(metadata)
	if err != nil {
		panic(err)
	}
	return "<!-- ctx:doc " + string(data) + " -->\n" + body
}

func writeStatusFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runStatusGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func findStatusCheck(report ContentStatus, documentPath string, state ContentState) *ContentCheck {
	for i := range report.Checks {
		if report.Checks[i].Path == documentPath && report.Checks[i].State == state {
			return &report.Checks[i]
		}
	}
	return nil
}
