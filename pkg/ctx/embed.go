package ctx

import (
	"embed"
	"fmt"
	"io/fs"
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
// Optional v2 add-ons are exposed separately through AddonTemplateFS.
func TemplateFSForLayout(layoutVersion int) (fs.FS, error) {
	root := ""
	switch layoutVersion {
	case LegacyLayoutVersion:
		root = "v1"
	case CurrentLayoutVersion:
		root = "v2"
	default:
		return nil, fs.ErrInvalid
	}
	return fs.Sub(templateBundleFS(), root)
}

// AddonTemplateFS returns one optional add-on's template tree with its add-on
// prefix stripped. The returned paths match the files installed by ctx add.
func AddonTemplateFS(id string) (fs.FS, error) {
	if _, ok := LookupAddon(id); !ok {
		return nil, fmt.Errorf("unknown add-on %q", id)
	}
	return fs.Sub(templateBundleFS(), "addons/"+id)
}

func templateFSForLayout(layoutVersion int) (fs.FS, error) {
	return TemplateFSForLayout(layoutVersion)
}

func readTemplateAsset(path string) ([]byte, error) {
	return fs.ReadFile(templateBundleFS(), path)
}
