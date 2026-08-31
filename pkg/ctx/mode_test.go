package ctx

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestInitWithOptionsTeamMode(t *testing.T) {
	repo := modeTestGitRepo(t)

	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("InitWithOptions(team): %v", err)
	}

	modeTestRequireFile(t, filepath.Join(repo, ".ctx", "config.json"))
	modeTestRequireFile(t, filepath.Join(repo, ".ctx", ".gitignore"))
	modeTestRequireFile(t, filepath.Join(repo, ".ctx", "local", "CONTINUE.md"))

	configBytes, err := os.ReadFile(filepath.Join(repo, ".ctx", "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	var config struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}
	if config.Mode != "team" {
		t.Errorf("config mode = %q, want team", config.Mode)
	}

	if modeTestGitIgnored(t, repo, ".ctx/config.json") {
		t.Error("team config is ignored; team mode must not exclude the whole .ctx folder")
	}
	if !modeTestGitIgnored(t, repo, ".ctx/local/CONTINUE.md") {
		t.Error("team-local CONTINUE.md is not effectively ignored by Git")
	}
	if modeTestHasWholeFolderExclude(t, repo, ".ctx") {
		t.Error("team mode added a whole-folder .ctx entry to .git/info/exclude")
	}
}

func TestTeamInitRechecksVisibilityAfterStaging(t *testing.T) {
	repo := modeTestGitRepo(t)
	mutated := make(chan error, 1)
	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			entries, err := os.ReadDir(repo)
			if err != nil {
				mutated <- err
				return
			}
			for _, entry := range entries {
				if entry.IsDir() && strings.HasPrefix(entry.Name(), ".ctx-init-") {
					mutated <- os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/\n"), 0o644)
					return
				}
			}
			time.Sleep(100 * time.Microsecond)
		}
		mutated <- errors.New("timed out waiting for init staging directory")
	}()

	err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	if mutateErr := <-mutated; mutateErr != nil {
		t.Fatal(mutateErr)
	}
	if err == nil || !strings.Contains(err.Error(), "postcondition") {
		t.Fatalf("team init error = %v, want post-publication visibility failure", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ".ctx")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed team init left published scaffold: %v", statErr)
	}
	if leftovers, globErr := filepath.Glob(filepath.Join(repo, ".ctx-init-*")); globErr != nil || len(leftovers) != 0 {
		t.Fatalf("failed team init left staging paths: %v (err=%v)", leftovers, globErr)
	}
}

func TestInitWithOptionsLocalModeExcludesWholeFolder(t *testing.T) {
	repo := modeTestGitRepo(t)

	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeLocal}); err != nil {
		t.Fatalf("InitWithOptions(local): %v", err)
	}

	modeTestRequireFile(t, filepath.Join(repo, ".ctx", "README.md"))
	if !modeTestGitIgnored(t, repo, ".ctx/README.md") {
		t.Error("local mode did not effectively ignore the whole .ctx folder")
	}
	if !modeTestHasWholeFolderExclude(t, repo, ".ctx") {
		t.Error("local mode did not add a whole-folder .ctx entry to .git/info/exclude")
	}
}

func TestInitWithOptionsRejectsInvalidMode(t *testing.T) {
	repo := modeTestGitRepo(t)

	err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: "invalid"})
	if err == nil {
		t.Fatal("InitWithOptions accepted an invalid mode")
	}
}

func TestInitWithOptionsRejectsFolderTraversal(t *testing.T) {
	tests := []string{
		"../escape",
		"nested/../../escape",
	}

	for _, folder := range tests {
		t.Run(strings.ReplaceAll(folder, "/", "_"), func(t *testing.T) {
			root := t.TempDir()
			repo := filepath.Join(root, "repo")
			if err := os.Mkdir(repo, 0o755); err != nil {
				t.Fatalf("mkdir repo: %v", err)
			}
			modeTestGitInit(t, repo)

			err := InitWithOptions(repo, InitOptions{Folder: folder, Mode: ModeTeam})
			if err == nil {
				t.Fatalf("InitWithOptions accepted traversal folder %q", folder)
			}
			if _, statErr := os.Lstat(filepath.Join(root, "escape")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("traversal folder %q created a path outside the repo (stat error: %v)", folder, statErr)
			}
		})
	}
}

