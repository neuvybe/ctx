package ctx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateV2RefreshesNamedBlocksAndAdvancesTemplateRevision(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	readmePath := filepath.Join(dest, "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	stale := bytes.Replace(readme, []byte("This folder keeps durable project facts"), []byte("STALE MANAGED CONTENT"), 1)
	if bytes.Equal(stale, readme) {
		t.Fatal("could not make v2 README managed body stale")
	}
	if err := os.WriteFile(readmePath, stale, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := readLifecycleConfig(t, dest)
	cfg.TemplateRevision = "1.0.0"
	if err := writeScaffoldConfig(dest, cfg); err != nil {
		t.Fatal(err)
	}

	touched, err := Update(repo, ".ctx")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !containsString(touched, "README.md") || !containsString(touched, "INDEX.md") {
		t.Fatalf("touched = %v, want v2 managed core", touched)
	}
	after, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(after, []byte("STALE MANAGED CONTENT")) || !bytes.Contains(after, []byte("This folder keeps durable project facts")) {
		t.Fatalf("v2 README managed body was not refreshed:\n%s", after)
	}
	if got := readLifecycleConfig(t, dest).TemplateRevision; got != CurrentTemplateRevision {
		t.Fatalf("templateRevision = %q, want %q", got, CurrentTemplateRevision)
	}
	if _, err := os.Lstat(filepath.Join(dest, ".ctx-version")); !os.IsNotExist(err) {
		t.Fatalf("v2 update created legacy .ctx-version: %v", err)
	}
	requireNoUpdateTemps(t, dest)
}

func TestUpdateV2MarkerMismatchDoesNotMutateOrStampCurrent(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
	}{
		{
			name: "wrong ID",
			mutate: func(content string) string {
				return strings.ReplaceAll(content, "readme-platform", "wrong-platform")
			},
		},
		{
			name: "markerless",
			mutate: func(content string) string {
				content = strings.ReplaceAll(content, "<!-- ctx:managed begin readme-platform -->\n", "")
				return strings.ReplaceAll(content, "<!-- ctx:managed end readme-platform -->\n", "")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mkRepo(t)
			if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
				t.Fatalf("InitWithOptions: %v", err)
			}
			dest := filepath.Join(repo, ".ctx")
			cfg := readLifecycleConfig(t, dest)
			cfg.TemplateRevision = "1.0.0"
			if err := writeScaffoldConfig(dest, cfg); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(dest, configFileName)
			configBefore, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			readmePath := filepath.Join(dest, "README.md")
			readme, err := os.ReadFile(readmePath)
			if err != nil {
				t.Fatal(err)
			}
			mutated := []byte(tt.mutate(string(readme)))
			if bytes.Equal(mutated, readme) {
				t.Fatal("test mutation did not change README")
			}
			if err := os.WriteFile(readmePath, mutated, 0o644); err != nil {
				t.Fatal(err)
			}

			if touched, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "managed block ID mismatch") {
				t.Fatalf("Update = (%v, %v), want named ID-set mismatch", touched, err)
			}
			if got, err := os.ReadFile(readmePath); err != nil || !bytes.Equal(got, mutated) {
				t.Fatalf("README changed after failed update: err=%v", err)
			}
			if got, err := os.ReadFile(configPath); err != nil || !bytes.Equal(got, configBefore) {
				t.Fatalf("config/template revision changed after failed update: err=%v", err)
			}
			requireNoUpdateTemps(t, dest)
		})
	}
}

func TestDoctorV2UsesConfigTemplateRevisionWithoutLegacyStamp(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	if _, err := os.Lstat(filepath.Join(dest, ".ctx-version")); !os.IsNotExist(err) {
		t.Fatalf("v2 init wrote legacy .ctx-version: %v", err)
	}
	cfg := readLifecycleConfig(t, dest)
	cfg.TemplateRevision = "1.0.0"
	if err := writeScaffoldConfig(dest, cfg); err != nil {
		t.Fatal(err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "template revision", "current")
}

func TestDoctorV2RejectsMarkerlessManagedDocument(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	readmePath := filepath.Join(repo, ".ctx", "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	markerless := strings.ReplaceAll(string(readme), "<!-- ctx:managed begin readme-platform -->\n", "")
	markerless = strings.ReplaceAll(markerless, "<!-- ctx:managed end readme-platform -->\n", "")
	if err := os.WriteFile(readmePath, []byte(markerless), 0o644); err != nil {
		t.Fatal(err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "markers balanced in README.md", "ID mismatch")
}

func TestDoctorRequiredOutputsFollowSelectedAddons(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam, Addons: []string{"review"}}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	reviewPath := filepath.Join(repo, ".ctx", "workflows", "review.md")
	if err := os.Remove(reviewPath); err != nil {
		t.Fatal(err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "expected files present", "workflows/review.md")
}

func TestDoctorRejectsOrphanedAddonOutput(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam, Addons: []string{"review"}}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	cfg := readLifecycleConfig(t, dest)
	cfg.Addons = nil
	if err := writeScaffoldConfig(dest, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(repo, ".ctx"); err != nil {
		t.Fatalf("Update after removing add-on ID: %v", err)
	}

	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "add-on catalog consistency", "workflows/review.md")
}

func TestUpdateV2RejectsSymlinkedManagedParentWithoutOutsideWrite(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam, Addons: []string{"review"}}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	workflowsPath := filepath.Join(dest, "workflows")
	reviewPath := filepath.Join(workflowsPath, "review.md")
	review, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatal(err)
	}
	outsideBefore := bytes.Replace(review, []byte("Finish the scoped change"), []byte("STALE external content"), 1)
	if bytes.Equal(outsideBefore, review) {
		t.Fatal("could not make external review workflow stale")
	}
	outsideWorkflows := filepath.Join(t.TempDir(), "workflows")
	if err := os.Mkdir(outsideWorkflows, 0o755); err != nil {
		t.Fatal(err)
	}
	outsideReview := filepath.Join(outsideWorkflows, "review.md")
	if err := os.WriteFile(outsideReview, outsideBefore, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(workflowsPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideWorkflows, workflowsPath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	if touched, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("Update = (%v, %v), want managed-parent symlink rejection", touched, err)
	}
	if got, err := os.ReadFile(outsideReview); err != nil || !bytes.Equal(got, outsideBefore) {
		t.Fatalf("external review changed after rejected update: err=%v\n%s", err, got)
	}
	requireNoUpdateTemps(t, dest)
}

func TestV2LifecycleRefusesTemplateRevisionDowngrade(t *testing.T) {
	repo := mkRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("InitWithOptions: %v", err)
	}
	dest := filepath.Join(repo, ".ctx")
	cfg := readLifecycleConfig(t, dest)
	cfg.TemplateRevision = "99.0.0"
	if err := writeScaffoldConfig(dest, cfg); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dest, configFileName)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if touched, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Fatalf("Update = (%v, %v), want newer-revision rejection", touched, err)
	}
	if after, err := os.ReadFile(configPath); err != nil || !bytes.Equal(after, before) {
		t.Fatalf("newer config changed after rejected downgrade: err=%v", err)
	}
	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	requireFailedDoctorCheckContaining(t, checks, "template revision", "newer")
}

func readLifecycleConfig(t *testing.T, dest string) Config {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dest, configFileName))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := parseConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
