// Package version carries the build's release version.
//
// The value is stamped at build time by the release pipeline
// (-ldflags "-X .../internal/version.Version=0.1.0"); a plain `go build` or
// `go run` leaves it at DevVersion, which is what a development checkout
// should report. Everything the product ships — backend, frontend, extension,
// plugin — carries one version per release (see ADR-0036's sibling decision in
// the release-automation issue), so this string identifies the whole app, not
// just the server binary.
package version

// DevVersion is what an unstamped build reports.
const DevVersion = "dev"

// Version is the release this binary was built from, or DevVersion.
var Version = DevVersion

// IsRelease reports whether this build was stamped by the release pipeline.
func IsRelease() bool { return Version != DevVersion && Version != "" }
