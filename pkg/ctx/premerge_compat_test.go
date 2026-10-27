package ctx

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
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
			assertPreMergeFlatConfig(t, dest, fixture.mode, fixture.addons, "2.0.1")
			assertPreMergeScaffoldHealth(t, repo, fixture.folder, fixture.mode)
			owned := addPreMergeOwnerContent(t, dest)
			ownerRegions := preMergeOwnerRegions(t, dest)
			flatLayout, ok := LayoutForVersion(2)
			if !ok {
				t.Fatal("flat layout is not supported")
			}

			if _, err := Update(repo, fixture.folder); err != nil {
				t.Fatalf("update historical scaffold: %v", err)
			}
			assertPreMergeOwnerRegions(t, dest, ownerRegions)
			for path, data := range owned {
				addTestEqualFile(t, filepath.Join(dest, filepath.FromSlash(path)), data)
			}
			assertPreMergeFlatConfig(t, dest, fixture.mode, fixture.addons, flatLayout.TemplateRevision)

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
			assertPreMergeFlatConfig(t, dest, fixture.mode, addons, flatLayout.TemplateRevision)
			for path, data := range owned {
				addTestEqualFile(t, filepath.Join(dest, filepath.FromSlash(path)), data)
			}
			assertPreMergeOwnerRegions(t, dest, ownerRegions)
			if index := string(addTestRead(t, filepath.Join(dest, "INDEX.md"))); strings.Contains(index, "/index.md") {
				t.Fatalf("folder-first routes introduced into flat scaffold:\n%s", index)
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
	assertPreMergeFlatConfig(t, dest, ModeTeam, []string{"glossary"}, "2.0.1")
	assertPreMergeScaffoldHealth(t, clone, ".ctx", ModeTeam)
	if status := runStatusGit(t, clone, "status", "--porcelain", "--untracked-files=all"); status != "" {
		t.Fatalf("hydration dirtied the cloned repository: %s", status)
	}
}

