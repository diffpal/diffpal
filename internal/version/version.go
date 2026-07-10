// Package version exposes build-time DiffPal version metadata.
package version

import "fmt"

var (
	Version   = "0.0.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func String() string {
	return fmt.Sprintf("%s+%s (%s)", Version, GitCommit, BuildDate)
}
