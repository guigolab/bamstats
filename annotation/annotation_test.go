package annotation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/Sirupsen/logrus"

	"github.com/dhconnelly/rtreego"
)

func TestParseFeature(t *testing.T) {
	e := bytes.Split([]byte(`chr1	11868	12227	exon`), []byte("\t"))
	chr := e[0]
	elem := e[3]
	start, end := parseInterval(e[1], e[2])
	if start != float64(11868) {
		t.Errorf("(parseInterval) expected %s, got %v", e[1], start)
	}
	if end != float64(12227) {
		t.Errorf("(parseInterval) expected %s, got %v", e[2], end)
	}
	f, err := parseFeature(chr, elem, start, end)
	if err != nil {
		t.Errorf("(parseFeature) Got error: %s", err)
	}
	if f.Chr() != "chr1" {
		t.Errorf("(parseFeature) expected chromosome %s, got %v", "chr1", f.Chr())
	}
	if f.Element() != "exon" {
		t.Errorf("(parseFeature) expected element %s, got %v", "exon", f.Chr())
	}
	if f.Start() != 11868 {
		t.Errorf("(parseFeature) expected start %d, got %v", 11868, f.Start())
	}
	if f.End() != 12227 {
		t.Errorf("(parseFeature) expected end %d, got %v", 12227, f.End())
	}
}

func TestCreateIndex(t *testing.T) {
	elements := []byte(`chr1	11868	12227	exon
chr2	12612	12721	exon
chr3	12974	13052	exon
chr4	13220	14501	exon
chr5	15004	15038	exon
chr6	15795	15947	exon
chr7	16606	16765	exon
chr8	16857	17055	exon
chr9	17232	17436	exon
chr10	17605	17742	exon
chr11	17914	18061	exon
chr12	18267	18366	exon
chr13	24737	24891	exon
chr14	29533	30039	exon
chr15	30266	30667	exon
chr16	30975	31109	exon
`)
	index := createIndex(NewScanner(bytes.NewReader(elements), map[string]int{}), nil)
	l := index.Len()
	isTab := func(c rune) bool {
		return c == '\n'
	}
	expLen := len(bytes.FieldsFunc(elements, isTab))
	if l != expLen {
		t.Errorf("(createIndex) expected length %v, got %v", expLen, l)
	}
	index.Range(func(k, v interface{}) bool {
		key := k.(string)
		value := v.(*rtreego.Rtree)
		typeString := fmt.Sprintf("%T", value)
		if typeString != "*rtreego.Rtree" {
			t.Errorf("(createIndex) expected *rtreego.Rtree, got %v", typeString)
		}
		validChr := regexp.MustCompile(`^chr`)
		if !validChr.MatchString(key) {
			t.Errorf("(createIndex) expected chrN key, got %v", key)
		}
		indexSize := value.Size()
		if indexSize != 1 {
			t.Errorf("(createIndex) expected one value per chromosome, got %v", indexSize)
		}
		return true
	})
}

func TestQueryIndex(t *testing.T) {
	elements := []byte(`chr1	11868	12227	exon
chr1	11868	31109	gene
chr1	12227	12612	intron
chr1	12612	12721	exon
chr1	12721	12974	intron
chr1	12974	13052	exon
chr1	13052	13220	intron
chr1	13220	14501	exon
chr1	14501	15004	intron
chr1	15004	15038	exon
chr1	15038	15795	intron
chr1	15795	15947	exon
chr1	15947	16606	intron
chr1	16606	16765	exon
chr1	16765	16857	intron
chr1	16857	17055	exon
chr1	17055	17232	intron
chr1	17232	17436	exon
chr1	17436	17605	intron
chr1	17605	17742	exon
chr1	17742	17914	intron
chr1	17914	18061	exon
chr1	18061	18267	intron
chr1	18267	18366	exon
chr1	18366	24737	intron
chr1	24737	24891	exon
chr1	24891	29533	intron
chr1	29533	30039	exon
chr1	30039	30266	intron
chr1	30266	30667	exon
chr1	30667	30975	intron
chr1	30975	31109	exon
`)
	index := createIndex(NewScanner(bytes.NewReader(elements), map[string]int{}), nil)
	for _, item := range []struct {
		query          Location
		expectedLength int
	}{
		{Location{"chr1", 17145, 17234}, 3},
	} {
		results := QueryIndex(index.Get(item.query.Chrom()), item.query.Start(), item.query.End())

		l := len(results)
		if l != item.expectedLength {
			t.Errorf("(QueryIndex) expected %v, got %v results", item.expectedLength, l)
		}
	}
}

func newRect(point rtreego.Point, size []float64, t *testing.T) *rtreego.Rect {
	rect, err := rtreego.NewRect(point, size)
	if err != nil {
		t.Fatal(err)
	}
	return rect
}

func TestMergeIntervals(t *testing.T) {
	chr := []byte("chr1")
	element := []byte("exon")
	elements := []*Feature{
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{11869}, []float64{358}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12010}, []float64{47}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12179}, []float64{48}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12613}, []float64{84}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12613}, []float64{108}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12975}, []float64{77}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{13221}, []float64{153}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{13221}, []float64{1188}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{13453}, []float64{217}, t),
		},
	}
	expected := []*Feature{
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{11869}, []float64{358}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12613}, []float64{108}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12975}, []float64{77}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{13221}, []float64{1188}, t),
		},
	}
	results := mergeIntervals(elements)
	if len(results) != len(expected) {
		t.Errorf("(MergeElements) Lengths of merged results differ from expected results.\ngot: %v \nexp: %v)", len(results), len(expected))
	}
	for i, e := range expected {
		if e.String() != results[i].String() {
			t.Errorf("(MergeElements) merged results error.\ngot: %v \nexp: %v", results[i], e)
		}
	}
}

func TestWriteElements(t *testing.T) {
	elements := []byte(`chr1	11868	12227	exon
chr2	12612	12721	exon
chr3	12974	13052	exon
chr4	13220	14501	exon
chr5	15004	15038	exon
chr6	15795	15947	exon
chr7	16606	16765	exon
chr8	16857	17055	exon
chr9	17232	17436	exon
chr10	17605	17742	exon
chr11	17914	18061	exon
chr12	18267	18366	exon
chr13	24737	24891	exon
chr14	29533	30039	exon
chr15	30266	30667	exon
chr16	30975	31109	exon
`)
	logrus.SetLevel(logrus.DebugLevel) // set debug level
	debugElementsFile = ".test.debug.elfile.bed"
	_ = createIndex(NewScanner(bytes.NewReader(elements), map[string]int{}), nil)
	e, err := ioutil.ReadFile(debugElementsFile)
	if os.IsNotExist(err) {
		t.Fatal("(createIndex) Debug elements file not found")
	}
	if err != nil {
		t.Fatalf("(createIndex) Cannot read debug elements file: %s", err)
	}
	if bytes.Compare(elements, e) != 0 {
		t.Fatalf("(createIndex) Debug elements file contents do not match the expected value")
	}
	os.Remove(debugElementsFile)
}
