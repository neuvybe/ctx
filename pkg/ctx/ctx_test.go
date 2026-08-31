package ctx

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// mkRepo makes a real Git repository so tests exercise Git's effective ignore
// semantics rather than only the presence of a .git directory.
func mkRepo(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = d
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
	return d
}

func TestSubstitute(t *testing.T) {
	cases := []struct{ in, project, date, want string }{
		{"# {{PROJECT}}", "proj", "2026-08-10", "# proj"},
		{"{{PROJECT}} {{DATE}}", "p", "2026-01-01", "p 2026-01-01"},
		{"{{FOUNDER}} keeps {{PROJECT}}", "proj", "x", "{{FOUNDER}} keeps proj"},
		{"no placeholders here", "p", "d", "no placeholders here"},
		{"", "p", "d", ""},
		{"{{DATE}} {{DATE}}", "p", "2026-09-01", "2026-09-01 2026-09-01"},
	}
	for _, c := range cases {
		got := substitute(c.in, c.project, c.date)
		if got != c.want {
			t.Errorf("substitute(%q,%q,%q) = %q, want %q", c.in, c.project, c.date, got, c.want)
		}
	}
}

func TestTemplateFS(t *testing.T) {
	fsys := TemplateFS()
	want := []string{
		"README.md", "OPERATING.md", "INDEX.md", "REVIEW.md", "local/CONTINUE.md",
		"context/overview.md", "context/architecture.md", "context/format.md",
		"context/extending.md", "context/known-issues.md", "context/glossary.md",
	}
	var got []string
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if path == "." {
			return nil
		}
		got = append(got, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk template: %v", err)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("template files = %v, want %v", got, want)
	}
}

func TestVersionedAndAddonTemplateFilesystems(t *testing.T) {
	v2, err := TemplateFSForLayout(CurrentLayoutVersion)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(v2, "context/caveats.md"); err != nil {
		t.Fatalf("v2 template filesystem: %v", err)
	}
	glossary, err := AddonTemplateFS("glossary")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(glossary, "context/glossary.md"); err != nil {
		t.Fatalf("glossary template filesystem: %v", err)
	}
	if _, err := AddonTemplateFS("unknown"); err == nil {
		t.Fatal("AddonTemplateFS accepted an unknown add-on")
	}
}

func TestInitWithOptionsTeam(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}

	// A sample file exists and {{PROJECT}} was substituted to the repo basename.
	idx, err := os.ReadFile(filepath.Join(repo, ".ctx", "INDEX.md"))
	if err != nil {
		t.Fatalf("read INDEX.md: %v", err)
	}
	if !strings.Contains(string(idx), filepath.Base(repo)) {
		t.Errorf("INDEX.md missing project name %q", filepath.Base(repo))
	}
	if strings.Contains(string(idx), "{{PROJECT}}") {
		t.Errorf("INDEX.md still has {{PROJECT}}")
	}

	// context/*.md {{PROJECT}} substituted too.
	ov, err := os.ReadFile(filepath.Join(repo, ".ctx", "context", "overview.md"))
	if err != nil {
		t.Fatalf("read overview.md: %v", err)
	}
	if strings.Contains(string(ov), "{{PROJECT}}") {
		t.Errorf("overview.md still has {{PROJECT}}")
	}

	if strings.Contains(string(idx), "{{ADDON_ROUTES}}") {
		t.Errorf("INDEX.md still has {{ADDON_ROUTES}}")
	}
	if !strings.Contains(string(idx), "context/glossary.md") {
		t.Errorf("INDEX.md missing default glossary route")
	}
	if _, err := os.Lstat(filepath.Join(repo, ".ctx", ".ctx-version")); !os.IsNotExist(err) {
		t.Fatalf("layout v2 should not create .ctx-version: %v", err)
	}

	// Team mode is the default: durable files are visible and living state is local.
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatalf("load scaffold config: %v", err)
	}
	if state.Config.Mode != ModeTeam || state.Legacy {
		t.Errorf("default state = %+v, want current team mode", state)
	}
	if state.Config.SchemaVersion != currentSchemaVersion || state.layoutVersion() != CurrentLayoutVersion || state.Config.Project != filepath.Base(repo) {
		t.Errorf("default config = %+v, want schema/layout v2 with stable project", state.Config)
	}
	if !reflect.DeepEqual(state.Config.Addons, []string{"glossary"}) {
		t.Errorf("default add-ons = %v, want [glossary]", state.Config.Addons)
	}
	if _, err := os.Stat(filepath.Join(repo, ".ctx", "context", "glossary.md")); err != nil {
		t.Errorf("default glossary missing: %v", err)
	}
	for _, optional := range []string{"OPERATING.md", "REVIEW.md"} {
		if _, err := os.Lstat(filepath.Join(repo, ".ctx", filepath.FromSlash(optional))); !os.IsNotExist(err) {
			t.Errorf("non-default add-on unexpectedly created %s: %v", optional, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".ctx", "local", "CONTINUE.md")); err != nil {
		t.Errorf("local/CONTINUE.md missing: %v", err)
	}
	if excluded, err := hasFolderExclusion(repo, ".ctx"); err != nil {
		t.Fatalf("check exclusion: %v", err)
	} else if excluded {
		t.Error("team-mode .ctx should not be excluded as a whole")
	}
	if ignored, err := gitCheckIgnored(repo, filepath.Join(".ctx", "context", ".ctx-update-probe")); err != nil || !ignored {
		t.Fatalf("lifecycle transaction probe ignored = %v, err = %v", ignored, err)
	}
}

