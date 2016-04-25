package bamstats_test

import (
	"runtime"
	"testing"

	"github.com/bamstats"
)

var (
	bam        = "data/test1.bam"
	annotation = "data/gencode.v22.annotation.201503031.chr1.bed"
	maxBuf     = 1000000
	reads      = -1
)

func BenchmarkGeneral(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bamstats.Process(bam, "", runtime.GOMAXPROCS(-1), maxBuf, reads)
	}
}

func BenchmarkCoverage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bamstats.Process(bam, annotation, runtime.GOMAXPROCS(-1), maxBuf, reads)
	}
}
