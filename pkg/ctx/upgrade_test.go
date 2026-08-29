package ctx

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPickAsset(t *testing.T) {
	assets := []Asset{
		{Name: "ctx_0.2.0_darwin_amd64.tar.gz", BrowserDownloadURL: "u1"},
		{Name: "ctx_0.2.0_darwin_arm64.tar.gz", BrowserDownloadURL: "u2"},
		{Name: "ctx_0.2.0_linux_amd64.tar.gz", BrowserDownloadURL: "u3"},
		{Name: "checksums.txt", BrowserDownloadURL: "u4"},
	}
	got, err := pickAsset(assets, "darwin", "arm64")
	if err != nil || got.BrowserDownloadURL != "u2" {
		t.Errorf("darwin/arm64 = %+v, %v, want u2", got, err)
	}
	if _, err := pickAsset(assets, "windows", "arm64"); err == nil {
		t.Errorf("expected error for missing windows/arm64 asset")
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct{ cur, lat string; want int }{
		{"0.1.0", "0.2.0", -1},
		{"0.2.0", "0.1.0", 1},
		{"0.1.0", "0.1.0", 0},
		{"v0.1.0", "0.1.0", 0},
		{"0.1.0-dev", "0.1.0", -1}, // dev < release
		{"0.1.0", "0.1.0-dev", 1},  // release > dev
		{"1.0.0", "0.9.9", 1},
		{"0.9.9", "1.0.0", -1},
		{"1.2.3", "1.2.3", 0},
		{"0.1.0-rc1", "0.1.0-rc2", -1},
		// Non-semver fallback to string compare.
		{"aaa", "zzz", -1},
		{"zzz", "aaa", 1},
		{"same", "same", 0},
	}
	for _, c := range cases {
		got := compareVersions(c.cur, c.lat)
		if got != c.want {
			t.Errorf("compareVersions(%q,%q) = %d, want %d", c.cur, c.lat, got, c.want)
		}
	}
}

func TestDetectInstallMethodFor(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/usr/local/lib/node_modules/@neuvybe/ctx/bin/ctx", "npm"},
		{"/opt/homebrew/Cellar/ctx/0.1.0/bin/ctx", "brew"},
		{"/Users/x/homebrew/bin/ctx", "brew"},
		{"/usr/local/bin/ctx", "direct"},
		{"/opt/myapp/bin/ctx", "direct"},
	}
	for _, c := range cases {
		if got := detectInstallMethodFor(c.path); got != c.want {
			t.Errorf("detectInstallMethodFor(%q) = %q, want %q", c.path, got, c.want)
		}
	}
	// go-install: GOPATH/bin
	t.Setenv("GOPATH", "/Users/x/go")
	if got := detectInstallMethodFor("/Users/x/go/bin/ctx"); got != "go-install" {
		t.Errorf("GOPATH/bin detection = %q, want go-install", got)
	}
}

func TestLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/latest" {
			http.NotFound(w, r)
			return
		}
		body := map[string]any{
			"tag_name": "v0.2.0",
			"assets": []map[string]string{
				{"name": "ctx_0.2.0_darwin_arm64.tar.gz", "browser_download_url": "https://example/asset"},
			},
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()
	rel, err := latestRelease(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("latestRelease: %v", err)
	}
	if rel.TagName != "v0.2.0" {
		t.Errorf("TagName = %q, want v0.2.0", rel.TagName)
	}
	if len(rel.Assets) != 1 || rel.Assets[0].Name != "ctx_0.2.0_darwin_arm64.tar.gz" {
		t.Errorf("Assets = %+v", rel.Assets)
	}
}

// makeTarGz builds a tar.gz in memory with the given file entries (name→content).
func makeTarGz(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range entries {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDownloadAndSwap(t *testing.T) {
	tarball := makeTarGz(t, map[string]string{
		"ctx":          "FAKE-BINARY-CONTENT-0.2.0",
		"checksum.txt": "irrelevant",
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(tarball)
	}))
	defer srv.Close()

	dir := t.TempDir()
	currentBin := filepath.Join(dir, "ctx")
	if err := os.WriteFile(currentBin, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := downloadAndSwap(context.Background(), srv.URL, currentBin); err != nil {
		t.Fatalf("downloadAndSwap: %v", err)
	}
	got, err := os.ReadFile(currentBin)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "FAKE-BINARY-CONTENT-0.2.0" {
		t.Errorf("after swap, binary = %q, want FAKE-BINARY-CONTENT-0.2.0", got)
	}
	// temp file cleaned up (renamed away, not left as .ctx-*.new)
	matches, _ := filepath.Glob(filepath.Join(dir, ".ctx-*.new"))
	if len(matches) != 0 {
		t.Errorf("leftover temp files: %v", matches)
	}
}

func TestDownloadAndSwapMissingEntry(t *testing.T) {
	tarball := makeTarGz(t, map[string]string{"other": "x"}) // no "ctx" entry
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(tarball)
	}))
	defer srv.Close()
	dir := t.TempDir()
	currentBin := filepath.Join(dir, "ctx")
	os.WriteFile(currentBin, []byte("OLD"), 0o755)
	if err := downloadAndSwap(context.Background(), srv.URL, currentBin); err == nil {
		t.Errorf("expected error when archive has no ctx entry")
	}
	// original preserved on failure
	got, _ := os.ReadFile(currentBin)
	if string(got) != "OLD" {
		t.Errorf("original binary should be preserved on failure, got %q", got)
	}
}

func TestUpgradeDirectAlreadyUpToDate(t *testing.T) {
	// Server reports a release equal to the current dev version's base -> "already up to date"
	// (0.1.0-dev < 0.1.0 would upgrade; so report an older release to force "up to date").
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v0.0.1",
			"assets":   []map[string]string{},
		})
	}))
	defer srv.Close()
	// Force detectInstallMethod to "direct" by pointing GOPATH elsewhere and running from temp.
	t.Setenv("GOPATH", "/nonexistent-gopath")
	res, err := Upgrade(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	// Current is 0.1.0-dev; latest 0.0.1 -> compareVersions(0.1.0-dev, 0.0.1): majors 0==0, minors 1>0 -> 1 (current newer) -> "already up to date".
	if res.Updated {
		t.Errorf("should not update when current is newer")
	}
	if res.Message == "" {
		t.Errorf("expected an up-to-date message")
	}
}