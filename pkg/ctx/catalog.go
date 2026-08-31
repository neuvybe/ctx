package ctx

import (
	"fmt"
	"sort"
	"strings"
)

const (
	LegacyLayoutVersion     = 1
	CurrentLayoutVersion    = 2
	CurrentTemplateRevision = "2.0.0"
)

type MarkerFormat string

const (
	MarkerFormatUnnamedV1 MarkerFormat = "unnamed-v1"
	MarkerFormatNamedV2   MarkerFormat = "named-v2"
)

// LayoutSpec defines a persisted scaffold layout independently of the CLI
// release version. TemplateRevision advances only when that layout's managed
// template content changes.
type LayoutSpec struct {
	Version          int
	TemplateRevision string
	MarkerFormat     MarkerFormat
}

// DocumentSpec is one output in a layout or selected add-on.
type DocumentSpec struct {
	Path            string
	TemplatePath    string
	ProjectFact     bool
	Local           bool
	Managed         bool
	ManagedMarkerID string
}

// Addon describes one optional v2 document set.
type Addon struct {
	ID          string
	Path        string
	Description string
	RouteNeed   string
	RoutePath   string
	Default     bool
}

var layouts = map[int]LayoutSpec{
	1: {Version: 1, TemplateRevision: "v1-frozen", MarkerFormat: MarkerFormatUnnamedV1},
	2: {Version: 2, TemplateRevision: CurrentTemplateRevision, MarkerFormat: MarkerFormatNamedV2},
}

var v1Documents = []DocumentSpec{
	{Path: "README.md", TemplatePath: "v1/README.md", Managed: true},
	{Path: "OPERATING.md", TemplatePath: "v1/OPERATING.md"},
	{Path: "INDEX.md", TemplatePath: "v1/INDEX.md"},
	{Path: "REVIEW.md", TemplatePath: "v1/REVIEW.md", Managed: true},
	{Path: "context/overview.md", TemplatePath: "v1/context/overview.md", ProjectFact: true},
	{Path: "context/architecture.md", TemplatePath: "v1/context/architecture.md", ProjectFact: true},
	{Path: "context/format.md", TemplatePath: "v1/context/format.md", ProjectFact: true},
	{Path: "context/extending.md", TemplatePath: "v1/context/extending.md", ProjectFact: true},
	{Path: "context/known-issues.md", TemplatePath: "v1/context/known-issues.md", ProjectFact: true},
	{Path: "context/glossary.md", TemplatePath: "v1/context/glossary.md", ProjectFact: true},
	{Path: "local/CONTINUE.md", TemplatePath: "v1/local/CONTINUE.md", Local: true},
}

var v2CoreDocuments = []DocumentSpec{
	{Path: "README.md", TemplatePath: "v2/README.md", Managed: true, ManagedMarkerID: "readme-platform"},
	{Path: "INDEX.md", TemplatePath: "v2/INDEX.md", Managed: true, ManagedMarkerID: "index-routing"},
	{Path: "context/overview.md", TemplatePath: "v2/context/overview.md", ProjectFact: true},
	{Path: "context/architecture.md", TemplatePath: "v2/context/architecture.md", ProjectFact: true},
	{Path: "context/caveats.md", TemplatePath: "v2/context/caveats.md", ProjectFact: true},
	{Path: "local/CONTINUE.md", TemplatePath: "v2/local/CONTINUE.md", Local: true},
}

type addonSpec struct {
	Addon
	Document DocumentSpec
}

var addonCatalog = []addonSpec{
	{Addon: Addon{ID: "operating", Path: "OPERATING.md", Description: "owner-ratified working agreement", RouteNeed: "Project-specific operating policy", RoutePath: "OPERATING.md"}, Document: DocumentSpec{Path: "OPERATING.md", TemplatePath: "addons/operating/OPERATING.md"}},
	{Addon: Addon{ID: "contracts", Path: "context/contracts.md", Description: "representation and compatibility boundaries", RouteNeed: "Data/API/storage contracts or compatibility changes", RoutePath: "context/contracts.md"}, Document: DocumentSpec{Path: "context/contracts.md", TemplatePath: "addons/contracts/context/contracts.md", ProjectFact: true}},
	{Addon: Addon{ID: "extending", Path: "context/extending.md", Description: "supported extension points", RouteNeed: "A supported extension point or new capability", RoutePath: "context/extending.md"}, Document: DocumentSpec{Path: "context/extending.md", TemplatePath: "addons/extending/context/extending.md", ProjectFact: true}},
	{Addon: Addon{ID: "glossary", Path: "context/glossary.md", Description: "project-specific terminology", RouteNeed: "A project-specific or ambiguous term", RoutePath: "context/glossary.md", Default: true}, Document: DocumentSpec{Path: "context/glossary.md", TemplatePath: "addons/glossary/context/glossary.md", ProjectFact: true}},
	{Addon: Addon{ID: "review", Path: "workflows/review.md", Description: "shared review workflow", RouteNeed: "The project's review procedure", RoutePath: "workflows/review.md"}, Document: DocumentSpec{Path: "workflows/review.md", TemplatePath: "addons/review/workflows/review.md", Managed: true, ManagedMarkerID: "review-workflow"}},
}

