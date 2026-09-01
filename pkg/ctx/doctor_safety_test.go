package ctx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorRejectsSymlinkedRequiredOutput(t *testing.T) {
	repo := mkRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	readmePath := filepath.Join(dest, "README.md")
	outsidePath := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outsidePath, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(readmePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, readmePath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "markers balanced in README.md", "symbolic link")
	requireFailedDoctorCheckContaining(t, checks, "expected files present", "README.md")
}

func TestDoctorRejectsSymlinkedConfig(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	configPath := filepath.Join(dest, configFileName)
	outsidePath := filepath.Join(t.TempDir(), "outside-config.json")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outsidePath, config, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(configPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, configPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "scaffold config", "symbolic link")
}

func TestDoctorRejectsSymlinkedRequiredParentDirectory(t *testing.T) {
	repo := mkRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	contextPath := filepath.Join(dest, "context")
	outsidePath := filepath.Join(t.TempDir(), "context")
	if err := os.Rename(contextPath, outsidePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, contextPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "expected files present", "context/")
}

func TestDoctorRejectsNonRegularRequiredOutput(t *testing.T) {
	repo := mkRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	readmePath := filepath.Join(repo, ".ctx", "README.md")
	if err := os.Remove(readmePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(readmePath, 0o755); err != nil {
		t.Fatal(err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "markers balanced in README.md", "not a regular file")
	requireFailedDoctorCheckContaining(t, checks, "expected files present", "README.md")
}

func TestDoctorRejectsSymlinkedLegacyVersionStamp(t *testing.T) {
	repo := mkRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	versionPath := filepath.Join(repo, ".ctx", ".ctx-version")
	outsidePath := filepath.Join(t.TempDir(), "outside-version")
	if err := os.WriteFile(outsidePath, []byte("sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(versionPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsidePath, versionPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, ".ctx-version stamp", "symbolic link")
}

func TestDoctorRejectsEmptyLegacyVersionStamp(t *testing.T) {
	repo := mkRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".ctx", ".ctx-version"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, ".ctx-version stamp", "empty")
}

func requireFailedDoctorCheckContaining(t *testing.T, checks []Check, name, detail string) {
	t.Helper()
	for _, check := range checks {
		if check.Name == name && !check.OK && strings.Contains(check.Detail, detail) {
			return
		}
	}
	t.Fatalf("missing failed Doctor check %q containing %q; checks=%+v", name, detail, checks)
}