func TestPreMergeFixturesWithNewerSupportedFlatRevision(t *testing.T) {
	// Keep this test serial: only this test's nested cases should observe the
	// temporary catalog revision, which is restored even if a nested case fails.
	original, ok := LayoutForVersion(2)
	if !ok {
		t.Fatal("flat layout is not supported")
	}
	parts := strings.Split(original.TemplateRevision, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid flat template revision: %q", original.TemplateRevision)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatal(err)
	}
	parts[2] = strconv.Itoa(patch + 1)
	advanced := original
	advanced.TemplateRevision = strings.Join(parts, ".")
	layouts[2] = advanced
	t.Cleanup(func() { layouts[2] = original })

	t.Run("update advances revision without converting saved layouts", TestPreMergeFlatScaffoldLifecycle)
	t.Run("hydration retains historical revision and durable bytes", TestPreMergeTeamScaffoldFreshCloneHydration)
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

func assertPreMergeFlatConfig(t *testing.T, dest string, mode Mode, addons []string, revision string) {
	t.Helper()
	cfg := addTestConfig(t, dest)
	// Identity and layout stay historically pinned. Inputs and hydration retain
	// the captured revision; update must advance to the supported flat revision.
	if cfg.SchemaVersion != 2 || cfg.LayoutVersion != 2 || cfg.TemplateRevision != revision || cfg.Project != "premerge-project" || cfg.Mode != mode || !slices.Equal(cfg.Addons, addons) {
		t.Fatalf("historical descriptor changed: %+v, want mode %s, add-ons %v, and revision %s", cfg, mode, addons, revision)
	}
}

func assertPreMergeScaffoldHealth(t *testing.T, repo, folder string, mode Mode) {
	t.Helper()
	dest := filepath.Join(repo, folder)
	before := preMergeScaffoldFiles(t, dest)
	cfg := addTestConfig(t, dest)
	flatLayout, ok := LayoutForVersion(2)
	if !ok {
		t.Fatal("flat layout is not supported")
	}
	comparison, err := compareTemplateRevisions(cfg.TemplateRevision, flatLayout.TemplateRevision)
	if err != nil || comparison > 0 {
		t.Fatalf("unsupported historical revision %q versus %q: %v", cfg.TemplateRevision, flatLayout.TemplateRevision, err)
	}
	checks, err := Doctor(repo, folder)
	if err != nil {
		t.Fatalf("doctor historical scaffold: %v", err)
	}
	revisionChecks := 0
	for _, check := range checks {
		if check.Name == "template revision" {
			revisionChecks++
			// An older supported fixture should request an update, not fail the
			// test before Update has a chance to advance its managed templates.
			if check.OK != (comparison == 0) || (comparison < 0 && !strings.Contains(check.Detail, "ctx update")) {
				t.Fatalf("unexpected template revision check: %+v", check)
			}
			continue
		}
		if !check.OK {
			t.Fatalf("doctor %s: %s", check.Name, check.Detail)
		}
	}
	if revisionChecks != 1 {
		t.Fatalf("template revision checks = %d, want exactly one", revisionChecks)
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

// Extract the two owner-controlled regions independently of the production
// managed-marker parser so a parser regression cannot mask a preservation bug.
func preMergeUnmanagedRegions(content []byte, marker string) ([2][]byte, error) {
	begin := []byte("<!-- ctx:managed begin " + marker + " -->")
	end := []byte("<!-- ctx:managed end " + marker + " -->")
	if bytes.Count(content, begin) != 1 || bytes.Count(content, end) != 1 {
		return [2][]byte{}, fmt.Errorf("expected one managed block named %q", marker)
	}
	prefix, rest, _ := bytes.Cut(content, begin)
	_, suffix, found := bytes.Cut(rest, end)
	if !found {
		return [2][]byte{}, fmt.Errorf("managed block %q ends before it begins", marker)
	}
	return [2][]byte{prefix, suffix}, nil
}

func preMergeOwnerRegions(t *testing.T, dest string) map[string][2][]byte {
	t.Helper()
	regions := map[string][2][]byte{}
	for path, marker := range map[string]string{"README.md": "readme-platform", "INDEX.md": "index-routing"} {
		var err error
		regions[path], err = preMergeUnmanagedRegions(addTestRead(t, filepath.Join(dest, path)), marker)
		if err != nil {
			t.Fatalf("snapshot project-owned %s content: %v", path, err)
		}
	}
	return regions
}

func assertPreMergeOwnerRegions(t *testing.T, dest string, want map[string][2][]byte) {
	t.Helper()
	for path, got := range preMergeOwnerRegions(t, dest) {
		if !reflect.DeepEqual(got, want[path]) {
			t.Fatalf("bytes outside the managed block changed in %s", path)
		}
	}
}

func TestPreMergeUnmanagedRegionComparison(t *testing.T) {
	begin := "<!-- ctx:managed begin readme-platform -->"
	end := "<!-- ctx:managed end readme-platform -->"
	original := "Owner prefix\n" + begin + "\nManaged body\n" + end + "\nOwner suffix\n"
	want, err := preMergeUnmanagedRegions([]byte(original), "readme-platform")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		content string
		equal   bool
		invalid bool
	}{
		{"unchanged owner regions", original, true, false},
		{"managed body can change", strings.ReplaceAll(original, "Managed body", "Updated managed body"), true, false},
		{"owner prefix changes are detected", strings.ReplaceAll(original, "Owner prefix", "Changed prefix"), false, false},
		{"owner suffix whitespace changes are detected", original + "\n", false, false},
		{"missing markers are rejected", strings.ReplaceAll(original, end, ""), false, true},
		{"duplicate markers are rejected", original + begin, false, true},
		{"reversed markers are rejected", end + "\n" + begin, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := preMergeUnmanagedRegions([]byte(test.content), "readme-platform")
			if (err != nil) != test.invalid {
				t.Fatalf("region extraction error = %v, want invalid=%t", err, test.invalid)
			}
			if !test.invalid && reflect.DeepEqual(got, want) != test.equal {
				t.Fatalf("owner region equality = %t, want %t", reflect.DeepEqual(got, want), test.equal)
			}
		})
	}
}

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
