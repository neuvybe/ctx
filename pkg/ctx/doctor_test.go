package ctx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func anyFailed(checks []Check) bool {
	for _, c := range checks {
		if !c.OK {
			return true
		}
	}
	return false
}

func TestDoctorHealthy(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}
	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if anyFailed(checks) {
		for _, c := range checks {
			if !c.OK {
				t.Errorf("FAIL: %s — %s", c.Name, c.Detail)
			}
		}
	}
}

func TestDoctorNoCtx(t *testing.T) {
	repo := mkRepo(t) // no .ctx
	checks, _ := Doctor(repo, ".ctx")
	if !anyFailed(checks) {
		t.Errorf("expected a failed check when .ctx/ is absent")
	}
}

func TestDoctorMissingVersion(t *testing.T) {
	repo := mkRepo(t)
	InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	os.Remove(filepath.Join(repo, ".ctx", ".ctx-version"))
	checks, _ := Doctor(repo, ".ctx")
	if !anyFailed(checks) {
		t.Errorf("expected failure for missing .ctx-version")
	}
}

func TestDoctorMissingExclude(t *testing.T) {
	repo := mkRepo(t)
	InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeLocal})
	// Rewrite exclude without the .ctx line.
	os.WriteFile(filepath.Join(repo, ".git", "info", "exclude"), []byte("# empty\n"), 0o644)
	checks, _ := Doctor(repo, ".ctx")
	if !anyFailed(checks) {
		t.Errorf("expected failure for missing exclude entry")
	}
}

func TestDoctorLeftoverPlaceholder(t *testing.T) {
	repo := mkRepo(t)
	InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	// Inject a leftover {{PROJECT}} into a context doc.
	os.WriteFile(filepath.Join(repo, ".ctx", "context", "overview.md"),
		[]byte("# {{PROJECT}} leftover\n"), 0o644)
	checks, _ := Doctor(repo, ".ctx")
	if !anyFailed(checks) {
		t.Errorf("expected failure for leftover {{PROJECT}}")
	}
}

func TestDoctorUnbalancedMarkers(t *testing.T) {
	repo := mkRepo(t)
	InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	readme, _ := os.ReadFile(filepath.Join(repo, ".ctx", "README.md"))
	// Remove the first end marker -> unbalanced.
	broken := strings.Replace(string(readme), managedEnd, "", 1)
	os.WriteFile(filepath.Join(repo, ".ctx", "README.md"), []byte(broken), 0o644)
	checks, _ := Doctor(repo, ".ctx")
	if !anyFailed(checks) {
		t.Errorf("expected failure for unbalanced managed markers")
	}
}

func TestDoctorMissingExpectedFile(t *testing.T) {
	repo := mkRepo(t)
	InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	os.Remove(filepath.Join(repo, ".ctx", "context", "glossary.md"))
	checks, _ := Doctor(repo, ".ctx")
	if !anyFailed(checks) {
		t.Errorf("expected failure for missing expected file")
	}
}
