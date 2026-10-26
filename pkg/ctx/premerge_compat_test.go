package ctx

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Unlike initFlatWithOptionsForTest, these inputs come from the pre-merge
// executable, not today's initializer, catalog, or template assets.
func TestPreMergeFlatScaffoldLifecycle(t *testing.T) {
	for _, fixture := range []struct {
		name   string
		folder string
		mode   Mode
		addons []string
	}{
		{"team-glossary-only", ".ctx", ModeTeam, []string{"glossary"}},
		{"local-core-only", "team_context-2.0", ModeLocal, nil},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			repo := loadPreMergeFlatFixture(t, fixture.name, fixture.folder)
			dest := filepath.Join(repo, fixture.folder)
			assertPreMergeFlatConfig(t, dest, fixture.mode, fixture.addons)
			assertPreMergeScaffoldHealth(t, repo, fixture.folder, fixture.mode)
			owned := addPreMergeOwnerContent(t, dest)
			beforeConfig := addTestRead(t, filepath.Join(dest, "config.json"))

			if _, err := Update(repo, fixture.folder); err != nil {
				t.Fatalf("update historical scaffold: %v", err)
			}
			addTestEqualFile(t, filepath.Join(dest, "config.json"), beforeConfig)
			for path, data := range owned {
				addTestEqualFile(t, filepath.Join(dest, filepath.FromSlash(path)), data)
			}
			assertPreMergeFlatConfig(t, dest, fixture.mode, fixture.addons)

			// Defaults must not be silently added to an existing saved selection.
			for _, addon := range []string{"behavior", "glossary"} {
				if !slices.Contains(fixture.addons, addon) {
					path := filepath.Join(dest, "context", addon+".md")
					if _, err := os.Lstat(path); !os.IsNotExist(err) {
						t.Fatalf("update introduced unselected add-on %s: %v", addon, err)
					}
				}
			}

			result, err := Add(repo, fixture.folder, []string{"behavior", "contracts", "review"})
			if err != nil {
				t.Fatalf("add to historical scaffold: %v", err)
			}
			if want := []string{"context/behavior.md", "context/contracts.md", "workflows/review.md"}; !slices.Equal(result.Files, want) {
				t.Fatalf("add-on output paths = %v, want %v", result.Files, want)
			}
			addons := []string{"behavior", "contracts"}
			addons = append(addons, fixture.addons...)
			addons = append(addons, "review")
			assertPreMergeFlatConfig(t, dest, fixture.mode, addons)
			for path, data := range owned {
				addTestEqualFile(t, filepath.Join(dest, filepath.FromSlash(path)), data)
			}
			for _, path := range []string{"README.md", "INDEX.md"} {
				if !bytes.Contains(addTestRead(t, filepath.Join(dest, path)), []byte(preMergeOwnerNote)) {
					t.Fatalf("lifecycle changed project-owned %s content", path)
				}
			}
			if index := string(addTestRead(t, filepath.Join(dest, "INDEX.md"))); strings.Contains(index, "/index.md") || !strings.Contains(index, preMergeOwnerRoute) {
				t.Fatalf("flat or project-owned routes changed:\n%s", index)
			}
			for _, category := range []string{"overview", "architecture", "caveats", "behavior", "contracts", "extending", "glossary"} {
				if _, err := os.Lstat(filepath.Join(dest, "context", category, "index.md")); !os.IsNotExist(err) {
					t.Fatalf("lifecycle introduced folder-first %s output: %v", category, err)
				}
			}
			assertPreMergeScaffoldHealth(t, repo, fixture.folder, fixture.mode)
			report, err := Status(repo, fixture.folder)
			if err != nil {
				t.Fatal(err)
			}
			if findStatusCheck(report, preMergeOwnerTopic, ContentNotReady) == nil {
				t.Fatalf("status missed authored nested topic: %+v", report)
			}
		})
	}
}

func TestPreMergeTeamScaffoldFreshCloneHydration(t *testing.T) {
	repo := loadPreMergeFlatFixture(t, "team-glossary-only", ".ctx")
	addPreMergeOwnerContent(t, filepath.Join(repo, ".ctx"))
	runStatusGit(t, repo, "add", ".ctx")
	runStatusGit(t, repo, "-c", "user.name=ctx-test", "-c", "user.email=ctx@example.invalid", "commit", "-qm", "share historical scaffold")
	clone := filepath.Join(t.TempDir(), "renamed-clone")
	runStatusGit(t, repo, "clone", "--quiet", "--no-hardlinks", repo, clone)
	dest := filepath.Join(clone, ".ctx")
	continuation := filepath.Join(dest, "local", "CONTINUE.md")
	if _, err := os.Lstat(continuation); !os.IsNotExist(err) {
		t.Fatalf("clone inherited ignored continuation: %v", err)
	}
	before := preMergeScaffoldFiles(t, dest)

	// Implicit current defaults must honor the old clone's persisted layout,
	// project name, and glossary-only selection rather than create a v3 scaffold.
	if err := InitWithOptions(clone, InitOptions{Folder: ".ctx"}); err != nil {
		t.Fatalf("hydrate historical clone: %v", err)
	}
	after := preMergeScaffoldFiles(t, dest)
	if !bytes.Contains(after["local/CONTINUE.md"], []byte("local working state for premerge-project")) {
		t.Fatal("hydration did not create continuation for the persisted project")
	}
	delete(after, "local/CONTINUE.md")
	if !reflect.DeepEqual(after, before) {
		t.Fatal("hydration changed durable files or created unexpected outputs")
	}
	assertPreMergeFlatConfig(t, dest, ModeTeam, []string{"glossary"})
	assertPreMergeScaffoldHealth(t, clone, ".ctx", ModeTeam)
	if status := runStatusGit(t, clone, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("hydration dirtied the cloned repository: %s", status)
	}
}

