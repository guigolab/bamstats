package bamstats

import "fmt"

const (
	VersionNumber      = 0.1
	MinorVersionNumber = 1
)

var PreVersionString = "-dev"

func Version() string {
	return getVersion(VersionNumber, MinorVersionNumber, PreVersionString)
}

func getVersion(version float32, minor uint8, pre string) string {
	return fmt.Sprintf("%.2g.%d%s", version, minor, pre)
}
