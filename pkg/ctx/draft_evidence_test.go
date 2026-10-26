package ctx

import (
	"path/filepath"
	"testing"
)

func TestDraftFactsRetainUncommittedEvidenceAcrossManagedUpdate(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}
	writeStatusFile(t, filepath.Join(repo, "src", "model.go"), "package src\n")
	runStatusGit(t, repo, "add", "src/model.go")
	runStatusGit(t, repo, "-c", "user.name=ctx-test", "-c", "user.email=ctx@example.invalid", "commit", "-qm", "record supporting source")
	writeStatusFile(t, filepath.Join(repo, "src", "model.go"), "package src\n\nconst Changed = true\n")
	metadata := documentMetadata{Status: "draft", Sources: []string{"src/model.go"}}
	writeStatusDocument(t, repo, "context/caveats.md", metadata,
		"# Caveats\n\nCurrent behavior follows src/model.go. Owner decision not documented.\n")
	path := filepath.Join(repo, ".ctx", "context", "caveats.md")
	before := addTestRead(t, path)
	sourcePath := filepath.Join(repo, "src", "model.go")
	sourceBefore := addTestRead(t, sourcePath)
	diffBefore := runStatusGit(t, repo, "diff", "--", "src/model.go")

	report, err := Status(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	check := findStatusCheck(report, "context/caveats.md", ContentNotReady)
	if report.Ready() || check == nil || check.Detail != "draft" {
		t.Fatalf("uncommitted draft with retained evidence = %+v, want draft readiness", report)
	}
	if _, err := Update(repo, ".ctx"); err != nil {
		t.Fatal(err)
	}
	addTestEqualFile(t, path, before)
	addTestEqualFile(t, sourcePath, sourceBefore)
	if diff := runStatusGit(t, repo, "diff", "--", "src/model.go"); diff == "" || diff != diffBefore {
		t.Fatal("managed update changed the uncommitted supporting diff")
	}
	if staged := runStatusGit(t, repo, "diff", "--cached"); staged != "" {
		t.Fatal("managed update staged the supporting evidence")
	}
}
