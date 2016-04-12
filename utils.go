package bamstats

import (
	"encoding/json"
	"os"

	log "github.com/Sirupsen/logrus"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func max(a, b uint32) uint32 {
	if a < b {
		return b
	}
	return a
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func OutputJson(stats interface{}) {
	b, err := json.MarshalIndent(stats, "", "\t")
	check(err)
	os.Stdout.Write(b)
}
