package ctx

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed all:template
var templateFS embed.FS

func templateBundleFS() fs.FS {
	sub, err := fs.Sub(templateFS, "template")
	if err != nil {
		// template/ is compiled in and the name is fixed; a failure here is a build bug.
		panic(err)
	}
	return sub
}

// TemplateFS preserves the original exported API and returns the frozen v1
// template tree with paths such as README.md and context/overview.md.
//
// Deprecated: use TemplateFSForLayout for an explicit layout version.
func TemplateFS() fs.FS {
	sub, err := TemplateFSForLayout(LegacyLayoutVersion)
	if err != nil {
		panic(err)
	}
	return sub
}

// TemplateFSForLayout returns the core template tree for a supported layout.
// Optional add-ons are exposed separately through AddonTemplateFSForLayout.
func TemplateFSForLayout(layoutVersion int) (fs.FS, error) {
	root := ""
	switch layoutVersion {
	case LegacyLayoutVersion:
		root = "v1"
	case FlatLayoutVersion:
		root = "v2"
	case CurrentLayoutVersion:
		root = "v3"
	default:
		return nil, fs.ErrInvalid
	}
	return fs.Sub(templateBundleFS(), root)
}

// AddonTemplateFS preserves the original flat layout-v2 add-on template API.
//
// Deprecated: use AddonTemplateFSForLayout with an explicit layout version.
func AddonTemplateFS(id string) (fs.FS, error) {
	return AddonTemplateFSForLayout(FlatLayoutVersion, id)
}

// AddonTemplateFSForLayout returns the template tree for a layout's add-on.
func AddonTemplateFSForLayout(layoutVersion int, id string) (fs.FS, error) {
	if layoutVersion != FlatLayoutVersion && layoutVersion != CurrentLayoutVersion {
		return nil, fmt.Errorf("layout v%d does not support add-ons", layoutVersion)
	}
	if _, ok := LookupAddon(id); !ok {
		return nil, fmt.Errorf("unknown add-on %q", id)
	}
	addon, _ := lookupAddonForLayout(layoutVersion, id)
	if layoutVersion == CurrentLayoutVersion && strings.HasPrefix(addon.Path, "context/") {
		return fs.Sub(templateBundleFS(), "addons-v3/"+id)
	}
	return fs.Sub(templateBundleFS(), "addons/"+id)
}

func templateFSForLayout(layoutVersion int) (fs.FS, error) {
	return TemplateFSForLayout(layoutVersion)
}

func readTemplateAsset(path string) ([]byte, error) {
	return fs.ReadFile(templateBundleFS(), path)
}
