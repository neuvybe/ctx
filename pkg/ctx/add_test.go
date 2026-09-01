package ctx

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

func TestAddGlossary(t *testing.T) {
	repo := addTestV2Repo(t, ModeTeam)
	dest := filepath.Join(repo, ".ctx")
	indexPath := filepath.Join(dest, "INDEX.md")
	indexBefore := addTestRead(t, indexPath)
	projectRoute := []byte("\nProject route sentinel: `context/domain/deep.md`\n")
	if err := os.WriteFile(indexPath, append(indexBefore, projectRoute...), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Add(repo, ".ctx", []string{"glossary"})
	if err != nil {
		t.Fatalf("Add glossary: %v", err)
	}
	if !reflect.DeepEqual(result.Addons, []string{"glossary"}) || !reflect.DeepEqual(result.Files, []string{"context/glossary.md"}) {
		t.Fatalf("Add result = %+v", result)
	}
	glossary := addTestRead(t, filepath.Join(dest, "context", "glossary.md"))
	if !bytes.Contains(glossary, []byte(filepath.Base(repo))) {
		t.Fatalf("glossary does not use persisted project name: %s", glossary)
	}
	cfg := addTestConfig(t, dest)
	if !reflect.DeepEqual(cfg.Addons, []string{"glossary"}) {
		t.Fatalf("config addons = %v, want [glossary]", cfg.Addons)
	}
	index := string(addTestRead(t, indexPath))
	if !strings.Contains(index, "`context/glossary.md`") {
		t.Fatalf("INDEX routing was not refreshed:\n%s", index)
	}
	if !strings.Contains(index, string(projectRoute)) {
		t.Fatalf("project-owned INDEX routing was not preserved:\n%s", index)
	}
	if modeTestGitIgnored(t, repo, ".ctx/context/glossary.md") {
		t.Fatal("team-mode add-on output is ignored")
	}
	addTestNoTemps(t, dest)
}

func TestAddMultipleDeterministicRoutesAndConfig(t *testing.T) {
	repo := addTestV2Repo(t, ModeTeam)
	dest := filepath.Join(repo, ".ctx")
	extra := filepath.Join(dest, "context", "deep", "owner-notes.md")
	if err := os.MkdirAll(filepath.Dir(extra), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(extra, []byte("owner-authored\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Add(repo, ".ctx", []string{"glossary,contracts", "extending", "contracts"})
	if err != nil {
		t.Fatalf("Add multiple: %v", err)
	}
	wantAddons := []string{"contracts", "extending", "glossary"}
	wantFiles := []string{"context/contracts.md", "context/extending.md", "context/glossary.md"}
	if !reflect.DeepEqual(result.Addons, wantAddons) || !reflect.DeepEqual(result.Files, wantFiles) {
		t.Fatalf("Add result = %+v, want addons %v files %v", result, wantAddons, wantFiles)
	}
	cfg := addTestConfig(t, dest)
	if !reflect.DeepEqual(cfg.Addons, wantAddons) {
		t.Fatalf("config addons = %v, want %v", cfg.Addons, wantAddons)
	}
	index := string(addTestRead(t, filepath.Join(dest, "INDEX.md")))
	positions := []int{
		strings.Index(index, "`context/contracts.md`"),
		strings.Index(index, "`context/extending.md`"),
		strings.Index(index, "`context/glossary.md`"),
	}
	if positions[0] < 0 || positions[1] <= positions[0] || positions[2] <= positions[1] {
		t.Fatalf("optional routes are absent or nondeterministic: positions %v\n%s", positions, index)
	}
	if got := string(addTestRead(t, extra)); got != "owner-authored\n" {
		t.Fatalf("arbitrary nested context doc changed: %q", got)
	}
	addTestNoTemps(t, dest)
}

func TestAddRefusesExistingOutputsWithoutMutation(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, path string)
	}{
		{
			name: "regular",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("owner content\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "directory",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlink",
			setup: func(t *testing.T, path string) {
				t.Helper()
				outside := filepath.Join(t.TempDir(), "outside.md")
				if err := os.WriteFile(outside, []byte("outside\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, path); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := addTestV2Repo(t, ModeTeam)
			dest := filepath.Join(repo, ".ctx")
			glossary := filepath.Join(dest, "context", "glossary.md")
			tt.setup(t, glossary)
			beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
			beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

			if _, err := Add(repo, ".ctx", []string{"contracts,glossary"}); err == nil {
				t.Fatal("Add accepted an existing output")
			}
			addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
			addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
			if _, err := os.Lstat(filepath.Join(dest, "context", "contracts.md")); !os.IsNotExist(err) {
				t.Fatalf("partial contracts output exists: %v", err)
			}
			addTestNoTemps(t, dest)
		})
	}
}

func TestAddRefusesTrackedButMissingOutput(t *testing.T) {
	repo := addTestV2Repo(t, ModeTeam)
	dest := filepath.Join(repo, ".ctx")
	glossary := filepath.Join(dest, "context", "glossary.md")
	if err := os.WriteFile(glossary, []byte("tracked owner content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	modeTestRunGit(t, repo, "add", ".ctx/context/glossary.md")
	if err := os.Remove(glossary); err != nil {
		t.Fatal(err)
	}
	beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
	beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

	if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "tracked index") {
		t.Fatalf("Add error = %v, want tracked-index rejection", err)
	}
	addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
	addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
	if _, err := os.Lstat(glossary); !os.IsNotExist(err) {
		t.Fatalf("tracked-but-missing output was recreated: %v", err)
	}
}

func TestAddRejectsV1WithoutMigration(t *testing.T) {
	repo := addTestV2Repo(t, ModeTeam)
	dest := filepath.Join(repo, ".ctx")
	v1, err := marshalConfig(Config{SchemaVersion: 1, Mode: ModeTeam})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, configFileName), v1, 0o644); err != nil {
		t.Fatal(err)
	}
	beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

	if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "no automatic migration") {
		t.Fatalf("Add v1 error = %v", err)
	}
	addTestEqualFile(t, filepath.Join(dest, configFileName), v1)
	addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
	if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
		t.Fatalf("v1 add created output: %v", err)
	}
}

func TestAddRequiresCurrentTemplateRevision(t *testing.T) {
	for _, revision := range []string{"1.0.0", "99.0.0"} {
		t.Run(revision, func(t *testing.T) {
			repo := addTestV2Repo(t, ModeTeam)
			dest := filepath.Join(repo, ".ctx")
			cfg := addTestConfig(t, dest)
			cfg.TemplateRevision = revision
			if err := writeScaffoldConfig(dest, cfg); err != nil {
				t.Fatal(err)
			}
			beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
			beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

			if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil {
				t.Fatal("Add accepted a non-current template revision")
			}
			addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
			addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
			if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
				t.Fatalf("rejected Add created output: %v", err)
			}
		})
	}
}

func TestAddRejectsMissingPreviouslyConfiguredOutput(t *testing.T) {
	repo := addTestV2Repo(t, ModeTeam)
	if _, err := Add(repo, ".ctx", []string{"contracts"}); err != nil {
		t.Fatalf("Add contracts: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	contractsPath := filepath.Join(dest, "context", "contracts.md")
	if err := os.Remove(contractsPath); err != nil {
		t.Fatal(err)
	}
	beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
	beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

	if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "context/contracts.md") {
		t.Fatalf("Add with missing configured output error = %v", err)
	}
	addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
	addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
	if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
		t.Fatalf("failed Add created glossary: %v", err)
	}
}

func TestAddRejectsIgnoredPreviouslyConfiguredOutput(t *testing.T) {
	repo := addTestV2Repo(t, ModeTeam)
	if _, err := Add(repo, ".ctx", []string{"contracts"}); err != nil {
		t.Fatalf("Add contracts: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/context/contracts.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
	beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

	if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "contracts.md") {
		t.Fatalf("Add with ignored configured output error = %v", err)
	}
	addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
	addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
	if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
		t.Fatalf("failed Add created glossary: %v", err)
	}
}

func TestAddRejectsIgnoredProjectOwnedSharedDocumentWithoutMutation(t *testing.T) {
	repo := addTestV2Repo(t, ModeTeam)
	dest := filepath.Join(repo, ".ctx")
	customPath := filepath.Join(dest, "context", "custom.md")
	customContent := []byte("owner-authored shared context\n")
	if err := os.WriteFile(customPath, customContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/context/custom.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
	beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

	if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "context/custom.md") {
		t.Fatalf("Add with ignored project-owned document error = %v", err)
	}
	addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
	addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
	addTestEqualFile(t, customPath, customContent)
	if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
		t.Fatalf("failed Add created glossary: %v", err)
	}
	addTestNoTemps(t, dest)
}

