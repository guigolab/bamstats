package bamstats

import "fmt"

// Constants for Major and Minor version numbers.
const (
	VersionNumber      = 0.3
	MinorVersionNumber = 0
)

// PreVersionString indicates wheather the program is a pre-release.
var PreVersionString = "-dev"

// GitCommit represents the git commit of the build
var GitCommit = ""

// Version returns the current version string.
func Version() string {
	return getVersion(VersionNumber, MinorVersionNumber, PreVersionString, GitCommit)
}

func getVersion(version float32, minor uint8, pre string, rev string) string {
	if rev != "" {
		rev = fmt.Sprintf("-%s", rev)
	}
	return fmt.Sprintf("%.2g.%d%s%s", version, minor, pre, rev)
}