func loadPreMergeFlatFixture(t *testing.T, name, folder string) string {
	t.Helper()
	repo := mkRepo(t)
	root := filepath.Join("testdata", "premerge-flat-v2", name)
	err := fs.WalkDir(os.DirFS(filepath.Join(root, "scaffold")), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !entry.Type().IsRegular() || !strings.HasSuffix(path, ".fixture") {
			t.Fatalf("unexpected historical fixture entry: %s", path)
		}
		data := addTestRead(t, filepath.Join(root, "scaffold", filepath.FromSlash(path)))
		output := filepath.Join(repo, folder, filepath.FromSlash(strings.TrimSuffix(path, ".fixture")))
		writeStatusFile(t, output, string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("load historical scaffold: %v", err)
	}
	writeStatusFile(t, filepath.Join(repo, ".git", "info", "exclude"), string(addTestRead(t, filepath.Join(root, "git-info-exclude.fixture"))))
	return repo
}

func assertPreMergeFlatConfig(t *testing.T, dest string, mode Mode, addons []string) {
	t.Helper()
	cfg := addTestConfig(t, dest)
	// Keep historical expectations literal, independent of current defaults.
	if cfg.SchemaVersion != 2 || cfg.LayoutVersion != 2 || cfg.TemplateRevision != "2.0.1" || cfg.Project != "premerge-project" || cfg.Mode != mode || !slices.Equal(cfg.Addons, addons) {
		t.Fatalf("historical descriptor changed: %+v, want mode %s and add-ons %v", cfg, mode, addons)
	}
}

func assertPreMergeScaffoldHealth(t *testing.T, repo, folder string, mode Mode) {
	t.Helper()
	dest := filepath.Join(repo, folder)
	before := preMergeScaffoldFiles(t, dest)
	checks, err := Doctor(repo, folder)
	if err != nil {
		t.Fatalf("doctor historical scaffold: %v", err)
	}
	for _, check := range checks {
		if !check.OK {
			t.Fatalf("doctor %s: %s", check.Name, check.Detail)
		}
	}
	report, err := Status(repo, folder)
	if err != nil || report.LayoutVersion != 2 || report.Ready() {
		t.Fatalf("historical draft status = %+v, error = %v", report, err)
	}
	if !reflect.DeepEqual(preMergeScaffoldFiles(t, dest), before) {
		t.Fatal("doctor or status modified historical scaffold")
	}
	for path := range before {
		wantIgnored := mode == ModeLocal || strings.HasPrefix(path, "local/")
		if ignored := modeTestGitIgnored(t, repo, filepath.Join(folder, filepath.FromSlash(path))); ignored != wantIgnored {
			t.Fatalf("visibility changed for %s: ignored=%t, want %t", path, ignored, wantIgnored)
		}
	}
	if staged := runStatusGit(t, repo, "diff", "--cached"); staged != "" {
		t.Fatalf("lifecycle staged files: %s", staged)
	}
	addTestNoTemps(t, dest)
}

const (
	preMergeOwnerNote  = "Owner-authored content must survive lifecycle commands."
	preMergeOwnerTopic = "context/architecture/owner-topic.md"
	preMergeOwnerRoute = "| Owner topic | `context/architecture/owner-topic.md` |"
)

func addPreMergeOwnerContent(t *testing.T, dest string) map[string][]byte {
	t.Helper()
	owned := map[string][]byte{}
	for path, data := range preMergeScaffoldFiles(t, dest) {
		if strings.HasPrefix(path, "context/") || path == "local/CONTINUE.md" {
			owned[path] = append(data, []byte("\n"+preMergeOwnerNote+"\n")...)
			writeStatusFile(t, filepath.Join(dest, filepath.FromSlash(path)), string(owned[path]))
		}
	}
	owned[preMergeOwnerTopic] = []byte("<!-- ctx:doc {\"status\":\"draft\",\"verifiedAt\":\"\",\"sources\":[]} -->\n\n# Owner topic\n\n" + preMergeOwnerNote + "\n")
	writeStatusFile(t, filepath.Join(dest, filepath.FromSlash(preMergeOwnerTopic)), string(owned[preMergeOwnerTopic]))
	for _, path := range []string{"README.md", "INDEX.md"} {
		data := addTestRead(t, filepath.Join(dest, path))
		writeStatusFile(t, filepath.Join(dest, path), string(data)+"\n"+preMergeOwnerNote+"\n")
	}
	index := filepath.Join(dest, "INDEX.md")
	writeStatusFile(t, index, string(addTestRead(t, index))+"\n"+preMergeOwnerRoute+"\n")
	return owned
}

func preMergeScaffoldFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !entry.Type().IsRegular() {
			t.Fatalf("unexpected non-file in scaffold: %s", path)
		}
		files[path] = addTestRead(t, filepath.Join(root, filepath.FromSlash(path)))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot historical scaffold: %v", err)
	}
	return files
}
