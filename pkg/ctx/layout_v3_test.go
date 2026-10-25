package ctx

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// initFlatWithOptionsForTest materializes a previously released scaffold.
// It deliberately renders the frozen catalog rather than changing production
// creation defaults or testing old lifecycle operations on a v3 descriptor.
func initFlatWithOptionsForTest(t *testing.T, repo string, raw InitOptions) error {
	t.Helper()
	opts, err := normalizeInitOptions(raw)
	if err != nil {
		return err
	}
	if err := InitWithOptions(repo, opts); err != nil {
		return err
	}
	dest := filepath.Join(repo, opts.Folder)
	facts, _ := ProjectFactDocuments(CurrentLayoutVersion, opts.Addons)
	for _, fact := range facts {
		path := filepath.Join(dest, filepath.FromSlash(fact.Path))
		if err := os.Remove(path); err != nil {
			return err
		}
		if err := os.Remove(filepath.Dir(path)); err != nil {
			return err
		}
	}
	routes, _ := addonRoutesMarkdown(FlatLayoutVersion, opts.Addons)
	documents, _ := DocumentsForLayout(FlatLayoutVersion, opts.Addons)
	for _, document := range documents {
		data, err := readTemplateAsset(document.TemplatePath)
		if err != nil {
			return err
		}
		content := renderTemplate(string(data), templateValues{Project: filepath.Base(repo), Date: "2026-10-25", Folder: opts.Folder, Mode: string(opts.Mode), ContinuePath: "local/CONTINUE.md", AddonRoutes: routes})
		path := filepath.Join(dest, filepath.FromSlash(document.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return writeScaffoldConfig(dest, Config{SchemaVersion: 2, LayoutVersion: FlatLayoutVersion, TemplateRevision: "2.0.1", Project: filepath.Base(repo), Mode: opts.Mode, Addons: opts.Addons})
}

func TestFolderFirstDefaultsAndTopicReadiness(t *testing.T) {
	for _, mode := range []Mode{ModeTeam, ModeLocal} {
		t.Run(string(mode), func(t *testing.T) {
			repo := mkRepo(t)
			if err := InitWithOptions(repo, InitOptions{Mode: mode}); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(repo, ".ctx")
			cfg := addTestConfig(t, dest)
			if cfg.LayoutVersion != 3 || cfg.SchemaVersion != 2 || cfg.TemplateRevision != "3.0.0" {
				t.Fatalf("new config = %+v", cfg)
			}
			for _, category := range []string{"overview", "architecture", "caveats", "behavior", "glossary"} {
				entries, err := os.ReadDir(filepath.Join(dest, "context", category))
				if err != nil || len(entries) != 1 || entries[0].Name() != "index.md" {
					t.Fatalf("%s must contain only its generated index: %v %v", category, entries, err)
				}
				if _, err := os.Lstat(filepath.Join(dest, "context", category+".md")); !os.IsNotExist(err) {
					t.Fatalf("flat %s emitted: %v", category, err)
				}
				if modeTestGitIgnored(t, repo, ".ctx/context/"+category+"/index.md") != (mode == ModeLocal) {
					t.Fatalf("incorrect %s visibility", category)
				}
				writeStatusDocument(t, repo, "context/"+category+"/index.md", documentMetadata{Status: "not-applicable"}, "# Summary\n")
			}
			writeStatusFile(t, filepath.Join(repo, "src/model.go"), "package src\n")
			runStatusGit(t, repo, "add", "src/model.go")
			runStatusGit(t, repo, "-c", "user.name=ctx-test", "-c", "user.email=ctx@example.invalid", "commit", "-qm", "record index evidence")
			commit := strings.TrimSpace(runStatusGit(t, repo, "rev-parse", "HEAD"))
			writeStatusDocument(t, repo, "context/glossary/index.md", documentMetadata{Status: "verified", VerifiedAt: commit + " @ 2026-10-25", Sources: []string{"src/model.go"}}, "# Glossary\n\nSee [hydration](hydration.md).\n")
			writeStatusDocument(t, repo, "context/glossary/hydration.md", documentMetadata{Status: "draft"}, "# Hydration\n")
			report, err := Status(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			if findStatusCheck(report, "context/glossary/index.md", ContentReady) == nil || findStatusCheck(report, "context/glossary/hydration.md", ContentNotReady) == nil || report.Ready() {
				t.Fatalf("index readiness must not certify its child: %+v", report)
			}
			before := addTestRead(t, filepath.Join(dest, "context/glossary/hydration.md"))
			if _, err := Update(repo, ".ctx"); err != nil {
				t.Fatal(err)
			}
			addTestEqualFile(t, filepath.Join(dest, "context/glossary/hydration.md"), before)
			if _, err := Add(repo, ".ctx", []string{"contracts", "review"}); err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"context/contracts/index.md", "workflows/review.md"} {
				_ = addTestRead(t, filepath.Join(dest, path))
			}
			for _, check := range mustDoctor(t, repo) {
				if !check.OK {
					t.Fatalf("doctor: %s: %s", check.Name, check.Detail)
				}
			}
			if diff := runStatusGit(t, repo, "diff", "--cached"); diff != "" {
				t.Fatal("lifecycle staged files")
			}
		})
	}
}

func mustDoctor(t *testing.T, repo string) []Check {
	t.Helper()
	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	return checks
}

func TestFlatScaffoldLifecycleNeverConvertsLayout(t *testing.T) {
	for _, mode := range []Mode{ModeTeam, ModeLocal} {
		t.Run(string(mode), func(t *testing.T) {
			repo := mkRepo(t)
			if err := initFlatWithOptionsForTest(t, repo, InitOptions{Mode: mode, Addons: []string{"glossary"}}); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(repo, ".ctx")
			fact := filepath.Join(dest, "context/glossary.md")
			before := addTestRead(t, fact)
			if _, err := Update(repo, ".ctx"); err != nil {
				t.Fatal(err)
			}
			result, err := Add(repo, ".ctx", []string{"behavior"})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.Files, []string{"context/behavior.md"}) {
				t.Fatalf("flat add = %+v", result)
			}
			addTestEqualFile(t, fact, before)
			if got := addTestConfig(t, dest); got.LayoutVersion != 2 || got.TemplateRevision != "2.0.1" {
				t.Fatalf("flat descriptor changed: %+v", got)
			}
			if strings.Contains(string(addTestRead(t, filepath.Join(dest, "INDEX.md"))), "/index.md") {
				t.Fatal("v3 routes introduced into v2")
			}
			if mode == ModeTeam {
				if err := os.Remove(filepath.Join(dest, "local/CONTINUE.md")); err != nil {
					t.Fatal(err)
				}
				if err := InitWithOptions(repo, InitOptions{Mode: mode}); err != nil {
					t.Fatal(err)
				}
				addTestEqualFile(t, fact, before)
			}
			for _, check := range mustDoctor(t, repo) {
				if !check.OK {
					t.Fatalf("%s: %s", check.Name, check.Detail)
				}
			}
			if _, err := Status(repo, ".ctx"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFolderFirstAddonParentBoundaries(t *testing.T) {
	for _, kind := range []string{"symlink", "orphan", "existing topic"} {
		t.Run(kind, func(t *testing.T) {
			repo := mkRepo(t)
			if err := InitWithOptions(repo, InitOptions{Addons: []string{}}); err != nil {
				t.Fatal(err)
			}
			dest := filepath.Join(repo, ".ctx")
			parent := filepath.Join(dest, "context/glossary")
			outside := t.TempDir()
			if kind == "symlink" {
				if err := os.Symlink(outside, parent); err != nil {
					t.Skip(err)
				}
			} else {
				if err := os.Mkdir(parent, 0o755); err != nil {
					t.Fatal(err)
				}
				name := "hydration.md"
				if kind == "orphan" {
					name = "index.md"
				}
				writeStatusFile(t, filepath.Join(parent, name), "# Project-owned content\n")
			}
			beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
			beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))
			_, err := Add(repo, ".ctx", []string{"glossary"})
			if kind == "existing topic" {
				if err != nil {
					t.Fatal(err)
				}
				addTestEqualFile(t, filepath.Join(parent, "hydration.md"), []byte("# Project-owned content\n"))
			} else {
				if err == nil {
					t.Fatal("unsafe or undeclared output accepted")
				}
				addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
				addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatalf("outside directory changed: %v %v", entries, err)
			}
			addTestNoTemps(t, dest)
		})
	}
}
