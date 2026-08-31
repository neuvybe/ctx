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

	// Intentional user-fill placeholders are preserved (not init-substituted).
	if !strings.Contains(string(idx), "{{FOUNDER}}") {
		t.Errorf("INDEX.md should retain {{FOUNDER}} for the user to fill")
	}

	// version stamp
	v, err := os.ReadFile(filepath.Join(repo, ".ctx", ".ctx-version"))
	if err != nil {
		t.Fatalf("read .ctx-version: %v", err)
	}
	if strings.TrimSpace(string(v)) != Version {
		t.Errorf(".ctx-version = %q, want %q", strings.TrimSpace(string(v)), Version)
	}

	// Team mode is the default: durable files are visible and living state is local.
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatalf("load scaffold config: %v", err)
	}
	if state.Config.Mode != ModeTeam || state.Legacy {
		t.Errorf("default state = %+v, want current team mode", state)
	}
	if _, err := os.Stat(filepath.Join(repo, ".ctx", "local", "CONTINUE.md")); err != nil {
		t.Errorf("local/CONTINUE.md missing: %v", err)
	}
	if excluded, err := hasFolderExclusion(repo, ".ctx"); err != nil {
		t.Fatalf("check exclusion: %v", err)
	} else if excluded {
		t.Error("team-mode .ctx should not be excluded as a whole")
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
	idx, err := os.ReadFile(filepath.Join(repo, ".agent", "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(idx), "{{FOLDER}}") || !strings.Contains(string(idx), ".agent/") {
		t.Errorf("custom folder was not rendered in INDEX.md")
	}
	if excluded, err := hasFolderExclusion(repo, ".agent"); err != nil {
		t.Fatal(err)
	} else if excluded {
		t.Error("team-mode custom folder should not be excluded as a whole")
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
