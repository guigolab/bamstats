package bamstats_test

import (
  "runtime"
  "testing"
  "github.com/bamstats"
)

type location struct {
  name string
  start, end uint32
}

func (s location) Chrom() string {
	return s.name
}
func (s location) Start() uint32 {
	return s.start
}
func (s location) End() uint32 {
	return s.end
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

var (
  bam = "data/test2.bam"
  annotation = "data/gencode.v22.annotation.201503031.all.bed.gz"
)

func BenchmarkCoverage(b *testing.B) {
  for i := 0; i < b.N; i++ {
	   bamstats.Coverage(bam, annotation, runtime.GOMAXPROCS(-1))
  }
}

func BenchmarkCoverage1(b *testing.B) {
  for i := 0; i < b.N; i++ {
    bamstats.Coverage1(bam, annotation, runtime.GOMAXPROCS(-1))
  }
}
