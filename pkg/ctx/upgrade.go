package ctx

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// DefaultReleasesAPI is the GitHub Releases API base for the ctx repo.
const DefaultReleasesAPI = "https://api.github.com/repos/neuvybe/ctx"

// Asset is a release asset (subset of the GitHub API fields).
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Release is a subset of the GitHub releases/latest response.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// latestRelease fetches <base>/releases/latest and decodes it.
func latestRelease(ctx context.Context, base string) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/releases/latest", nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("releases API returned %d", resp.StatusCode)
	}
	var r Release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return Release{}, err
	}
	return r, nil
}

// pickAsset chooses the asset matching goos/arch by goreleaser's naming
// convention: ctx_<version>_<goos>_<arch>.tar.gz (it matches on "_<goos>_<arch>.").
func pickAsset(assets []Asset, goos, arch string) (Asset, error) {
	want := fmt.Sprintf("_%s_%s.", goos, arch)
	for _, a := range assets {
		if strings.Contains(a.Name, want) {
			return a, nil
		}
	}
	return Asset{}, fmt.Errorf("no release asset for %s/%s (looked for %q in %d assets)", goos, arch, want, len(assets))
}

// parseSemver parses a "vX.Y.Z" or "X.Y.Z" (optionally with "-pre") into parts.
func parseSemver(v string) (major, minor, patch int, pre string, ok bool) {
	v = strings.TrimPrefix(v, "v")
	main, pre := v, ""
	if i := strings.Index(v, "-"); i >= 0 {
		main, pre = v[:i], v[i+1:]
	}
	parts := strings.Split(main, ".")
	if len(parts) != 3 {
		return 0, 0, 0, "", false
	}
	var nums [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return 0, 0, 0, "", false
		}
		nums[i] = n
	}
	return nums[0], nums[1], nums[2], pre, true
}

// compareVersions returns -1 if current < latest, 0 if equal, 1 if current > latest.
// A pre-release (e.g. "0.1.0-dev") is considered older than the same version
// without a pre-release (e.g. "0.1.0").
func compareVersions(current, latest string) int {
	cM, cm, cp, cPre, cok := parseSemver(current)
	lM, lm, lp, lPre, lok := parseSemver(latest)
	if !cok || !lok {
		// Fallback to string compare for non-semver strings.
		switch {
		case current == latest:
			return 0
		case current < latest:
			return -1
		default:
			return 1
		}
	}
	if cM != lM {
		if cM < lM {
			return -1
		}
		return 1
	}
	if cm != lm {
		if cm < lm {
			return -1
		}
		return 1
	}
	if cp != lp {
		if cp < lp {
			return -1
		}
		return 1
	}
	// Pre-release: a release (no pre) is newer than a pre-release.
	if cPre == "" && lPre != "" {
		return 1
	}
	if cPre != "" && lPre == "" {
		return -1
	}
	if cPre != lPre {
		if cPre < lPre {
			return -1
		}
		return 1
	}
	return 0
}

// detectInstallMethodFor classifies how the binary at exePath was installed.
func detectInstallMethodFor(exePath string) string {
	exe := filepath.Clean(exePath)
	switch {
	case strings.Contains(exe, "node_modules"):
		return "npm"
	case strings.Contains(exe, "Cellar"), strings.Contains(exe, "/homebrew/"):
		return "brew"
	case isUnderGoBin(exe):
		return "go-install"
	default:
		return "direct"
	}
}

// detectInstallMethod classifies how the running binary was installed.
func detectInstallMethod() string {
	exe, err := os.Executable()
	if err != nil {
		return "direct"
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return detectInstallMethodFor(exe)
}

func isUnderGoBin(exe string) bool {
	gp := os.Getenv("GOPATH")
	if gp == "" {
		return false
	}
	gb := filepath.Join(gp, "bin") + string(filepath.Separator)
	return strings.HasPrefix(exe, gb)
}

// extractBinaryFromTarGz extracts the entry named `name` from a tar.gz stream
// (goreleaser archives the binary as `ctx`) and writes it to w.
func extractBinaryFromTarGz(r io.Reader, w io.Writer, name string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) == name && hdr.Typeflag == tar.TypeReg {
			_, err := io.Copy(w, tr)
			return err
		}
	}
	return fmt.Errorf("entry %q not found in archive", name)
}

// downloadAndSwap downloads the asset at assetURL (a tar.gz containing a `ctx`
// binary), extracts it to a temp file next to currentBin, and atomically
// replaces currentBin via os.Rename. The running process keeps the old inode.
func downloadAndSwap(ctx context.Context, assetURL, currentBin string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("asset download returned %d", resp.StatusCode)
	}
	dir := filepath.Dir(currentBin)
	tmp, err := os.CreateTemp(dir, ".ctx-*.new")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op on success (renamed away)
	if err := extractBinaryFromTarGz(resp.Body, tmp, "ctx"); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}
	return os.Rename(tmpPath, currentBin)
}

// UpgradeResult describes the outcome of an upgrade attempt.
type UpgradeResult struct {
	Method  string // npm / brew / go-install / direct
	Updated bool   // true if the binary was replaced
	Message string // human-readable result
}

// Upgrade fetches the latest release from base and upgrades the CLI.
// For package-manager installs (npm/brew/go-install) it returns a hint instead
// of self-replacing (those managers own the binary). For a direct binary it
// self-replaces from the matching OS/arch release asset.
func Upgrade(ctx context.Context, base string) (UpgradeResult, error) {
	method := detectInstallMethod()
	switch method {
	case "npm":
		return UpgradeResult{Method: method, Message: "installed via npm; run `npm i -g @neuvybe/ctx@latest`"}, nil
	case "brew":
		return UpgradeResult{Method: method, Message: "installed via brew; run `brew upgrade ctx`"}, nil
	case "go-install":
		return UpgradeResult{Method: method, Message: "installed via go install; run `go install github.com/neuvybe/ctx/cmd/ctx@latest`"}, nil
	}

	rel, err := latestRelease(ctx, base)
	if err != nil {
		return UpgradeResult{Method: method}, fmt.Errorf("fetch latest release: %w", err)
	}
	latest := strings.TrimPrefix(rel.TagName, "v")
	if c := compareVersions(Version, latest); c >= 0 {
		return UpgradeResult{Method: method, Message: fmt.Sprintf("already up to date (v%s)", Version)}, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return UpgradeResult{Method: method}, err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	asset, err := pickAsset(rel.Assets, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return UpgradeResult{Method: method}, err
	}
	if err := downloadAndSwap(ctx, asset.BrowserDownloadURL, exe); err != nil {
		return UpgradeResult{Method: method}, fmt.Errorf("download & swap: %w", err)
	}
	return UpgradeResult{Method: method, Updated: true, Message: fmt.Sprintf("upgraded v%s → v%s", Version, latest)}, nil
}