package ctx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseManaged(t *testing.T) {
	content := "pre\n" + managedBegin + "\ninner1\n" + managedEnd + "\nbetween\n" +
		managedBegin + "\ninner2a\ninner2b\n" + managedEnd + "\npost\n"
	blocks := parseManaged(content)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if got := strings.Join(blocks[0].innerLines, "\n"); got != "inner1" {
		t.Errorf("block0 inner = %q, want %q", got, "inner1")
	}
	if got := strings.Join(blocks[1].innerLines, "\n"); got != "inner2a\ninner2b" {
		t.Errorf("block1 inner = %q, want %q", got, "inner2a\ninner2b")
	}
}

func TestParseManagedNone(t *testing.T) {
	if len(parseManaged("no markers\nhere\n")) != 0 {
		t.Errorf("expected 0 blocks for markerless content")
	}
}

func TestHasManaged(t *testing.T) {
	if hasManaged("nothing") {
		t.Errorf("hasManaged(nothing) = true, want false")
	}
	if !hasManaged(managedBegin + "\nx\n" + managedEnd) {
		t.Errorf("hasManaged(block) = false, want true")
	}
}

func TestMarkersBalanced(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"x\n" + managedBegin + "\ny\n" + managedEnd + "\nz", true},
		{managedBegin + "\ny\n", false},              // dangling begin
		{managedEnd + "\ny\n" + managedBegin, false}, // end before begin
		{"no markers", true},                         // zero blocks is balanced
		{managedBegin + "\n" + managedEnd + "\n" + managedBegin + "\n" + managedEnd, true},
	}
	for _, c := range cases {
		if got := markersBalanced(c.s); got != c.want {
			t.Errorf("markersBalanced(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestUpdateManagedContentRefreshesAndPreserves(t *testing.T) {
	existing := "intro-user\n" + managedBegin + "\nOLD INNER\n" + managedEnd + "\nmiddle-user-with FILL=alice\n" +
		managedBegin + "\nOLD2\n" + managedEnd + "\ntrailer-user\n"
	newTpl := "intro-user\n" + managedBegin + "\nNEW INNER\n" + managedEnd + "\nmiddle-TEMPLATE\n" +
		managedBegin + "\nNEW2\n" + managedEnd + "\ntrailer-TEMPLATE\n"
	got, existingN, newN := updateManagedContent(existing, newTpl)
	if existingN != 2 || newN != 2 {
		t.Errorf("counts = (%d,%d), want (2,2)", existingN, newN)
	}
	// Managed inner refreshed.
	if !strings.Contains(got, "NEW INNER") || !strings.Contains(got, "NEW2") {
		t.Errorf("managed content not refreshed:\n%s", got)
	}
	if strings.Contains(got, "OLD INNER") || strings.Contains(got, "OLD2") {
		t.Errorf("old managed content not replaced:\n%s", got)
	}
	// User text from existing preserved (incl. the user fill); template user text NOT inserted.
	if !strings.Contains(got, "intro-user") || !strings.Contains(got, "middle-user-with FILL=alice") || !strings.Contains(got, "trailer-user") {
		t.Errorf("existing user text not preserved:\n%s", got)
	}
	if strings.Contains(got, "middle-TEMPLATE") || strings.Contains(got, "trailer-TEMPLATE") {
		t.Errorf("template user text leaked in:\n%s", got)
	}
}

func TestUpdateManagedContentCountMismatchLeavesExtras(t *testing.T) {
	// existing has 2 managed blocks, template has 1 -> refresh block 0, leave block 1 as-is.
	existing := managedBegin + "\nOLD0\n" + managedEnd + "\nuser\n" + managedBegin + "\nOLD1-KEEP\n" + managedEnd + "\n"
	newTpl := managedBegin + "\nNEW0\n" + managedEnd + "\n"
	got, existingN, newN := updateManagedContent(existing, newTpl)
	if existingN != 2 || newN != 1 {
		t.Errorf("counts = (%d,%d), want (2,1)", existingN, newN)
	}
	if !strings.Contains(got, "NEW0") {
		t.Errorf("block 0 not refreshed")
	}
	if !strings.Contains(got, "OLD1-KEEP") {
		t.Errorf("extra existing block 1 should be left as-is, lost it:\n%s", got)
	}
}

func TestUpdateManagedContentNoMarkersUnchanged(t *testing.T) {
	existing := "totally user content\nno markers\n"
	got, _, _ := updateManagedContent(existing, "irrelevant")
	if got != existing {
		t.Errorf("markerless existing changed:\n got: %q\nwant: %q", got, existing)
	}
}

func TestUpdateRefreshesManagedAndPreservesUserFill(t *testing.T) {
	repo := mkRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(repo, ".ctx", "README.md")
	b, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	// User fill: set the owner-instructions path (outside managed blocks).
	got := strings.ReplaceAll(string(b), "{{OWNER_INSTRUCTIONS_PATH}}", "OWNERPATH-XYZ")
	// Corrupt a managed line so we can prove it gets refreshed.
	got = strings.ReplaceAll(got, "Team mode leaves the durable files visible to Git", "CORRUPTED-MANAGED")
	if err := os.WriteFile(readmePath, []byte(got), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Update(repo, ".ctx"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "OWNERPATH-XYZ") {
		t.Errorf("user fill OWNERPATH-XYZ not preserved:\n%s", after)
	}
	if strings.Contains(string(after), "CORRUPTED-MANAGED") {
		t.Errorf("managed corruption not refreshed away:\n%s", after)
	}
	if !strings.Contains(string(after), "Team mode leaves the durable files visible to Git") {
		t.Errorf("managed line not restored from template")
	}
	// version stamp bumped to CLI Version
	v, _ := os.ReadFile(filepath.Join(repo, ".ctx", ".ctx-version"))
	if strings.TrimSpace(string(v)) != Version {
		t.Errorf(".ctx-version = %q, want %q", strings.TrimSpace(string(v)), Version)
	}
}

func TestUpdateNoCtxErrors(t *testing.T) {
	repo := mkRepo(t) // no .ctx
	if _, err := Update(repo, ".ctx"); err == nil {
		t.Errorf("Update with no .ctx should error")
	}
}

func TestUpdateSkipsUserOwnedFile(t *testing.T) {
	repo := mkRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(repo, ".ctx", "README.md")
	// Strip markers -> README becomes user-owned.
	b, _ := os.ReadFile(readmePath)
	stripped := strings.ReplaceAll(strings.ReplaceAll(string(b), managedBegin, ""), managedEnd, "")
	if err := os.WriteFile(readmePath, []byte(stripped), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(repo, ".ctx"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, _ := os.ReadFile(readmePath)
	if string(after) != stripped {
		t.Errorf("user-owned README was modified by update")
	}
	// version stamp still bumped
	v, _ := os.ReadFile(filepath.Join(repo, ".ctx", ".ctx-version"))
	if strings.TrimSpace(string(v)) != Version {
		t.Errorf(".ctx-version should still bump even when files skipped")
	}
}