func TestInitWithOptionsTeamRejectsEffectiveIgnore(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	if err == nil || !strings.Contains(err.Error(), "ignored by Git") {
		t.Fatalf("team init error = %v, want effective-ignore error", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ".ctx")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed team init published a scaffold: %v", statErr)
	}
}

func TestInitWithOptionsTeamRejectsIgnoredSharedFile(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/context/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	if err == nil || !strings.Contains(err.Error(), ".ctx/context/overview.md") {
		t.Fatalf("team init error = %v, want ignored shared-file error", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ".ctx")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed team init published a scaffold: %v", statErr)
	}
}

func TestDoctorTeamChecksEverySharedAndLocalFile(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/INDEX.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if !hasFailedCheckContaining(checks, "shared context Git-visible", ".ctx/INDEX.md") {
		t.Fatalf("Doctor did not report ignored shared file: %+v", checks)
	}

	if err := os.Remove(filepath.Join(repo, ".gitignore")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".ctx", ".gitignore"), []byte("/local/CONTINUE.md\n/local/.ctx-ignore-probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".ctx", "local", "notes.md"), []byte("private\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checks, err = Doctor(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if !hasFailedCheckContaining(checks, "local state private", "local") {
		t.Fatalf("Doctor did not report visible local file or invalid rule: %+v", checks)
	}
}

func TestInitWithOptionsLocalVerifiesPrivacyAndRollsBack(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("!/.ctx/\n!/.ctx/README.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exclude := filepath.Join(repo, ".git", "info", "exclude")
	before, err := os.ReadFile(exclude)
	if err != nil {
		t.Fatal(err)
	}

	err = InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeLocal})
	if err == nil || !strings.Contains(err.Error(), "could not ignore") {
		t.Fatalf("local init error = %v, want privacy postcondition failure", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ".ctx")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed local init published a scaffold: %v", statErr)
	}
	after, err := os.ReadFile(exclude)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("failed local init did not restore info/exclude\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestLocalModeRejectsVisibleDirectoryAndFutureFiles(t *testing.T) {
	repo := modeTestGitRepo(t)
	rules := "!/.ctx/\n/.ctx/*\n!/.ctx/future-secret.txt\n"
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(rules), 0o644); err != nil {
		t.Fatal(err)
	}
	exclude := filepath.Join(repo, ".git", "info", "exclude")
	before, err := os.ReadFile(exclude)
	if err != nil {
		t.Fatal(err)
	}

	err = InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeLocal})
	if err == nil || !strings.Contains(err.Error(), "as a directory") {
		t.Fatalf("local init error = %v, want visible-directory failure", err)
	}
	if _, statErr := os.Lstat(filepath.Join(repo, ".ctx")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed local init published a scaffold: %v", statErr)
	}
	after, err := os.ReadFile(exclude)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("failed local init did not restore info/exclude\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestDoctorRejectsVisibleLocalDirectory(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeLocal}); err != nil {
		t.Fatal(err)
	}
	rules := "!/.ctx/\n/.ctx/*\n!/.ctx/future-secret.txt\n"
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(rules), 0o644); err != nil {
		t.Fatal(err)
	}
	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if !hasFailedCheckContaining(checks, "local context ignored", ".ctx/ (directory)") {
		t.Fatalf("Doctor did not reject visible local directory: %+v", checks)
	}
}

func TestInitializedLocalPostconditionRejectsVisibleDirectoryAndFutureFiles(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeLocal}); err != nil {
		t.Fatal(err)
	}
	rules := "!/.ctx/\n/.ctx/*\n!/.ctx/future-secret.txt\n"
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(rules), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := loadScaffoldState(filepath.Join(repo, ".ctx"))
	if err != nil {
		t.Fatal(err)
	}
	err = verifyInitializedScaffold(repo, filepath.Join(repo, ".ctx"), ".ctx", state)
	if err == nil || !strings.Contains(err.Error(), "as a directory") {
		t.Fatalf("local postcondition error = %v, want visible-directory failure", err)
	}
}

