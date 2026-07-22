// NOMVGEN
package main

import "runtime/debug"

// version is injected at release build time via the linker, e.g.:
//
//	go build -ldflags "-X main.version=v0.5.2"
//
// It is empty for plain `go build`/`go run` and for `go install` without ldflags.
var version = ""

// MVEPVersion resolves the toolkit CLI version, in order of preference:
//  1. the linker-injected `version` (release binaries),
//  2. the module version from build info (`go install …@vX.Y.Z`),
//  3. "dev" for local builds.
func MVEPVersion() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}
