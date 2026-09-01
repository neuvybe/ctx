package ctx

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLayoutCatalogOutputsAndManagedTemplatesAreCoherent(t *testing.T) {
	allAddons := make([]string, 0, len(ListAddons()))
	for _, addon := range ListAddons() {
		allAddons = append(allAddons, addon.ID)
	}
	tests := []struct {
		layout int
		addons []string
	}{
		{layout: LegacyLayoutVersion},
		{layout: CurrentLayoutVersion, addons: allAddons},
	}
	for _, tt := range tests {
		layout, ok := LayoutForVersion(tt.layout)
		if !ok {
			t.Fatalf("layout v%d missing from catalog", tt.layout)
		}
		format, err := managedMarkerFormatFor(layout.MarkerFormat)
		if err != nil {
			t.Fatal(err)
		}
		documents, err := DocumentsForLayout(tt.layout, tt.addons)
		if err != nil {
			t.Fatal(err)
		}
		seenOutputs := make(map[string]bool)
		for _, document := range documents {
			requireSafeCatalogPath(t, document.Path)
			requireSafeCatalogPath(t, document.TemplatePath)
			if seenOutputs[document.Path] {
				t.Fatalf("layout v%d has duplicate output %q", tt.layout, document.Path)
			}
			seenOutputs[document.Path] = true
			template, err := readTemplateAsset(document.TemplatePath)
			if err != nil {
				t.Fatalf("read %s: %v", document.TemplatePath, err)
			}
			parsed, err := parseManagedDocument(string(template), format)
			if err != nil {
				t.Fatalf("parse %s with %s: %v", document.TemplatePath, layout.MarkerFormat, err)
			}
			if document.Managed {
				if len(parsed.blocks) == 0 {
					t.Fatalf("managed document %s has no managed blocks", document.TemplatePath)
				}
				if err := validateManagedDocumentCatalog(document, parsed, format); err != nil {
					t.Fatalf("managed catalog mismatch for %s: %v", document.TemplatePath, err)
				}
			} else if len(parsed.blocks) != 0 {
				t.Fatalf("unmanaged document %s contains managed blocks", document.TemplatePath)
			}
		}

		required, err := RequiredOutputs(tt.layout, tt.addons, false)
		if err != nil {
			t.Fatal(err)
		}
		seenRequired := make(map[string]bool)
		for _, path := range required {
			requireSafeCatalogPath(t, path)
			if seenRequired[path] {
				t.Fatalf("layout v%d has duplicate required output %q", tt.layout, path)
			}
			seenRequired[path] = true
		}
		if got := seenRequired[".ctx-version"]; got != (tt.layout == LegacyLayoutVersion) {
			t.Fatalf("layout v%d .ctx-version requirement = %v", tt.layout, got)
		}
	}
}

func TestLayoutCatalogRejectsLegacyV2Hybrid(t *testing.T) {
	if _, err := RequiredOutputs(CurrentLayoutVersion, nil, true); err == nil {
		t.Fatal("RequiredOutputs accepted a config-less legacy v2 hybrid")
	}
}

func TestAddonCatalogIDsAndPathsAreUnique(t *testing.T) {
	seenIDs := make(map[string]bool)
	seenPaths := make(map[string]bool)
	for _, addon := range ListAddons() {
		if addon.ID == "" || strings.ContainsAny(addon.ID, " ,/") {
			t.Fatalf("invalid add-on ID %q", addon.ID)
		}
		if seenIDs[addon.ID] {
			t.Fatalf("duplicate add-on ID %q", addon.ID)
		}
		if seenPaths[addon.Path] {
			t.Fatalf("duplicate add-on output %q", addon.Path)
		}
		seenIDs[addon.ID] = true
		seenPaths[addon.Path] = true
		requireSafeCatalogPath(t, addon.Path)
	}
}

func requireSafeCatalogPath(t *testing.T, path string) {
	t.Helper()
	native := filepath.FromSlash(path)
	clean := filepath.Clean(native)
	if path == "" || filepath.IsAbs(native) || clean == "." || clean == ".." || clean != native || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		t.Fatalf("unsafe catalog path %q", path)
	}
}
