// Package version holds the Swiflow release string (overridable via -ldflags).
package version

// Version is the application version shown in About / health.
// Override at link time: -X github.com/OptLTD/swiflow/internal/version.Version=x.y.z
var Version = "0.1.0"
