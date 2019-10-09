package bamstats

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
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
	bamFile      = "data/process-test.bam"
	expectedJSON = map[string]string{
		"general":      "data/expected-general.json",
		"coverage":     "data/expected-coverage.json",
		"coverageUniq": "data/expected-coverage-uniq.json",
		"rnaseq":       "data/expected-rnaseq.json",
		"strand":       "data/expected-strand.json",
	}
	expectedMapLenCoverage     = 4
	expectedMapLenCoverageUniq = 5
	annotationFiles            = []string{"data/coverage-test.bed", "data/coverage-test.gtf.gz", "data/coverage-test-shuffled.bed", "data/coverage-test-shuffled.gtf.gz"}
	maxBuf                     = 1000000
	reads                      = -1
)

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
	stats := readStats([]string{"general"}, t)
	// stats := readExpected(expectedGeneralJSON, t)
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
	expectedMapLen := expectedMapLenCoverage
	for _, annotationFile := range append(annotationFiles, "data/coverage-test-merged.bed", "data/coverage-test-merged-shuffled.bed") {
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
		stats.NewMap(out["coverage"]).OutputJSON(&b)
		stats := readStats([]string{"coverage"}, t)
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
	expectedMapLen := expectedMapLenCoverageUniq
	for _, annotationFile := range append(annotationFiles, "data/coverage-test-merged.bed", "data/coverage-test-merged-shuffled.bed") {
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
		stats.NewMap(out["coverageUniq"]).OutputJSON(&b)
		stats := readStats([]string{"coverageUniq"}, t)
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
	expectedMapLen := expectedMapLenCoverage
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
	stats.NewMap(out["rnaseq"]).OutputJSON(&b)
	stats := readStats([]string{"rnaseq"}, t)
	if len(b.Bytes()) != len(stats) {
		err := dump(b, "observed-rnaseq.json")
		if err != nil {
			t.Errorf("(Process) Debug dump error: %s", err)
		}
		t.Error("(Process) RNAseqStats are different")
	}
}

func TestStrand(t *testing.T) {
	var b bytes.Buffer
	expectedMapLen := expectedMapLenCoverage
	for _, annotationFile := range annotationFiles {
		b.Reset()
		out, err := Process(bamFile, annotationFile, runtime.GOMAXPROCS(-1), maxBuf, reads, false)
		checkTest(err, t)
		l := len(out)
		if l > expectedMapLen {
			t.Errorf("(Process) Expected StatsMap of length %d, got %d", expectedMapLen, l)
		}
		_, ok := out["strand"].(*stats.StrandStats)
		if !ok {
			t.Errorf("(Process) Wrong return type - expected StrandStats, got %T", out["coverage"])
		}
		stats.NewMap(out["strand"]).OutputJSON(&b)
		stats := readStats([]string{"strand"}, t)
		if len(b.Bytes()) != len(stats) {
			err := dump(b, "observed-strand.json")
			if err != nil {
				t.Errorf("(Process) Debug dump error: %s", err)
			}
			t.Error("(Process) StrandStats are different")
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

func dump(b bytes.Buffer, fname string) error {
	s, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Write(b.Bytes())
	return err
}

func readStats(keys []string, t *testing.T) []byte {
	stats := make(map[string]interface{})
	for _, k := range keys {
		stats[k] = readJSON(expectedJSON[k], t)
	}
	b, err := json.MarshalIndent(stats, "", "\t")
	if err != nil {
		return nil
	}
	return b
}

func readJSON(path string, t *testing.T) interface{} {
	b, err := ioutil.ReadFile(path)
	checkTest(err, t)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}
