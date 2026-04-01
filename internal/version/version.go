package version

import "fmt"

var (
	Version   = "0.1.0-dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func Full() string {
	return fmt.Sprintf("ti-code %s (%s, %s)", Version, GitCommit, BuildDate)
}

func Short() string {
	return Version
}
