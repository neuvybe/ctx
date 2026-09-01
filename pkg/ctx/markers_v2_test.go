package ctx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func namedManagedBlock(id, body string) string {
	return "<!-- ctx:managed begin " + id + " -->\n" + body + "<!-- ctx:managed end " + id + " -->"
}

func TestNamedManagedUpdateMatchesByIDAndPreservesOutsideBytes(t *testing.T) {
	existing := "user-prefix\r\n" + namedManagedBlock("index-routing", "OLD-INDEX\n") +
		"\r\nuser-middle\n" + namedManagedBlock("readme-platform", "OLD-README\n") + "\nuser-suffix"
	template := namedManagedBlock("readme-platform", "NEW-README\n") + "\n" +
		namedManagedBlock("index-routing", "NEW-INDEX\n")

	got, err := updateManagedContentStrict(existing, template, managedMarkersNamed)
	if err != nil {
		t.Fatalf("updateManagedContentStrict: %v", err)
	}
	want := "user-prefix\r\n" + namedManagedBlock("index-routing", "NEW-INDEX\n") +
		"\r\nuser-middle\n" + namedManagedBlock("readme-platform", "NEW-README\n") + "\nuser-suffix"
	if got != want {
		t.Errorf("updated content did not preserve exact target framing\n got: %q\nwant: %q", got, want)
	}
}

func TestNamedManagedGrammarRejectsMalformedAndMixedMarkers(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "unnamed v1 marker", content: managedBegin + "\nx\n" + managedEnd},
		{name: "mixed formats", content: namedManagedBlock("readme-platform", "x\n") + "\n" + managedBegin + "\ny\n" + managedEnd},
		{name: "mismatched end", content: "<!-- ctx:managed begin readme-platform -->\nx\n<!-- ctx:managed end index-routing -->"},
		{name: "duplicate ID", content: namedManagedBlock("readme-platform", "x\n") + "\n" + namedManagedBlock("readme-platform", "y\n")},
		{name: "nested", content: "<!-- ctx:managed begin readme-platform -->\n<!-- ctx:managed begin index-routing -->\n<!-- ctx:managed end index-routing -->\n<!-- ctx:managed end readme-platform -->"},
		{name: "uppercase ID", content: namedManagedBlock("Readme-Platform", "x\n")},
		{name: "underscore ID", content: namedManagedBlock("readme_platform", "x\n")},
		{name: "leading whitespace", content: " <!-- ctx:managed begin readme-platform -->\nx\n<!-- ctx:managed end readme-platform -->"},
		{name: "trailing whitespace", content: "<!-- ctx:managed begin readme-platform --> \nx\n<!-- ctx:managed end readme-platform -->"},
		{name: "dangling begin", content: "<!-- ctx:managed begin readme-platform -->\nx\n"},
		{name: "end first", content: "<!-- ctx:managed end readme-platform -->"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseManagedDocument(tt.content, managedMarkersNamed); err == nil {
				t.Fatalf("parseManagedDocument accepted malformed content:\n%s", tt.content)
			}
		})
	}
}

func TestManagedFormatsDoNotCrossAccept(t *testing.T) {
	unnamed := managedBegin + "\nlegacy\n" + managedEnd
	named := namedManagedBlock("readme-platform", "current\n")
	if _, err := parseManagedDocument(unnamed, managedMarkersNamed); err == nil {
		t.Fatal("v2 parser accepted unnamed v1 markers")
	}
	if _, err := parseManagedDocument(named, managedMarkersUnnamed); err == nil {
		t.Fatal("v1 parser accepted named v2 markers")
	}
}

func TestStrictManagedUpdateRejectsMismatchedBlocks(t *testing.T) {
	tests := []struct {
		name     string
		format   managedMarkerFormat
		existing string
		template string
	}{
		{
			name:     "v1 ordinal count",
			format:   managedMarkersUnnamed,
			existing: managedBegin + "\none\n" + managedEnd + "\n" + managedBegin + "\ntwo\n" + managedEnd,
			template: managedBegin + "\nnew\n" + managedEnd,
		},
		{
			name:     "v2 ID set",
			format:   managedMarkersNamed,
			existing: namedManagedBlock("readme-platform", "old\n"),
			template: namedManagedBlock("index-routing", "new\n"),
		},
		{
			name:     "v2 target subset",
			format:   managedMarkersNamed,
			existing: namedManagedBlock("readme-platform", "old\n"),
			template: namedManagedBlock("readme-platform", "new\n") + "\n" + namedManagedBlock("index-routing", "new\n"),
		},
		{
			name:     "v2 target superset",
			format:   managedMarkersNamed,
			existing: namedManagedBlock("readme-platform", "old\n") + "\n" + namedManagedBlock("index-routing", "old\n"),
			template: namedManagedBlock("readme-platform", "new\n"),
		},
		{
			name:     "v2 empty target",
			format:   managedMarkersNamed,
			existing: "user-owned-looking content\n",
			template: namedManagedBlock("readme-platform", "new\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateManagedContentStrict(tt.existing, tt.template, tt.format)
			if err == nil {
				t.Fatalf("updateManagedContentStrict succeeded with mismatched blocks: %q", got)
			}
		})
	}
}

func TestUpdateV1BlockMismatchDoesNotWriteContentOrVersion(t *testing.T) {
	repo := mkRepo(t)
	dest := filepath.Join(repo, ".ctx")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "CONTINUE.md"), []byte("legacy local state\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	readme := managedBegin + "\nfirst\n" + managedEnd + "\n" +
		managedBegin + "\nsecond\n" + managedEnd + "\n" +
		managedBegin + "\nextra\n" + managedEnd + "\n"
	readmePath := filepath.Join(dest, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	// Markerless v1 managed outputs are explicitly user-owned and skipped.
	if err := os.WriteFile(filepath.Join(dest, "REVIEW.md"), []byte("user owned\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(dest, ".ctx-version")
	version := []byte("before-mismatch\n")
	if err := os.WriteFile(versionPath, version, 0o644); err != nil {
		t.Fatal(err)
	}

	if touched, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "count mismatch") {
		t.Fatalf("Update = (%v, %v), want block-count mismatch", touched, err)
	}
	if got, err := os.ReadFile(readmePath); err != nil || string(got) != readme {
		t.Fatalf("README changed after failed update: got=%q err=%v", got, err)
	}
	if got, err := os.ReadFile(versionPath); err != nil || string(got) != string(version) {
		t.Fatalf("version changed after failed update: got=%q err=%v", got, err)
	}
}