func TestHydrationCleanupPreservesReplacementDirectory(t *testing.T) {
	root := t.TempDir()
	localDir := filepath.Join(root, "local")
	if err := os.Mkdir(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createdInfo, err := os.Lstat(localDir)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(root, "local-created-by-ctx")
	if err := os.Rename(localDir, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(localDir, 0o700); err != nil {
		t.Fatal(err)
	}

	removeHydrationLocalDirectory(localDir, createdInfo)

	info, err := os.Lstat(localDir)
	if err != nil {
		t.Fatalf("cleanup removed a replacement directory: %v", err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("replacement directory changed: mode %v", info.Mode())
	}
}

func TestHydrationCleanupRemovesOwnedEmptyDirectory(t *testing.T) {
	localDir := filepath.Join(t.TempDir(), "local")
	if err := os.Mkdir(localDir, 0o755); err != nil {
		t.Fatal(err)
	}
	createdInfo, err := os.Lstat(localDir)
	if err != nil {
		t.Fatal(err)
	}

	removeHydrationLocalDirectory(localDir, createdInfo)

	if _, err := os.Lstat(localDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned empty hydration directory was not removed: %v", err)
	}
}

func TestInitHydratesFreshTeamCloneLocalStateOnly(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}
	readmePath := filepath.Join(repo, ".ctx", "README.md")
	before, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(repo, ".ctx", "local")); err != nil {
		t.Fatal(err)
	}

	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatalf("hydrate team local state: %v", err)
	}
	modeTestRequireFile(t, filepath.Join(repo, ".ctx", "local", "CONTINUE.md"))
	after, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("team hydration changed a durable file")
	}
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err == nil {
		t.Fatal("team init overwrote an already-hydrated scaffold")
	}
}

func TestSchemaV1HydrationPreservesRenderedProjectAfterCloneRename(t *testing.T) {
	for _, tt := range []struct {
		name string
		crlf bool
	}{
		{name: "LF"},
		{name: "CRLF", crlf: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			original := filepath.Join(root, "original-project")
			if err := os.Mkdir(original, 0o755); err != nil {
				t.Fatal(err)
			}
			modeTestGitInit(t, original)
			if err := Init(original, ".ctx"); err != nil {
				t.Fatalf("legacy Init: %v", err)
			}
			dest := filepath.Join(original, ".ctx")
			if err := os.Remove(filepath.Join(dest, "CONTINUE.md")); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dest, ".gitignore"), []byte("/local/\n**/.ctx-update-*\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := writeConfig(dest, ModeTeam); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(original, ".git", "info", "exclude"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			if tt.crlf {
				for _, name := range []string{"README.md", "INDEX.md"} {
					path := filepath.Join(dest, name)
					content, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					content = []byte(strings.ReplaceAll(string(content), "\n", "\r\n"))
					if err := os.WriteFile(path, content, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}

			renamed := filepath.Join(root, "renamed-clone")
			if err := os.Rename(original, renamed); err != nil {
				t.Fatal(err)
			}
			if err := InitWithOptions(renamed, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
				t.Fatalf("hydrate renamed schema-v1 clone: %v", err)
			}
			continuation, err := os.ReadFile(filepath.Join(renamed, ".ctx", "local", "CONTINUE.md"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(continuation), "for original-project") || strings.Contains(string(continuation), "for renamed-clone") {
				t.Fatalf("hydrated continuation changed project identity:\n%s", continuation)
			}
		})
	}
}

func TestTeamHydrationRejectsIncompleteOrIgnoredSharedFiles(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, repo string)
	}{
		{
			name: "missing durable file",
			mutate: func(t *testing.T, repo string) {
				t.Helper()
				if err := os.Remove(filepath.Join(repo, ".ctx", "README.md")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "ignored custom shared file",
			mutate: func(t *testing.T, repo string) {
				t.Helper()
				custom := filepath.Join(repo, ".ctx", "context", "custom.md")
				if err := os.WriteFile(custom, []byte("custom\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/context/custom.md\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := modeTestGitRepo(t)
			if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(filepath.Join(repo, ".ctx", "local")); err != nil {
				t.Fatal(err)
			}
			tt.mutate(t, repo)
			if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err == nil {
				t.Fatal("hydration accepted an unhealthy team scaffold")
			}
			if _, err := os.Lstat(filepath.Join(repo, ".ctx", "local", "CONTINUE.md")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed hydration published local state: %v", err)
			}
		})
	}
}

func TestDoctorRejectsVisibleTeamLocalDirectory(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}
	rules := "/local/\n!/loc*/\n/local/*\n!/local/future-secret.txt\n/local/CONTINUE.md\n/local/.ctx-ignore-probe\n"
	if err := os.WriteFile(filepath.Join(repo, ".ctx", ".gitignore"), []byte(rules), 0o644); err != nil {
		t.Fatal(err)
	}
	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if !hasFailedCheckContaining(checks, "local state private", ".ctx/local/") {
		t.Fatalf("Doctor did not reject visible team local directory: %+v", checks)
	}
}

func TestConcurrentLocalInitPublishesOnePrivateScaffold(t *testing.T) {
	repo := modeTestGitRepo(t)
	const workers = 8
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeLocal})
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent local init successes = %d, want 1", successes)
	}
	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatal(err)
	}
	if anyFailed(checks) {
		t.Fatalf("winning concurrent local scaffold is unhealthy: %+v", checks)
	}
}