func TestAddLocalModeRequiresContinuation(t *testing.T) {
	repo := addTestV2Repo(t, ModeLocal)
	dest := filepath.Join(repo, ".ctx")
	if err := os.Remove(filepath.Join(dest, "local", "CONTINUE.md")); err != nil {
		t.Fatal(err)
	}

	if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "local/CONTINUE.md") {
		t.Fatalf("Add with missing local continuation error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
		t.Fatalf("failed Add created glossary: %v", err)
	}
}

func TestAddLocalModeRejectsVisibleDirectoryAndFutureFiles(t *testing.T) {
	repo := addTestV2Repo(t, ModeLocal)
	dest := filepath.Join(repo, ".ctx")
	rules := "!/.ctx/\n/.ctx/*\n!/.ctx/future-secret.txt\n"
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(rules), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
	beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

	if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "visible to Git") {
		t.Fatalf("Add with visible local directory error = %v", err)
	}
	addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
	addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
	if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
		t.Fatalf("failed Add created glossary: %v", err)
	}
}

func TestAddTeamModeRequiresPrivateContinuationBoundary(t *testing.T) {
	t.Run("missing continuation", func(t *testing.T) {
		repo := addTestV2Repo(t, ModeTeam)
		dest := filepath.Join(repo, ".ctx")
		if err := os.Remove(filepath.Join(dest, "local", "CONTINUE.md")); err != nil {
			t.Fatal(err)
		}

		if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "local/CONTINUE.md") {
			t.Fatalf("Add with missing team continuation error = %v", err)
		}
	})

	t.Run("visible continuation", func(t *testing.T) {
		repo := addTestV2Repo(t, ModeTeam)
		dest := filepath.Join(repo, ".ctx")
		if err := os.WriteFile(filepath.Join(dest, ".gitignore"), []byte("**/.ctx-update-*\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil || !strings.Contains(err.Error(), "local") {
			t.Fatalf("Add with visible team continuation error = %v", err)
		}
		if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
			t.Fatalf("failed Add created glossary: %v", err)
		}
	})
}

