package ctx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommandDefaultsToTeamMode(t *testing.T) {
	repo := mkRepo(t)
	var output bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"init", repo})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("ctx init: %v", err)
	}
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatal(err)
	}
	if state.Config.Mode != ModeTeam {
		t.Fatalf("command mode = %s, want team", state.Config.Mode)
	}
	if strings.Join(state.Config.Addons, ",") != "glossary" {
		t.Fatalf("default command add-ons = %v, want [glossary]", state.Config.Addons)
	}
	if !strings.Contains(output.String(), "mode team") || !strings.Contains(output.String(), "visible to Git") {
		t.Fatalf("team-mode output missing visibility summary:\n%s", output.String())
	}
}

func TestInitCommandAcceptsLocalMode(t *testing.T) {
	repo := mkRepo(t)
	var output bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"init", repo, "--mode", "local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("ctx init --mode local: %v", err)
	}
	if ignored, err := gitCheckIgnored(repo, filepath.Join(".ctx", "README.md")); err != nil || !ignored {
		t.Fatalf("local scaffold ignored = %v, err = %v", ignored, err)
	}
	if !strings.Contains(output.String(), "mode local") || !strings.Contains(output.String(), "entire .ctx/ folder is ignored") {
		t.Fatalf("local-mode output missing visibility summary:\n%s", output.String())
	}
}

func TestInitCommandAcceptsAddons(t *testing.T) {
	repo := mkRepo(t)
	var output bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"init", repo, "--with", "contracts"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("ctx init --with: %v", err)
	}
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(state.Config.Addons, ",") != "contracts,glossary" {
		t.Fatalf("installed add-ons = %v", state.Config.Addons)
	}
}

func TestInitCommandCanOptOutOfDefaultGlossary(t *testing.T) {
	repo := mkRepo(t)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init", repo, "--without", "glossary"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ctx init --without glossary: %v", err)
	}
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Config.Addons) != 0 {
		t.Fatalf("installed add-ons = %v, want none", state.Config.Addons)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".ctx", "context", "glossary.md")); !os.IsNotExist(err) {
		t.Fatalf("opt-out created glossary: %v", err)
	}
}

func TestInitCommandRejectsAddonInWithAndWithout(t *testing.T) {
	repo := mkRepo(t)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init", repo, "--with", "glossary", "--without", "glossary"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "both --with and --without") {
		t.Fatalf("overlapping flags error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".ctx")); !os.IsNotExist(err) {
		t.Fatalf("overlapping flags created scaffold: %v", err)
	}
}

func TestInitCommandRejectsWithoutForNonDefaultAddon(t *testing.T) {
	repo := mkRepo(t)
	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init", repo, "--without", "contracts"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "not enabled by default") {
		t.Fatalf("non-default --without error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".ctx")); !os.IsNotExist(err) {
		t.Fatalf("invalid --without created scaffold: %v", err)
	}
}

func TestAddCommandListsAndInstallsAddon(t *testing.T) {
	var catalog bytes.Buffer
	listCmd := NewRootCmd()
	listCmd.SetOut(&catalog)
	listCmd.SetErr(&catalog)
	listCmd.SetArgs([]string{"add", "--list"})
	if err := listCmd.Execute(); err != nil {
		t.Fatalf("ctx add --list: %v", err)
	}
	for _, id := range []string{"operating", "contracts", "extending", "glossary", "review"} {
		if !strings.Contains(catalog.String(), id) {
			t.Fatalf("catalog missing %s:\n%s", id, catalog.String())
		}
	}
	var glossaryLine string
	for _, line := range strings.Split(catalog.String(), "\n") {
		if strings.HasPrefix(line, "glossary") {
			glossaryLine = line
			break
		}
	}
	if !strings.Contains(glossaryLine, "default for new scaffolds") {
		t.Fatalf("catalog does not identify glossary as the default:\n%s", catalog.String())
	}

	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam, Addons: []string{}}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	addCmd := NewRootCmd()
	addCmd.SetOut(&output)
	addCmd.SetErr(&output)
	addCmd.SetArgs([]string{"add", repo, "glossary"})
	if err := addCmd.Execute(); err != nil {
		t.Fatalf("ctx add: %v", err)
	}
	if !strings.Contains(output.String(), "installed add-on(s) glossary") {
		t.Fatalf("add output missing result:\n%s", output.String())
	}
	if _, err := os.Stat(filepath.Join(repo, ".ctx", "context", "glossary.md")); err != nil {
		t.Fatalf("glossary not installed: %v", err)
	}
}

func TestAddCommandAcceptsMultipleAddonsInCurrentDirectory(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam, Addons: []string{}}); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	}()

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"add", "contracts", "glossary"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ctx add contracts glossary: %v", err)
	}
	cfg := addTestConfig(t, filepath.Join(repo, ".ctx"))
	if strings.Join(cfg.Addons, ",") != "contracts,glossary" {
		t.Fatalf("installed add-ons = %v", cfg.Addons)
	}
}

func TestInitCommandReportsTeamHydration(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repo, ".ctx", "local")); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"init", repo})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("ctx init hydration: %v", err)
	}
	if !strings.Contains(output.String(), "hydrated local state") || !strings.Contains(output.String(), "durable context was left unchanged") {
		t.Fatalf("hydration output was ambiguous:\n%s", output.String())
	}
}

func TestInitCommandRejectsInvalidModeWithoutRenderingError(t *testing.T) {
	repo := mkRepo(t)
	var output bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"init", repo, "--mode", "private"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("ctx init accepted invalid mode")
	}
	if output.Len() != 0 {
		t.Fatalf("Cobra rendered an error despite SilenceErrors; main would duplicate it:\n%s", output.String())
	}
	if _, err := os.Lstat(filepath.Join(repo, ".ctx")); !os.IsNotExist(err) {
		t.Fatalf("invalid mode created a scaffold: %v", err)
	}
}