func TestConcurrentLinkedWorktreeModesCannotBothPublish(t *testing.T) {
	root := t.TempDir()
	mainRepo := filepath.Join(root, "main")
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(mainRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	modeTestGitInit(t, mainRepo)
	modeTestRunGit(t, mainRepo, "-c", "user.name=ctx-test", "-c", "user.email=ctx@example.invalid", "commit", "--allow-empty", "-qm", "initial")
	modeTestRunGit(t, mainRepo, "worktree", "add", "-qb", "ctx-mode-race-test", worktree)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	// Both initializations contend on the common Git-directory lock. Whichever
	// mode publishes first makes the other invalid: team mode leaves a sibling
	// scaffold, while local mode installs a common whole-folder exclusion. The
	// winner is intentionally nondeterministic; exactly one publish is not.
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		errs <- InitWithOptions(mainRepo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	}()
	go func() {
		defer wg.Done()
		<-start
		errs <- InitWithOptions(worktree, InitOptions{Folder: ".ctx", Mode: ModeLocal})
	}()
	close(start)
	wg.Wait()
	close(errs)
	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent linked-worktree init successes = %d, want 1", successes)
	}

	_, mainErr := os.Stat(filepath.Join(mainRepo, ".ctx"))
	_, worktreeErr := os.Stat(filepath.Join(worktree, ".ctx"))
	mainPublished := mainErr == nil
	worktreePublished := worktreeErr == nil
	if mainErr != nil && !errors.Is(mainErr, os.ErrNotExist) {
		t.Fatalf("inspect main-worktree scaffold: %v", mainErr)
	}
	if worktreeErr != nil && !errors.Is(worktreeErr, os.ErrNotExist) {
		t.Fatalf("inspect linked-worktree scaffold: %v", worktreeErr)
	}
	if mainPublished == worktreePublished {
		t.Fatalf("published destinations: main=%v linked=%v, want exactly one", mainPublished, worktreePublished)
	}

	if mainPublished {
		checks, doctorErr := Doctor(mainRepo, ".ctx")
		if doctorErr != nil || anyFailed(checks) {
			t.Fatalf("winning team scaffold unhealthy: err=%v checks=%+v", doctorErr, checks)
		}
	} else {
		checks, doctorErr := Doctor(worktree, ".ctx")
		if doctorErr != nil || anyFailed(checks) {
			t.Fatalf("winning local scaffold unhealthy: err=%v checks=%+v", doctorErr, checks)
		}
	}
}

