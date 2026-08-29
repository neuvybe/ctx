package ctx

import (
	"embed"
	"io/fs"
)

//go:embed all:template
var templateFS embed.FS

// TemplateFS returns the embedded template/ directory as a filesystem with
// the "template/" prefix stripped (so paths are "README.md", "context/overview.md", …).
func TemplateFS() fs.FS {
	sub, err := fs.Sub(templateFS, "template")
	if err != nil {
		// template/ is compiled in and the name is fixed; a failure here is a build bug.
		panic(err)
	}
	return sub
}