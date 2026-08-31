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