func TestInitWithOptionsExplicitEmptyAddonsCreatesCoreOnly(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam, Addons: []string{}}); err != nil {
		t.Fatal(err)
	}
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Config.Addons) != 0 {
		t.Fatalf("core-only config add-ons = %v, want none", state.Config.Addons)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".ctx", "context", "glossary.md")); !os.IsNotExist(err) {
		t.Fatalf("core-only init created glossary: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(repo, ".ctx", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "No optional add-ons installed") {
		t.Fatalf("core-only INDEX missing empty routing guidance:\n%s", index)
	}
}

func TestInitWithOptionsImplicitAddonsHonorPersistedConfigDuringHydration(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam, Addons: []string{}}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repo, ".ctx", "local")); err != nil {
		t.Fatal(err)
	}
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("hydrate persisted core-only scaffold with implicit defaults: %v", err)
	}
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Config.Addons) != 0 {
		t.Fatalf("hydration changed persisted add-ons to %v", state.Config.Addons)
	}
	if _, err := os.Stat(filepath.Join(repo, ".ctx", "local", "CONTINUE.md")); err != nil {
		t.Fatalf("hydration did not restore continuation: %v", err)
	}
}

func TestInitRefusesOverwrite(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err == nil {
		t.Errorf("second Init should have errored, got nil")
	}
}

func TestInitCustomFolder(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".agent", Mode: ModeTeam}); err != nil {
		t.Fatalf("Init .agent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".agent", "INDEX.md")); err != nil {
		t.Errorf(".agent/INDEX.md missing: %v", err)
	}
	readme, err := os.ReadFile(filepath.Join(repo, ".agent", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(readme), "{{FOLDER}}") || !strings.Contains(string(readme), ".agent/") {
		t.Errorf("custom folder was not rendered in README.md")
	}
	if excluded, err := hasFolderExclusion(repo, ".agent"); err != nil {
		t.Fatal(err)
	} else if excluded {
		t.Error("team-mode custom folder should not be excluded as a whole")
	}
}

func TestInitWithAddonsRendersOnlySelectedRoutes(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{
		Folder: ".ctx",
		Mode:   ModeTeam,
		Addons: []string{"glossary,contracts", "glossary"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"context/contracts.md", "context/glossary.md"} {
		if _, err := os.Stat(filepath.Join(repo, ".ctx", filepath.FromSlash(name))); err != nil {
			t.Fatalf("selected add-on %s missing: %v", name, err)
		}
	}
	if _, err := os.Lstat(filepath.Join(repo, ".ctx", "OPERATING.md")); !os.IsNotExist(err) {
		t.Fatalf("unselected add-on was created: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(repo, ".ctx", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"context/contracts.md", "context/glossary.md"} {
		if !strings.Contains(string(index), name) {
			t.Fatalf("INDEX missing selected route %s:\n%s", name, index)
		}
	}
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state.Config.Addons, []string{"contracts", "glossary"}) {
		t.Fatalf("persisted add-ons = %v", state.Config.Addons)
	}
}

func TestInitNotAGitRepo(t *testing.T) {
	d := t.TempDir() // no .git
	if err := InitWithOptions(d, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err == nil {
		t.Errorf("Init on non-git dir should error")
	}
}

func TestEnsureExcludedIdempotent(t *testing.T) {
	repo := mkRepo(t)
	if err := ensureExcluded(repo, ".ctx"); err != nil {
		t.Fatalf("first ensureExcluded: %v", err)
	}
	if err := ensureExcluded(repo, ".ctx"); err != nil {
		t.Fatalf("second ensureExcluded: %v", err)
	}
	exc, _ := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	// Exactly one root-anchored "/.ctx/" entry line.
	count := strings.Count(string(exc), "\n/.ctx/\n")
	if count != 1 {
		t.Errorf("expected exactly 1 .ctx/ exclude entry, got %d in:\n%s", count, exc)
	}
}

func TestResolveGitDirNormal(t *testing.T) {
	repo := mkRepo(t)
	got, err := resolveGitDir(repo)
	if err != nil {
		t.Fatalf("resolveGitDir: %v", err)
	}
	if got != filepath.Join(repo, ".git") {
		t.Errorf("resolveGitDir = %q, want %q", got, filepath.Join(repo, ".git"))
	}
}

func TestResolveGitDirGitfileAbsolute(t *testing.T) {
	repo := t.TempDir()
	realGit := filepath.Join(repo, "real-git")
	if err := os.Mkdir(realGit, 0o755); err != nil {
		t.Fatal(err)
	}
	// .git is a gitfile pointing at an absolute realGit.
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: "+realGit+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveGitDir(repo)
	if err != nil {
		t.Fatalf("resolveGitDir: %v", err)
	}
	if got != realGit {
		t.Errorf("resolveGitDir = %q, want %q", got, realGit)
	}
}

func TestResolveGitDirGitfileRelative(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "sub", "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// .git gitfile with a relative path -> resolved against the repo.
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: sub/git"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := resolveGitDir(repo)
	if err != nil {
		t.Fatalf("resolveGitDir: %v", err)
	}
	want := filepath.Join(repo, "sub", "git")
	if got != want {
		t.Errorf("resolveGitDir = %q, want %q", got, want)
	}
}