func TestGitCommandsIgnoreInheritedRepositorySelection(t *testing.T) {
	repo := modeTestGitRepo(t)
	other := modeTestGitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("/.ctx/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	t.Setenv("GIT_WORK_TREE", other)
	err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	if err == nil || !strings.Contains(err.Error(), "ignored by Git") {
		t.Fatalf("team init with inherited Git env error = %v, want target-repo ignore failure", err)
	}
}

func TestGitCommandsPreserveEffectiveIgnoreConfig(t *testing.T) {
	repo := modeTestGitRepo(t)
	ignoreFile := filepath.Join(t.TempDir(), "global-ignore")
	if err := os.WriteFile(ignoreFile, []byte("/.ctx/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.excludesFile")
	t.Setenv("GIT_CONFIG_VALUE_0", ignoreFile)

	err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam})
	if err == nil || !strings.Contains(err.Error(), "ignored by Git") {
		t.Fatalf("team init with injected effective ignore error = %v, want ignore failure", err)
	}
}

func TestDoctorRejectsTrackedPrivateState(t *testing.T) {
	tests := []struct {
		name    string
		mode    Mode
		tracked string
		check   string
	}{
		{name: "team local state", mode: ModeTeam, tracked: ".ctx/local/CONTINUE.md", check: "local state private"},
		{name: "local scaffold", mode: ModeLocal, tracked: ".ctx/README.md", check: "local context untracked"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := modeTestGitRepo(t)
			if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: tt.mode}); err != nil {
				t.Fatal(err)
			}
			modeTestRunGit(t, repo, "add", "-f", "--", tt.tracked)
			checks, err := Doctor(repo, ".ctx")
			if err != nil {
				t.Fatal(err)
			}
			if !hasFailedCheckContaining(checks, tt.check, "tracked") {
				t.Fatalf("Doctor did not reject tracked private state: %+v", checks)
			}
		})
	}
}

func TestLocalModeUsesCommonExcludeInLinkedWorktree(t *testing.T) {
	root := t.TempDir()
	mainRepo := filepath.Join(root, "main")
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(mainRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	modeTestGitInit(t, mainRepo)
	modeTestRunGit(t, mainRepo, "-c", "user.name=ctx-test", "-c", "user.email=ctx@example.invalid", "commit", "--allow-empty", "-qm", "initial")
	modeTestRunGit(t, mainRepo, "worktree", "add", "-qb", "ctx-mode-test", worktree)

	if err := InitWithOptions(worktree, InitOptions{Folder: ".ctx", Mode: ModeLocal}); err != nil {
		t.Fatalf("InitWithOptions(local worktree): %v", err)
	}
	if !modeTestGitIgnored(t, worktree, ".ctx/README.md") {
		t.Error("linked-worktree local scaffold is visible to Git")
	}
	checks, err := Doctor(worktree, ".ctx")
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}
	if anyFailed(checks) {
		t.Fatalf("Doctor reported unhealthy linked-worktree local scaffold: %+v", checks)
	}
}

func TestLocalModeRejectsTeamScaffoldInSiblingWorktree(t *testing.T) {
	root := t.TempDir()
	mainRepo := filepath.Join(root, "main")
	worktree := filepath.Join(root, "worktree")
	if err := os.Mkdir(mainRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	modeTestGitInit(t, mainRepo)
	modeTestRunGit(t, mainRepo, "-c", "user.name=ctx-test", "-c", "user.email=ctx@example.invalid", "commit", "--allow-empty", "-qm", "initial")
	modeTestRunGit(t, mainRepo, "worktree", "add", "-qb", "ctx-mode-conflict-test", worktree)
	if err := InitWithOptions(mainRepo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}

	err := InitWithOptions(worktree, InitOptions{Folder: ".ctx", Mode: ModeLocal})
	if err == nil || !strings.Contains(err.Error(), "sibling worktree") {
		t.Fatalf("local init error = %v, want sibling team-mode conflict", err)
	}
	if _, statErr := os.Lstat(filepath.Join(worktree, ".ctx")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("conflicting local init published a scaffold: %v", statErr)
	}
}

func TestLegacyLocalScaffoldStaysLegacy(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := Init(repo, ".ctx"); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(repo, ".ctx")

	state, err := loadScaffoldState(dest)
	if err != nil {
		t.Fatalf("load legacy state: %v", err)
	}
	if !state.Legacy || state.Config.Mode != ModeLocal || state.continuePath() != "CONTINUE.md" {
		t.Fatalf("legacy state = %+v", state)
	}
	if _, err := Update(repo, ".ctx"); err != nil {
		t.Fatalf("Update legacy scaffold: %v", err)
	}
	for _, path := range []string{"config.json", ".gitignore", "local/CONTINUE.md"} {
		if _, err := os.Lstat(filepath.Join(dest, filepath.FromSlash(path))); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("legacy update created %s: %v", path, err)
		}
	}
	checks, err := Doctor(repo, ".ctx")
	if err != nil {
		t.Fatalf("Doctor legacy scaffold: %v", err)
	}
	if anyFailed(checks) {
		t.Fatalf("Doctor reported healthy legacy scaffold as unhealthy: %+v", checks)
	}
}

