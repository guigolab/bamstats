package bamstats

import "fmt"

const (
	VersionNumber      = 0.1
	MinorVersionNumber = 1
)

func Version() string {
	return getVersion(VersionNumber, MinorVersionNumber)
}

func getVersion(version float32, minor uint8) string {
	return fmt.Sprintf("%.2g.%d", version, minor)
}
