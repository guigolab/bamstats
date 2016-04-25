package bamstats

import (
	"runtime"
	"testing"
)

var (
	bamFile        = "data/test1.bam"
	annotationFile = "data/gencode.v22.annotation.201503031.chr1.bed"
	maxBuf         = 1000000
	reads          = -1
)

func BenchmarkGeneral(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Process(bamFile, "", runtime.GOMAXPROCS(-1), maxBuf, reads)
	}
}

func BenchmarkCoverage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Process(bamFile, annotationFile, runtime.GOMAXPROCS(-1), maxBuf, reads)
	}
}