func TestMissingConfigInNewLayoutIsNotLegacy(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repo, ".ctx", "config.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "missing config.json") {
		t.Fatalf("Update error = %v, want missing-config failure", err)
	}
}

func TestMissingConfigWithoutLegacyContinuationIsAmbiguous(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeTeam}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(repo, ".ctx")
	if err := os.Remove(filepath.Join(dest, "config.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dest, "local")); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(repo, ".ctx"); err == nil || !strings.Contains(err.Error(), "cannot determine scaffold mode") {
		t.Fatalf("Update error = %v, want ambiguous-layout failure", err)
	}
}

func TestUpdateAndDoctorAcceptSafeExistingCustomPath(t *testing.T) {
	repo := modeTestGitRepo(t)
	if err := InitWithOptions(repo, InitOptions{Folder: ".ctx", Mode: ModeLocal}); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join("docs", "legacy context")
	if err := os.Mkdir(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(repo, ".ctx"), filepath.Join(repo, folder)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".git", "info", "exclude"), []byte(excludePattern(filepath.ToSlash(folder))+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(repo, folder); err != nil {
		t.Fatalf("Update safe existing custom path: %v", err)
	}
	checks, err := Doctor(repo, folder)
	if err != nil {
		t.Fatalf("Doctor safe existing custom path: %v", err)
	}
	if anyFailed(checks) {
		t.Fatalf("Doctor rejected safe existing custom path: %+v", checks)
	}
}

func TestUpdateAndDoctorRejectFolderTraversal(t *testing.T) {
	repo := modeTestGitRepo(t)
	if _, err := Update(repo, "../outside"); err == nil {
		t.Fatal("Update accepted folder traversal")
	}
	if _, err := Doctor(repo, "../outside"); err == nil {
		t.Fatal("Doctor accepted folder traversal")
	}
}

func hasFailedCheckContaining(checks []Check, name, detail string) bool {
	for _, check := range checks {
		if !check.OK && check.Name == name && strings.Contains(check.Detail, detail) {
			return true
		}
	}
	return false
}

func modeTestGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git is required for mode integration tests: %v", err)
	}
	repo := t.TempDir()
	modeTestGitInit(t, repo)
	return repo
}

func modeTestGitInit(t *testing.T, repo string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, output)
	}
}

func modeTestRunGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func modeTestRequireFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("required file %s: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("required path %s is not a regular file", path)
	}
}

func modeTestGitIgnored(t *testing.T, repo, path string) bool {
	t.Helper()
	cmd := exec.Command("git", "check-ignore", "-q", "--", filepath.ToSlash(path))
	cmd.Dir = repo
	err := cmd.Run()
	if err == nil {
		return true
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false
	}
	t.Fatalf("git check-ignore %s: %v", path, err)
	return false
}

func modeTestHasWholeFolderExclude(t *testing.T, repo, folder string) bool {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(repo, ".git", "info", "exclude"))
	if err != nil {
		t.Fatalf("read .git/info/exclude: %v", err)
	}
	wants := map[string]bool{
		folder:             true,
		folder + "/":       true,
		"/" + folder:       true,
		"/" + folder + "/": true,
	}
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if wants[line] {
			return true
		}
	}
	return false
}
