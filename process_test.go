package bamstats

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/guigolab/bamstats/stats"
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
	expectedRNAseqJSON       = "data/expected-rnaseq.json"
	annotationFiles          = []string{"data/coverage-test.bed", "data/coverage-test.gtf.gz", "data/coverage-test-shuffled.bed", "data/coverage-test-shuffled.gtf.gz"}
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
	_, ok := out["general"].(*stats.GeneralStats)
	if !ok {
		t.Errorf("(Process) Wrong return type - expected GeneralStats, got %T", out["general"])
	}
	out.OutputJSON(&b)
	stats := readExpected(expectedGeneralJSON, t)
	if len(b.Bytes()) != len(stats) {
		err := dump(b, "observed-general.json")
		if err != nil {
			t.Errorf("(Process) Debug dump error: %s", err)
		}
		t.Error("(Process) GeneralStats are different")
	}
}

func TestIssue18(t *testing.T) {
	out, err := Process("data/issue18.bam", "", runtime.GOMAXPROCS(-1), maxBuf, reads, false)
	checkTest(err, t)
	l := len(out)
	if l != 1 {
		t.Errorf("(Process) Expected StatsMap of length 1, got %d", l)
	}
	_, ok := out["general"].(*stats.GeneralStats).Reads.Mapped[1]
	if !ok {
		t.Errorf("(Process) Bad NH tag in read stats")
	}
}

func TestCoverage(t *testing.T) {
	var b bytes.Buffer
	expectedMapLen := 3
	for _, annotationFile := range annotationFiles {
		b.Reset()
		out, err := Process(bamFile, annotationFile, runtime.GOMAXPROCS(-1), maxBuf, reads, false)
		checkTest(err, t)
		l := len(out)
		if l > expectedMapLen {
			t.Errorf("(Process) Expected StatsMap of length %d, got %d", expectedMapLen, l)
		}
		_, ok := out["general"].(*stats.GeneralStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected GeneralStats, got %T", out["general"])
		}
		_, ok = out["coverage"].(*stats.CoverageStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected CoverageStats, got %T", out["coverage"])
		}
		out.OutputJSON(&b)
		stats := readExpected(expectedCoverageJSON, t)
		if len(b.Bytes()) != len(stats) {
			err := dump(b, "observed-coverage.json")
			if err != nil {
				t.Errorf("(Process) Debug dump error: %s", err)
			}
			t.Error("(Process) CoverageStats are different")
		}
	}
}

func TestCoverageUniq(t *testing.T) {
	var b bytes.Buffer
	expectedMapLen := 4
	for _, annotationFile := range annotationFiles {
		b.Reset()
		out, err := Process(bamFile, annotationFile, runtime.GOMAXPROCS(-1), maxBuf, reads, true)
		checkTest(err, t)
		l := len(out)
		if l > expectedMapLen {
			t.Errorf("(Process) Expected StatsMap of length %d, got %d", expectedMapLen, l)
		}
		_, ok := out["general"].(*stats.GeneralStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected GeneralStats, got %T", out["general"])
		}
		_, ok = out["coverage"].(*stats.CoverageStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected CoverageStats, got %T", out["coverage"])
		}
		_, ok = out["coverageUniq"].(*stats.CoverageStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected CoverageStats, got %T", out["coverageUniq"])
		}
		out.OutputJSON(&b)
		stats := readExpected(expectedCoverageUniqJSON, t)
		if len(b.Bytes()) != len(stats) {
			err := dump(b, "observed-coverage-uniq.json")
			if err != nil {
				t.Errorf("(Process) Debug dump error: %s", err)
			}
			t.Error("(Process) CoverageStats are different")
		}
	}
}

func TestRNAseq(t *testing.T) {
	var b bytes.Buffer
	bamFile := "data/rnaseq-test.bam"
	annotationFile := "data/rnaseq.gtf.gz"
	expectedMapLen := 3
	out, err := Process(bamFile, annotationFile, runtime.GOMAXPROCS(-1), maxBuf, reads, false)
	checkTest(err, t)
	l := len(out)
	if l > expectedMapLen {
		t.Errorf("(Process) Expected StatsMap of length %d, got %d", expectedMapLen, l)
	}
	_, ok := out["general"].(*stats.GeneralStats)
	if !ok {
		t.Errorf("(Process) Wrong return type - expected GeneralStats, got %T", out["general"])
	}
	_, ok = out["coverage"].(*stats.CoverageStats)
	if !ok {
		t.Errorf("(Process) Wrong return type - expected CoverageStats, got %T", out["coverage"])
	}
	_, ok = out["rnaseq"].(*stats.RNAseqStats)
	if !ok {
		t.Errorf("(Process) Wrong return type - expected RNAseqStats, got %T", out["rnaseq"])
	}
	out.OutputJSON(&b)
	stats := readExpected(expectedRNAseqJSON, t)
	if len(b.Bytes()) != len(stats) {
		err := dump(b, "observed-rnaseq.json")
		if err != nil {
			t.Errorf("(Process) Debug dump error: %s", err)
		}
		t.Error("(Process) RNAseqStats are different")
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

func dump(b bytes.Buffer, fname string) error {
	s, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Write(b.Bytes())
	return err
}
