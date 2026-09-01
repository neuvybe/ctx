package ctx

// Version is the ctx CLI version. Override at build time with
//
//	-ldflags "-X github.com/neuvybe/ctx/pkg/ctx.Version=0.1.0"
//
// so release builds stamp a real version; the default marks a dev build.
var Version = "0.3.0"
