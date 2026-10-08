package ctx

import (
	"bytes"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type populatedExample struct {
	root     string
	dir      string
	files    map[string][]byte
	facts    map[string]documentMetadata
	revision string
	date     string
	config   Config
}

func loadPopulatedExample(t *testing.T) populatedExample {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "docs", "examples", "ctx")
	var reference struct {
		Release    string `json:"release"`
		Revision   string `json:"revision"`
		VerifiedOn string `json:"verifiedOn"`
	}
	if err := decodeStrictJSON(addTestRead(t, filepath.Join(dir, "reference.json")), &reference); err != nil {
		t.Fatal(err)
	}
	if reference.Release == "" {
		t.Fatal("example reference must name its CLI release")
	}
	if _, _, detail := parseVerifiedAt(reference.Revision + " @ " + reference.VerifiedOn); detail != "" {
		t.Fatal(detail)
	}
	walkthrough := string(addTestRead(t, filepath.Join(dir, "README.md")))
	for name, value := range map[string]string{"release": reference.Release, "revision": reference.Revision, "verification date": reference.VerifiedOn} {
		if !strings.Contains(walkthrough, value) {
			t.Fatalf("walkthrough does not name the recorded %s: %s", name, value)
		}
	}
	example := populatedExample{
		root: root, dir: dir, files: map[string][]byte{},
		facts: map[string]documentMetadata{}, revision: reference.Revision, date: reference.VerifiedOn,
	}
	scaffold := filepath.Join(dir, "scaffold")
	err = filepath.WalkDir(scaffold, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			t.Fatalf("example contains a non-regular file: %s", path)
		}
		name, err := filepath.Rel(scaffold, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		data := addTestRead(t, path)
		example.files[name] = data
		if strings.HasPrefix(name, "local/") {
			t.Fatalf("publish continuation outside the example scaffold: %s", name)
		}
		if strings.HasPrefix(name, "context/") && strings.HasSuffix(name, ".md") {
			metadata, found, err := parseDocumentMetadata(data)
			if err != nil || !found {
				t.Fatalf("%s metadata: found=%v, err=%v", name, found, err)
			}
			if metadata.Status != "verified" || metadata.VerifiedAt != reference.Revision+" @ "+reference.VerifiedOn || len(metadata.Sources) == 0 {
				t.Fatalf("%s must carry the example's recorded verification and evidence: %+v", name, metadata)
			}
			for _, source := range metadata.Sources {
				normalized, err := validateStatusSourcePath(root, ".ctx", source)
				if err != nil {
					t.Fatalf("%s source %s: %v", name, source, err)
				}
				info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(normalized)))
				if err != nil || !info.Mode().IsRegular() {
					t.Fatalf("%s source %s is not a regular repository file: %v", name, source, err)
				}
			}
			example.facts[name] = metadata
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	example.config, err = parseConfig(example.files[configFileName])
	if err != nil {
		t.Fatal(err)
	}
	if example.config.Mode != ModeTeam || example.config.Project != "ctx" || !reflect.DeepEqual(example.config.Addons, DefaultAddonIDs()) {
		t.Fatalf("example must demonstrate the default team scaffold: %+v", example.config)
	}
	required, err := RequiredOutputs(example.config.LayoutVersion, example.config.Addons, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range required {
		if name == "local/CONTINUE.md" {
			_ = addTestRead(t, filepath.Join(dir, "CONTINUE.example.md"))
		} else if _, exists := example.files[name]; !exists {
			t.Fatalf("example is missing required durable output %s", name)
		}
	}
	return example
}

func TestPopulatedContextExampleMetadataAndRoutes(t *testing.T) {
	example := loadPopulatedExample(t)
	// The reference is intentionally pinned. Validate structure and source
	// paths without requiring historical Git objects in shallow CI checkouts;
	// these checks do not re-certify prose against a moving HEAD.
	markdownLink := regexp.MustCompile(`\[[^\]\n]*\]\(([^()\s]+)\)`)
	routes := map[string][]string{}
	scaffold := filepath.Join(example.dir, "scaffold")
	err := filepath.WalkDir(example.dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return walkErr
		}
		for _, match := range markdownLink.FindAllSubmatch(addTestRead(t, path), -1) {
			link, err := url.Parse(string(match[1]))
			if err != nil {
				t.Fatalf("%s has an invalid link: %v", path, err)
			}
			if link.IsAbs() || link.Host != "" || link.Path == "" {
				continue
			}
			target := filepath.Join(filepath.Dir(path), filepath.FromSlash(link.Path))
			rel, err := filepath.Rel(example.root, target)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				t.Fatalf("%s link escapes the repository: %s", path, link.Path)
			}
			if _, err := os.Stat(target); err != nil {
				t.Fatalf("%s has a broken local link to %s: %v", path, link.Path, err)
			}
			from, _ := filepath.Rel(scaffold, path)
			to, _ := filepath.Rel(scaffold, target)
			routes[filepath.ToSlash(from)] = append(routes[filepath.ToSlash(from)], filepath.ToSlash(to))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Managed INDEX routes use code-formatted paths, while authored routes use
	// Markdown links. Check both so an orphaned child cannot silently pass.
	indexPath := regexp.MustCompile("\x60(context/[^\x60]+\\.md)\x60")
	for _, match := range indexPath.FindAllSubmatch(example.files["INDEX.md"], -1) {
		target := string(match[1])
		if _, exists := example.files[target]; !exists {
			t.Fatalf("INDEX routes to missing document %s", target)
		}
		routes["INDEX.md"] = append(routes["INDEX.md"], target)
	}
	reachable := map[string]bool{}
	queue := []string{"INDEX.md"}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if reachable[current] {
			continue
		}
		reachable[current] = true
		queue = append(queue, routes[current]...)
	}
	for name := range example.facts {
		if !reachable[name] {
			t.Errorf("fact document is not reachable from INDEX: %s", name)
		}
	}
}