func TestAddMalformedIndexAndIgnoredOutputDoNotMutate(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, repo, dest string)
	}{
		{
			name: "malformed index",
			setup: func(t *testing.T, repo, dest string) {
				t.Helper()
				path := filepath.Join(dest, "INDEX.md")
				content := string(addTestRead(t, path))
				content = strings.Replace(content, "<!-- ctx:managed end index-routing -->", "<!-- ctx:managed end wrong-id -->", 1)
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "extra managed index block",
			setup: func(t *testing.T, repo, dest string) {
				t.Helper()
				path := filepath.Join(dest, "INDEX.md")
				content := string(addTestRead(t, path)) + "\n<!-- ctx:managed begin unexpected -->\nextra\n<!-- ctx:managed end unexpected -->\n"
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "ignored team output",
			setup: func(t *testing.T, repo, dest string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/context/glossary.md\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := addTestV2Repo(t, ModeTeam)
			dest := filepath.Join(repo, ".ctx")
			tt.setup(t, repo, dest)
			beforeConfig := addTestRead(t, filepath.Join(dest, configFileName))
			beforeIndex := addTestRead(t, filepath.Join(dest, "INDEX.md"))

			if _, err := Add(repo, ".ctx", []string{"glossary"}); err == nil {
				t.Fatal("Add unexpectedly succeeded")
			}
			addTestEqualFile(t, filepath.Join(dest, configFileName), beforeConfig)
			addTestEqualFile(t, filepath.Join(dest, "INDEX.md"), beforeIndex)
			if _, err := os.Lstat(filepath.Join(dest, "context", "glossary.md")); !os.IsNotExist(err) {
				t.Fatalf("failed Add created output: %v", err)
			}
			addTestNoTemps(t, dest)
		})
	}
}

func TestConcurrentAddDoesNotLoseConfigOrRoutes(t *testing.T) {
	repo := addTestV2Repo(t, ModeTeam)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, addon := range []string{"contracts", "glossary"} {
		addon := addon
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Add(repo, ".ctx", []string{addon})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Add: %v", err)
		}
	}
	cfg := addTestConfig(t, filepath.Join(repo, ".ctx"))
	if !reflect.DeepEqual(cfg.Addons, []string{"contracts", "glossary"}) {
		t.Fatalf("config lost concurrent add-on: %v", cfg.Addons)
	}
	index := string(addTestRead(t, filepath.Join(repo, ".ctx", "INDEX.md")))
	for _, path := range []string{"context/contracts.md", "context/glossary.md"} {
		if !strings.Contains(index, "`"+path+"`") {
			t.Fatalf("INDEX lost route %s:\n%s", path, index)
		}
	}
}

func TestAddLocalModeKeepsNewOutputPrivate(t *testing.T) {
	repo := addTestV2Repo(t, ModeLocal)
	if _, err := Add(repo, ".ctx", []string{"glossary"}); err != nil {
		t.Fatalf("Add local glossary: %v", err)
	}
	if !modeTestGitIgnored(t, repo, ".ctx/context/glossary.md") {
		t.Fatal("local-mode add-on output is visible to Git")
	}
}

func addTestV2Repo(t *testing.T, mode Mode) string {
	t.Helper()
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: mode, Addons: []string{}}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	return repo
}

func addTestRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func addTestConfig(t *testing.T, dest string) Config {
	t.Helper()
	cfg, err := parseConfig(addTestRead(t, filepath.Join(dest, configFileName)))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return cfg
}

func addTestEqualFile(t *testing.T, path string, want []byte) {
	t.Helper()
	if got := addTestRead(t, path); !bytes.Equal(got, want) {
		t.Fatalf("%s changed\ngot:  %q\nwant: %q", path, got, want)
	}
}

func addTestNoTemps(t *testing.T, dest string) {
	t.Helper()
	var leftovers []string
	err := filepath.WalkDir(dest, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), ".ctx-update-") {
			leftovers = append(leftovers, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk temp files: %v", err)
	}
	sort.Strings(leftovers)
	if len(leftovers) > 0 {
		t.Fatalf("leftover add temp files: %v", leftovers)
	}
}
