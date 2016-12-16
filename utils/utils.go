package utils

import (
	"bufio"
	"encoding/json"
	"io"
	"os"

	log "github.com/Sirupsen/logrus"
)

func Check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// OutputJSON writes the json representation of stats to an io.Writer
func OutputJSON(writer io.Writer, stats interface{}) {
	b, err := json.MarshalIndent(stats, "", "\t")
	Check(err)
	writer.Write(b)
	if w, ok := writer.(*bufio.Writer); ok {
		w.Flush()
	}
}

// NewOutput return a new io.Writer given an output file name. If the file name is '-' os.Stdout is returned.
func NewOutput(output string) io.Writer {
	switch output {
	case "-":
		return os.Stdout
	default:
		f, err := os.Create(output)
		Check(err)
		return bufio.NewWriter(f)
	}
}
