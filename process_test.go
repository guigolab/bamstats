package bamstats

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	. "github.com/guigolab/bamstats/stats"
	. "github.com/guigolab/bamstats/utils"
)

func checkTest(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

var (
	bamFile                  = "data/process-test.bam"
	expectedGeneralJSON      = "data/expected-general.json"
	expectedCoverageJSON     = "data/expected-coverage.json"
	expectedCoverageUniqJSON = "data/expected-coverage-uniq.json"
	annotationFiles          = []string{"data/coverage-test.bed", "data/coverage-test.gtf.gz"}
	maxBuf                   = 1000000
	reads                    = -1
)

func readExpected(path string, t *testing.T) []byte {
	f, err := os.Open(path)
	checkTest(err, t)
	var b bytes.Buffer
	_, err = b.ReadFrom(f)
	checkTest(err, t)
	return b.Bytes()
}

func TestGeneral(t *testing.T) {
	var b bytes.Buffer
	out, err := Process(bamFile, "", runtime.GOMAXPROCS(-1), maxBuf, reads, false)
	checkTest(err, t)
	l := len(out)
	if l > 1 {
		t.Errorf("(Process) Expected StatsMap of length 1, got %d", l)
	}
	_, ok := out["general"].(*GeneralStats)
	if !ok {
		t.Errorf("(Process) Wrong return type - expected GeneralStats, got %T", out["general"])
	}
	OutputJSON(&b, out)
	stats := readExpected(expectedGeneralJSON, t)
	if len(b.Bytes()) != len(stats) {
		t.Error("(Process) GeneralStats are different")
	}
}

func TestCoverage(t *testing.T) {
	var b bytes.Buffer
	for _, annotationFile := range annotationFiles {
		b.Reset()
		out, err := Process(bamFile, annotationFile, runtime.GOMAXPROCS(-1), maxBuf, reads, false)
		checkTest(err, t)
		l := len(out)
		if l > 2 {
			t.Errorf("(Process) Expected StatsMap of length 2, got %d", l)
		}
		_, ok := out["general"].(*GeneralStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected GeneralStats, got %T", out["general"])
		}
		_, ok = out["coverage"].(*CoverageStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected CoverageStats, got %T", out["coverage"])
		}
		OutputJSON(&b, out)
		stats := readExpected(expectedCoverageJSON, t)
		if len(b.Bytes()) != len(stats) {
			t.Error("(Process) CoverageStats are different")
		}
	}
}

func TestCoverageUniq(t *testing.T) {
	var b bytes.Buffer
	for _, annotationFile := range annotationFiles {
		b.Reset()
		out, err := Process(bamFile, annotationFile, runtime.GOMAXPROCS(-1), maxBuf, reads, true)
		checkTest(err, t)
		l := len(out)
		if l > 3 {
			t.Errorf("(Process) Expected StatsMap of length 3, got %d", l)
		}
		_, ok := out["general"].(*GeneralStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected GeneralStats, got %T", out["general"])
		}
		_, ok = out["coverage"].(*CoverageStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected CoverageStats, got %T", out["coverage"])
		}
		OutputJSON(&b, out)
		stats := readExpected(expectedCoverageUniqJSON, t)
		if len(b.Bytes()) != len(stats) {
			t.Error("(Process) CoverageStats are different")
		}
	}
}

func BenchmarkGeneral(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Process(bamFile, "", runtime.GOMAXPROCS(-1), maxBuf, reads, false)
	}
}

func BenchmarkCoverage(b *testing.B) {
	for _, annotationFile := range annotationFiles {
		for i := 0; i < b.N; i++ {
			Process(bamFile, annotationFile, runtime.GOMAXPROCS(-1), maxBuf, reads, false)
		}
	}
}