func LayoutForVersion(version int) (LayoutSpec, bool) {
	layout, ok := layouts[version]
	return layout, ok
}

func ListAddons() []Addon {
	addons := make([]Addon, 0, len(addonCatalog))
	for _, spec := range addonCatalog {
		addons = append(addons, spec.Addon)
	}
	return addons
}

// DefaultAddonIDs returns the add-ons selected for a new layout-v2 scaffold.
// The returned slice is independent and may be modified by the caller.
func DefaultAddonIDs() []string {
	var defaults []string
	for _, spec := range addonCatalog {
		if spec.Default {
			defaults = append(defaults, spec.ID)
		}
	}
	return defaults
}

func LookupAddon(id string) (Addon, bool) {
	for _, spec := range addonCatalog {
		if spec.ID == id {
			return spec.Addon, true
		}
	}
	return Addon{}, false
}

func DocumentsForLayout(layoutVersion int, addons []string) ([]DocumentSpec, error) {
	switch layoutVersion {
	case 1:
		if len(addons) > 0 {
			return nil, fmt.Errorf("layout v1 does not support add-ons")
		}
		return append([]DocumentSpec(nil), v1Documents...), nil
	case 2:
		selected, err := normalizeAddonSelection(addons)
		if err != nil {
			return nil, err
		}
		documents := append([]DocumentSpec(nil), v2CoreDocuments...)
		for _, spec := range addonCatalog {
			if selected[spec.ID] {
				documents = append(documents, spec.Document)
			}
		}
		return documents, nil
	default:
		return nil, fmt.Errorf("unsupported layoutVersion %d", layoutVersion)
	}
}

func RequiredOutputs(layoutVersion int, addons []string, legacy bool) ([]string, error) {
	if legacy && layoutVersion != LegacyLayoutVersion {
		return nil, fmt.Errorf("legacy scaffold cannot use layoutVersion %d", layoutVersion)
	}
	documents, err := DocumentsForLayout(layoutVersion, addons)
	if err != nil {
		return nil, err
	}
	outputs := make([]string, 0, len(documents)+3)
	for _, document := range documents {
		path := document.Path
		if legacy && document.Local {
			path = "CONTINUE.md"
		}
		outputs = append(outputs, path)
	}
	if !legacy {
		outputs = append(outputs, ".gitignore", configFileName)
	}
	if layoutVersion == 1 {
		outputs = append(outputs, ".ctx-version")
	}
	return outputs, nil
}

func ProjectFactDocuments(layoutVersion int, addons []string) ([]DocumentSpec, error) {
	documents, err := DocumentsForLayout(layoutVersion, addons)
	if err != nil {
		return nil, err
	}
	var facts []DocumentSpec
	for _, document := range documents {
		if document.ProjectFact {
			facts = append(facts, document)
		}
	}
	return facts, nil
}

func ManagedDocuments(layoutVersion int, addons []string) ([]DocumentSpec, error) {
	documents, err := DocumentsForLayout(layoutVersion, addons)
	if err != nil {
		return nil, err
	}
	var managed []DocumentSpec
	for _, document := range documents {
		if document.Managed {
			managed = append(managed, document)
		}
	}
	return managed, nil
}

func normalizeAddonNames(addons []string) ([]string, error) {
	seen := make(map[string]bool)
	var names []string
	for _, value := range addons {
		for _, item := range strings.Split(value, ",") {
			id := strings.TrimSpace(item)
			if id == "" {
				continue
			}
			if _, ok := LookupAddon(id); !ok {
				valid := make([]string, 0, len(addonCatalog))
				for _, spec := range addonCatalog {
					valid = append(valid, spec.ID)
				}
				sort.Strings(valid)
				return nil, fmt.Errorf("unknown add-on %q (available: %s)", id, strings.Join(valid, ", "))
			}
			if !seen[id] {
				seen[id] = true
				names = append(names, id)
			}
		}
	}
	sort.Strings(names)
	return names, nil
}

func normalizeAddonSelection(addons []string) (map[string]bool, error) {
	names, err := normalizeAddonNames(addons)
	if err != nil {
		return nil, err
	}
	selected := make(map[string]bool, len(names))
	for _, id := range names {
		selected[id] = true
	}
	return selected, nil
}

func addonRoutesMarkdown(addons []string) (string, error) {
	names, err := normalizeAddonNames(addons)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "No optional add-ons installed. Run `ctx add --list` to inspect the catalog.", nil
	}
	var out strings.Builder
	out.WriteString("| Need | Read |\n|---|---|\n")
	for _, id := range names {
		addon, _ := LookupAddon(id)
		fmt.Fprintf(&out, "| %s | `%s` |\n", addon.RouteNeed, addon.RoutePath)
	}
	return strings.TrimSuffix(out.String(), "\n"), nil
}
