package ctx

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestParseConfigNormalizesSchemaOneAsLayoutOne(t *testing.T) {
	cfg, err := parseConfig([]byte("{\"schemaVersion\":1,\"mode\":\"team\"}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SchemaVersion != 1 || cfg.LayoutVersion != LegacyLayoutVersion || cfg.Mode != ModeTeam {
		t.Fatalf("config = %+v", cfg)
	}
	if cfg.TemplateRevision != layouts[LegacyLayoutVersion].TemplateRevision {
		t.Fatalf("template revision = %q", cfg.TemplateRevision)
	}
	data, err := marshalConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{\n  \"schemaVersion\": 1,\n  \"mode\": \"team\"\n}\n" {
		t.Fatalf("schema-one encoding changed:\n%s", data)
	}
}

func TestConfigSchemaTwoRoundTripCanonicalizesAddons(t *testing.T) {
	want := Config{
		SchemaVersion:    currentSchemaVersion,
		LayoutVersion:    CurrentLayoutVersion,
		TemplateRevision: CurrentTemplateRevision,
		Project:          "example",
		Mode:             ModeLocal,
		Addons:           []string{"contracts", "glossary"},
	}
	input := []byte(`{
  "schemaVersion": 2,
  "layoutVersion": 2,
  "templateRevision": "2.0.0",
  "project": "example",
  "mode": "local",
  "addons": ["glossary", "contracts", "glossary"]
}`)
	cfg, err := parseConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("config = %+v, want %+v", cfg, want)
	}
	data, err := marshalConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip Config
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(roundTrip, want) {
		t.Fatalf("round trip = %+v, want %+v", roundTrip, want)
	}
}

func TestParseConfigRejectsMalformedOrUnsupportedDescriptors(t *testing.T) {
	tests := map[string]string{
		"unknown schema":        `{"schemaVersion":3,"mode":"team"}`,
		"missing layout":        `{"schemaVersion":2,"templateRevision":"2.0.0","project":"p","mode":"team"}`,
		"missing revision":      `{"schemaVersion":2,"layoutVersion":2,"project":"p","mode":"team"}`,
		"missing project":       `{"schemaVersion":2,"layoutVersion":2,"templateRevision":"2.0.0","mode":"team"}`,
		"invalid mode":          `{"schemaVersion":2,"layoutVersion":2,"templateRevision":"2.0.0","project":"p","mode":"secret"}`,
		"unknown addon":         `{"schemaVersion":2,"layoutVersion":2,"templateRevision":"2.0.0","project":"p","mode":"team","addons":["mystery"]}`,
		"schema one extra data": `{"schemaVersion":1,"mode":"team","project":"p"}`,
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseConfig([]byte(input)); err == nil {
				t.Fatal("parseConfig accepted invalid descriptor")
			}
		})
	}
}

func TestCatalogCoreAndAddonSelection(t *testing.T) {
	documents, err := DocumentsForLayout(CurrentLayoutVersion, []string{"glossary", "review"})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, document := range documents {
		paths = append(paths, document.Path)
	}
	want := []string{
		"README.md", "INDEX.md", "context/overview.md", "context/architecture.md",
		"context/caveats.md", "local/CONTINUE.md", "context/glossary.md", "workflows/review.md",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("document paths = %v, want %v", paths, want)
	}
	routes, err := addonRoutesMarkdown([]string{"review", "glossary"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(routes, "context/glossary.md") || !strings.Contains(routes, "workflows/review.md") {
		t.Fatalf("routes missing selected add-ons:\n%s", routes)
	}
	if strings.Contains(routes, "OPERATING.md") {
		t.Fatalf("routes included unselected add-on:\n%s", routes)
	}
}

func TestDefaultAddonIDs(t *testing.T) {
	defaults := DefaultAddonIDs()
	if !reflect.DeepEqual(defaults, []string{"glossary"}) {
		t.Fatalf("default add-ons = %v, want [glossary]", defaults)
	}
	var marked []string
	for _, addon := range ListAddons() {
		if addon.Default {
			marked = append(marked, addon.ID)
		}
	}
	if !reflect.DeepEqual(marked, defaults) {
		t.Fatalf("catalog default metadata = %v, want %v", marked, defaults)
	}
	defaults[0] = "contracts"
	if got := DefaultAddonIDs(); !reflect.DeepEqual(got, []string{"glossary"}) {
		t.Fatalf("caller mutation changed catalog defaults: %v", got)
	}
}

func TestNormalizeInitOptionsDistinguishesDefaultAndCoreOnlyAddons(t *testing.T) {
	defaults, err := normalizeInitOptions(InitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(defaults.Addons, []string{"glossary"}) {
		t.Fatalf("nil add-ons normalized to %v, want [glossary]", defaults.Addons)
	}

	coreOnly, err := normalizeInitOptions(InitOptions{Addons: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(coreOnly.Addons) != 0 {
		t.Fatalf("explicit empty add-ons normalized to %v, want none", coreOnly.Addons)
	}
}

func TestCatalogRejectsV2LegacyHybrid(t *testing.T) {
	if _, err := RequiredOutputs(CurrentLayoutVersion, nil, true); err == nil {
		t.Fatal("v2 legacy hybrid was accepted")
	}
}
