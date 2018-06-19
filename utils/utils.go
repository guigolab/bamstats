package utils

import (
	"bufio"
	"io"
	"os"
)

// Max returns the maximum of two integers
func Max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewWriter return a new io.Writer given an output file name. If the file name is '-' os.Stdout is returned.
func NewWriter(output string) io.Writer {
	switch output {
	case "-":
		return os.Stdout
	default:
		f, err := os.Create(output)
		if err != nil {
			return nil
		}
		return bufio.NewWriter(f)
	}
}
