package bamstats

import "fmt"

// Constants for Major and Minor version numbers.
const (
	VersionNumber      = 0.2
	MinorVersionNumber = 0
)

// PreVersionString indicates wheather the program is a pre-release.
var PreVersionString = ""

// Version returns the current version string.
func Version() string {
	return getVersion(VersionNumber, MinorVersionNumber, PreVersionString)
}

func getVersion(version float32, minor uint8, pre string) string {
	return fmt.Sprintf("%.2g.%d%s", version, minor, pre)
}