func TestPopulatedContextExampleLifecycle(t *testing.T) {
	example := loadPopulatedExample(t)
	draftRepo := mkRepo(t)
	runExampleCommand(t, true, "init", draftRepo)
	runExampleCommand(t, true, "doctor", draftRepo)
	draft := runExampleCommand(t, false, "status", draftRepo)
	if !strings.Contains(draft, "✗ context/behavior.md — draft") || !strings.Contains(draft, "✗ context/glossary.md — draft") {
		t.Fatalf("generated context does not match the draft output shown in the walkthrough:\n%s", draft)
	}

	repo := mkRepo(t)
	sources := map[string]bool{}
	for _, metadata := range example.facts {
		for _, source := range metadata.Sources {
			sources[source] = true
		}
	}
	for source := range sources {
		writeStatusFile(t, filepath.Join(repo, filepath.FromSlash(source)), string(addTestRead(t, filepath.Join(example.root, filepath.FromSlash(source)))))
	}
	runStatusGit(t, repo, "add", ".")
	runStatusGit(t, repo, "-c", "user.name=ctx-example-test", "-c", "user.email=ctx@example.invalid", "commit", "-qm", "record example evidence")
	revision := strings.TrimSpace(runStatusGit(t, repo, "rev-parse", "HEAD"))

	// Use an isolated evidence commit for lifecycle checks, rather than depend
	// on the snapshot's historical objects or a clean developer worktree.
	// Only test copies are rebased; stored verification metadata stays pinned.
	dest := filepath.Join(repo, ".ctx")
	for name, data := range example.files {
		if _, fact := example.facts[name]; fact {
			data = bytes.Replace(data, []byte(example.revision+" @ "+example.date), []byte(revision+" @ "+example.date), 1)
		}
		writeStatusFile(t, filepath.Join(dest, filepath.FromSlash(name)), string(data))
	}
	beforeHydration, err := snapshotScaffoldTree(dest)
	if err != nil {
		t.Fatal(err)
	}
	runExampleCommand(t, true, "init", repo)
	for name, entry := range beforeHydration {
		if entry.mode.IsRegular() {
			addTestEqualFile(t, filepath.Join(dest, name), entry.data)
		}
	}
	continuation := filepath.Join(dest, "local", "CONTINUE.md")
	writeStatusFile(t, continuation, string(addTestRead(t, filepath.Join(example.dir, "CONTINUE.example.md"))))
	if !modeTestGitIgnored(t, repo, ".ctx/local/CONTINUE.md") {
		t.Fatal("sample continuation would be visible to ordinary Git tracking")
	}
	for name := range example.files {
		if modeTestGitIgnored(t, repo, ".ctx/"+name) {
			t.Fatalf("durable example output is unexpectedly ignored: %s", name)
		}
	}
	runExampleCommand(t, true, "doctor", repo)
	runExampleCommand(t, true, "status", repo)

	factsBefore := map[string][]byte{}
	for name := range example.facts {
		factsBefore[name] = addTestRead(t, filepath.Join(dest, filepath.FromSlash(name)))
	}
	index := addTestRead(t, filepath.Join(dest, "INDEX.md"))
	ownerRoutes := bytes.SplitN(index, []byte("<!-- ctx:managed end index-routing -->"), 2)[1]
	runExampleCommand(t, true, "update", repo)
	addTestEqualFile(t, continuation, addTestRead(t, filepath.Join(example.dir, "CONTINUE.example.md")))
	for name, data := range factsBefore {
		addTestEqualFile(t, filepath.Join(dest, filepath.FromSlash(name)), data)
	}
	if !bytes.HasSuffix(addTestRead(t, filepath.Join(dest, "INDEX.md")), ownerRoutes) {
		t.Fatal("update changed the example's project-owned routing")
	}
	if cfg := addTestConfig(t, dest); !reflect.DeepEqual(cfg.Addons, example.config.Addons) {
		t.Fatalf("update changed selected add-ons: %v", cfg.Addons)
	}

	sourcePath := filepath.Join(repo, "pkg", "ctx", "init.go")
	source := addTestRead(t, sourcePath)
	writeStatusFile(t, sourcePath, string(source)+"\n// Example: evidence has changed since verification.\n")
	stale := runExampleCommand(t, false, "status", repo)
	changedChild := false
	for _, line := range strings.Split(stale, "\n") {
		if strings.HasPrefix(line, "✗ context/behavior/initialization.md — ") && strings.Contains(line, "pkg/ctx/init.go") {
			changedChild = true
		}
	}
	if !changedChild {
		t.Fatalf("status did not identify the child and changed evidence:\n%s", stale)
	}
	runExampleCommand(t, true, "doctor", repo)
	runExampleCommand(t, true, "update", repo)
	runExampleCommand(t, false, "status", repo) // Update cannot reverify project facts.

	runStatusGit(t, repo, "add", "pkg/ctx/init.go")
	runStatusGit(t, repo, "-c", "user.name=ctx-example-test", "-c", "user.email=ctx@example.invalid", "commit", "-qm", "change example evidence")
	newRevision := strings.TrimSpace(runStatusGit(t, repo, "rev-parse", "HEAD"))
	for name := range example.facts {
		path := filepath.Join(dest, filepath.FromSlash(name))
		data := bytes.Replace(addTestRead(t, path), []byte(revision+" @ "+example.date), []byte(newRevision+" @ "+example.date), 1)
		writeStatusFile(t, path, string(data))
	}
	runExampleCommand(t, true, "status", repo)
	runStatusGit(t, repo, "add", ".ctx")
	trackedLocal, err := gitTrackedFiles(repo, ".ctx/local")
	if err != nil || len(trackedLocal) != 0 {
		t.Fatalf("ordinary staging captured local state: %v, err=%v", trackedLocal, err)
	}
	runStatusGit(t, repo, "-c", "user.name=ctx-example-test", "-c", "user.email=ctx@example.invalid", "commit", "-qm", "share example context")
	clone := filepath.Join(t.TempDir(), "teammate")
	runStatusGit(t, repo, "clone", "--quiet", "--no-hardlinks", repo, clone)
	if _, err := os.Lstat(filepath.Join(clone, ".ctx", "local", "CONTINUE.md")); !os.IsNotExist(err) {
		t.Fatalf("fresh clone inherited continuation: %v", err)
	}
	sharedBefore, err := snapshotScaffoldTree(filepath.Join(clone, ".ctx"))
	if err != nil {
		t.Fatal(err)
	}
	runExampleCommand(t, true, "init", clone)
	for name, entry := range sharedBefore {
		if entry.mode.IsRegular() {
			addTestEqualFile(t, filepath.Join(clone, ".ctx", name), entry.data)
		}
	}
	runExampleCommand(t, true, "doctor", clone)
	runExampleCommand(t, true, "status", clone)
	runExampleCommand(t, false, "init", clone) // Existing continuation is never reset.

	runExampleCommand(t, true, "add", "--list")
	runExampleCommand(t, true, "add", clone, "contracts")
	runExampleCommand(t, true, "doctor", clone)
	addedDraft := runExampleCommand(t, false, "status", clone)
	if !strings.Contains(addedDraft, "✗ context/contracts.md — draft") {
		t.Fatalf("new add-on did not return the example to draft readiness:\n%s", addedDraft)
	}
}

func runExampleCommand(t *testing.T, wantSuccess bool, args ...string) string {
	t.Helper()
	var output bytes.Buffer
	cmd := NewRootCmd()
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs(args)
	err := cmd.Execute()
	if (err == nil) != wantSuccess {
		t.Fatalf("ctx %s: err=%v, want success=%v\n%s", strings.Join(args, " "), err, wantSuccess, output.String())
	}
	return output.String()
}
