package ctx

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBehaviorAddonSupportsHierarchicalEvidenceAndPreservesProjectContent(t *testing.T) {
	for _, mode := range []Mode{ModeTeam, ModeLocal} {
		t.Run(string(mode), func(t *testing.T) {
			repo := mkRepo(t)
			if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: mode, Addons: []string{"glossary"}}); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(repo, ".ctx")
			cfg := addTestConfig(t, dest)
			cfg.TemplateRevision = "2.0.0" // The released layout-v2 template revision.
			if err := writeScaffoldConfig(dest, cfg); err != nil {
				t.Fatal(err)
			}
			if _, err := Update(repo, ".ctx"); err != nil {
				t.Fatalf("update existing scaffold before adoption: %v", err)
			}
			if got := addTestConfig(t, dest); got.TemplateRevision != CurrentTemplateRevision || !reflect.DeepEqual(got.Addons, []string{"glossary"}) {
				t.Fatalf("update changed selected add-ons or failed to advance revision: %+v", got)
			}
			if _, err := os.Lstat(filepath.Join(dest, "context", "behavior.md")); !os.IsNotExist(err) {
				t.Fatalf("update implicitly installed behavior: %v", err)
			}

			indexPath := filepath.Join(dest, "INDEX.md")
			ownerRoute := []byte("\n| Access decisions | `context/behavior/access.md` |\n")
			index := append(addTestRead(t, indexPath), ownerRoute...)
			if err := os.WriteFile(indexPath, index, 0o644); err != nil {
				t.Fatal(err)
			}
			glossaryPath := filepath.Join(dest, "context", "glossary.md")
			glossaryBefore := addTestRead(t, glossaryPath)

			result, err := Add(repo, ".ctx", []string{"behavior"})
			if err != nil {
				t.Fatalf("adopt behavior: %v", err)
			}
			if !reflect.DeepEqual(result.Addons, []string{"behavior"}) || !reflect.DeepEqual(result.Files, []string{"context/behavior.md"}) {
				t.Fatalf("behavior installation result = %+v", result)
			}
			if got := addTestConfig(t, dest).Addons; !reflect.DeepEqual(got, []string{"behavior", "glossary"}) {
				t.Fatalf("installed add-ons = %v, want behavior and glossary", got)
			}
			index = addTestRead(t, indexPath)
			if !bytes.Contains(index, []byte("`context/behavior.md`")) || !bytes.Contains(index, ownerRoute) {
				t.Fatalf("behavior routing or project-owned route missing:\n%s", index)
			}
			addTestEqualFile(t, glossaryPath, glossaryBefore)
			checks, err := Doctor(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			for _, check := range checks {
				if !check.OK {
					t.Errorf("doctor after behavior installation: %s: %s", check.Name, check.Detail)
				}
			}
			report, err := Status(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			if check := findStatusCheck(report, "context/behavior.md", ContentNotReady); check == nil || check.Detail != "draft" {
				t.Fatalf("new behavior readiness = %+v, want draft", check)
			}

			source := filepath.Join(repo, "src", "rules.go")
			writeStatusFile(t, source, "package rules\n\nconst Eligible = true\n")
			runStatusGit(t, repo, "add", "src/rules.go")
			runStatusGit(t, repo, "-c", "user.name=ctx-test", "-c", "user.email=ctx@example.invalid", "commit", "-qm", "record business rules")
			revision := strings.TrimSpace(runStatusGit(t, repo, "rev-parse", "HEAD"))
			metadata := documentMetadata{Status: "verified", VerifiedAt: revision + " @ 2026-10-07", Sources: []string{"src/rules.go"}}
			writeStatusDocument(t, repo, "context/behavior.md", metadata, "# Behavior\n\nCurrent behavior: eligible requests are accepted.\n\nSee [access decisions](behavior/access.md).\n")
			writeStatusDocument(t, repo, "context/behavior/access.md", metadata, "# Access decisions\n\nEligible requests are accepted.\n")
			behaviorPath := filepath.Join(dest, "context", "behavior.md")
			childPath := filepath.Join(dest, "context", "behavior", "access.md")
			behaviorBefore := addTestRead(t, behaviorPath)
			childBefore := addTestRead(t, childPath)
			if _, err := Update(repo, ".ctx"); err != nil {
				t.Fatal(err)
			}
			addTestEqualFile(t, behaviorPath, behaviorBefore)
			addTestEqualFile(t, childPath, childBefore)
			if !bytes.Contains(addTestRead(t, indexPath), ownerRoute) {
				t.Fatal("update removed project-owned behavior routing")
			}
			report, err = Status(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"context/behavior.md", "context/behavior/access.md"} {
				if check := findStatusCheck(report, path, ContentReady); check == nil || !strings.Contains(check.Detail, "verified at") {
					t.Fatalf("verified behavior document %s readiness = %+v", path, check)
				}
				if ignored := modeTestGitIgnored(t, repo, ".ctx/"+path); ignored != (mode == ModeLocal) {
					t.Fatalf("%s ignored = %v in %s mode", path, ignored, mode)
				}
			}

			writeStatusFile(t, source, "package rules\n\nconst Eligible = false\n")
			report, err = Status(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"context/behavior.md", "context/behavior/access.md"} {
				if check := findStatusCheck(report, path, ContentNotReady); check == nil || !strings.Contains(check.Detail, "src/rules.go") {
					t.Fatalf("changed behavior evidence %s readiness = %+v", path, check)
				}
			}
			if err := os.Remove(behaviorPath); err != nil {
				t.Fatal(err)
			}
			checks, err = Doctor(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			requireFailedDoctorCheckContaining(t, checks, "expected files present", "context/behavior.md")
		})
	}
}

func TestHydrationPreservesScaffoldCreatedBeforeDefaultBehavior(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam, Addons: []string{"glossary"}}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(repo, ".ctx")
	cfg := addTestConfig(t, dest)
	cfg.TemplateRevision = "2.0.0"
	if err := writeScaffoldConfig(dest, cfg); err != nil {
		t.Fatal(err)
	}
	durableBefore := make(map[string][]byte)
	for _, name := range []string{"README.md", "INDEX.md", configFileName, "context/glossary.md"} {
		durableBefore[name] = addTestRead(t, filepath.Join(dest, filepath.FromSlash(name)))
	}
	if err := os.Remove(filepath.Join(dest, "local", "CONTINUE.md")); err != nil {
		t.Fatal(err)
	}
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("hydrate existing scaffold with new implicit defaults: %v", err)
	}
	for name, content := range durableBefore {
		addTestEqualFile(t, filepath.Join(dest, filepath.FromSlash(name)), content)
	}
	if _, err := os.Lstat(filepath.Join(dest, "context", "behavior.md")); !os.IsNotExist(err) {
		t.Fatalf("hydration implicitly installed behavior: %v", err)
	}
	if !modeTestGitIgnored(t, repo, ".ctx/local/CONTINUE.md") {
		t.Fatal("hydrated continuation is visible to Git")
	}
}
