package ctx

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Check is one doctor finding.
type Check struct {
	Name   string
	OK     bool
	Detail string
}

func pass(name, detail string) Check  { return Check{Name: name, OK: true, Detail: detail} }
func fail(name, detail string) Check  { return Check{Name: name, OK: false, Detail: detail} }

// expectedFiles are the files ctx init must have produced.
var expectedFiles = []string{
	"README.md", "OPERATING.md", "CONTINUE.md", "INDEX.md", "REVIEW.md",
	"context/overview.md", "context/architecture.md", "context/format.md",
	"context/extending.md", "context/known-issues.md", "context/glossary.md",
}

// Doctor validates a .ctx/ folder: existence, version stamp, exclude entry, no
// leftover init-substituted placeholders, balanced managed markers, expected
// files present. Returns the checks; the command exits non-zero if any fail.
func Doctor(repo, folder string) ([]Check, error) {
	dest := filepath.Join(repo, folder)
	var checks []Check

	// 1. folder exists
	if info, err := os.Stat(dest); err != nil || !info.IsDir() {
		return []Check{fail(".ctx/ folder exists", fmt.Sprintf("no %s; run `ctx init` first", dest))}, nil
	}
	checks = append(checks, pass(".ctx/ folder exists", dest))

	// 2. .ctx-version stamp
	if v, err := os.ReadFile(filepath.Join(dest, ".ctx-version")); err == nil {
		checks = append(checks, pass(".ctx-version stamp", strings.TrimSpace(string(v))))
	} else {
		checks = append(checks, fail(".ctx-version stamp", "missing"))
	}

	// 3. .git/info/exclude has the folder
	gd, err := resolveGitDir(repo)
	if err != nil {
		checks = append(checks, fail(".git/info/exclude entry", err.Error()))
	} else {
		exc, err := os.ReadFile(filepath.Join(gd, "info", "exclude"))
		if err != nil {
			checks = append(checks, fail(".git/info/exclude entry", "no exclude file"))
		} else if !containsExcludeLine(string(exc), folder) {
			checks = append(checks, fail(".git/info/exclude entry", fmt.Sprintf("%s not excluded (would leak into git)", folder)))
		} else {
			checks = append(checks, pass(".git/info/exclude entry", folder+"/"))
		}
	}

	// 4. no leftover init-substituted placeholders ({{PROJECT}}/{{DATE}}) in any .md
	leftover := []string{}
	_ = fs.WalkDir(os.DirFS(dest), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		b, rerr := os.ReadFile(filepath.Join(dest, path))
		if rerr != nil {
			return nil
		}
		if strings.Contains(string(b), "{{PROJECT}}") || strings.Contains(string(b), "{{DATE}}") {
			leftover = append(leftover, path)
		}
		return nil
	})
	if len(leftover) == 0 {
		checks = append(checks, pass("no {{PROJECT}}/{{DATE}} leftovers", ""))
	} else {
		checks = append(checks, fail("no {{PROJECT}}/{{DATE}} leftovers", strings.Join(leftover, ", ")))
	}

	// 5. managed markers balanced in managed files
	for _, name := range managedFiles {
		b, err := os.ReadFile(filepath.Join(dest, name))
		if err != nil {
			continue // missing managed file is reported by the expected-files check
		}
		if !markersBalanced(string(b)) {
			checks = append(checks, fail("markers balanced in "+name, "unbalanced <!-- ctx:managed --> begin/end"))
		} else {
			checks = append(checks, pass("markers balanced in "+name, ""))
		}
	}

	// 6. expected files present
	missing := []string{}
	for _, name := range expectedFiles {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		checks = append(checks, pass("expected files present", fmt.Sprintf("%d files", len(expectedFiles))))
	} else {
		checks = append(checks, fail("expected files present", "missing: "+strings.Join(missing, ", ")))
	}

	return checks, nil
}

func containsExcludeLine(exclude, folder string) bool {
	for _, line := range strings.Split(exclude, "\n") {
		if strings.TrimRight(strings.TrimSpace(line), "/") == folder {
			return true
		}
	}
	return false
}