package annotation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/dhconnelly/rtreego"
)

func TestParseFeature(t *testing.T) {
	e := bytes.Split([]byte(`chr1	11868	12227	exon	.	+`), []byte("\t"))
	chr := e[0]
	elem := e[3]
	score := parseScore(e[4])
	strand := e[5][0]
	start, end := parseInterval(e[1], e[2])
	if start != float64(11868) {
		t.Errorf("(parseInterval) expected %s, got %v", e[1], start)
	}
	if end != float64(12227) {
		t.Errorf("(parseInterval) expected %s, got %v", e[2], end)
	}
	f, err := parseFeature(chr, elem, score, strand, start, end)
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
	index := createIndex(NewScanner(bytes.NewReader(elements), map[string]int{}))
	l := index.Len()
	isTab := func(c rune) bool {
		return c == '\n'
	}
	expLen := len(bytes.FieldsFunc(elements, isTab))
	if l != expLen {
		t.Errorf("(createIndex) expected length %v, got %v", expLen, l)
	}
	for key, value := range *index {
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
	}
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
	index := createIndex(NewScanner(bytes.NewReader(elements), map[string]int{}))
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
	elements := []rtreego.Spatial{
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
	elements := []byte(`chr1	11868	12227	exon	.	+
chr2	12612	12721	exon	.	+
chr3	12974	13052	exon	.	+
chr4	13220	14501	exon	.	+
chr5	15004	15038	exon	.	+
chr6	15795	15947	exon	.	+
chr7	16606	16765	exon	.	+
chr8	16857	17055	exon	.	+
chr9	17232	17436	exon	.	+
chr10	17605	17742	exon	.	+
chr11	17914	18061	exon	.	+
chr12	18267	18366	exon	.	+
chr13	24737	24891	exon	.	+
chr14	29533	30039	exon	.	+
chr15	30266	30667	exon	.	+
chr16	30975	31109	exon	.	+
`)
	os.Setenv(dumpElementsEnv, "yes")
	debugElementsFile = ".test.debug.elfile.bed"
	createIndex(NewScanner(bytes.NewReader(elements), map[string]int{}))
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
	// os.Remove(debugElementsFile)
}

func TestSortFeatures(t *testing.T) {
	bsorted := []byte(`chr1	11868	12227	exon
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
chrX	13000	21000	exon
chrY	11000	23000	exon
chrM	22000	40000	exon
`)
	belements := []byte(`chr13	24737	24891	exon
chr2	12612	12721	exon
chrY	11000	23000	exon
chr11	17914	18061	exon
chr1	11868	12227	exon
chrX	13000	21000	exon
chr4	13220	14501	exon
chr5	15004	15038	exon
chr12	18267	18366	exon
chr6	15795	15947	exon
chr16	30975	31109	exon
chr7	16606	16765	exon
chrM	22000	40000	exon
chr8	16857	17055	exon
chr9	17232	17436	exon
chr10	17605	17742	exon
chr14	29533	30039	exon
chr3	12974	13052	exon
chr15	30266	30667	exon
`)
	var elements, sorted FeatureSlice
	s := NewScanner(bytes.NewReader(bsorted), map[string]int{})
	for s.Next() {
		sorted = append(sorted, s.Feat())
	}

	s = NewScanner(bytes.NewReader(belements), map[string]int{})
	for s.Next() {
		elements = append(elements, s.Feat())
	}
	sort.Sort(elements)
	for i, v := range elements {
		if v.String() != sorted[i].String() {
			t.Errorf("Wrong sort: expected %s, got %s", sorted[i], v)
		}
	}
}

func TestIssue23(t *testing.T) {
	chr := []byte("chr1")
	element := []byte("exon")
	elements := []rtreego.Spatial{
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
			location: newRect(rtreego.Point{13221}, []float64{1188}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{13453}, []float64{217}, t),
		},
	}

	intron := []byte("intron")
	expected := []*Feature{
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{11869}, []float64{358}, t),
		},
		&Feature{
			chr:      chr,
			element:  intron,
			location: newRect(rtreego.Point{12227}, []float64{386}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12613}, []float64{108}, t),
		},
		&Feature{
			chr:      chr,
			element:  intron,
			location: newRect(rtreego.Point{12721}, []float64{254}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{12975}, []float64{77}, t),
		},
		&Feature{
			chr:      chr,
			element:  intron,
			location: newRect(rtreego.Point{13052}, []float64{169}, t),
		},
		&Feature{
			chr:      chr,
			element:  element,
			location: newRect(rtreego.Point{13221}, []float64{1188}, t),
		},
	}
	results := interleaveFeatures(mergeIntervals(elements), 11869, 14409, "exon", []byte("intron"), false)
	if len(results) != len(expected) {
		t.Errorf("(MergeElements) Lengths of merged results differ from expected results.\ngot: %v \nexp: %v)", len(results), len(expected))
	}
	for i, e := range expected {
		if e.String() != results[i].String() {
			t.Errorf("(MergeElements) merged results error.\ngot: %v \nexp: %v", results[i], e)
		}
	}
}

func TestReadFeatures(t *testing.T) {
	chrLens := map[string]int{
		"chr1": 248956422,
	}

	elems := map[string]int{
		"exon":       106742,
		"gene":       5397,
		"intergenic": 3202,
		"intron":     26230,
	}
	mergedElems := map[string]int{
		"exon":       29431,
		"gene":       3201,
		"intergenic": 3202,
		"intron":     26230,
	}

	for _, i := range []struct {
		f        string
		expected map[string]int
	}{
		{
			"../data/coverage-test.bed",
			mergedElems,
		},
		{
			"../data/coverage-test-shuffled.bed",
			mergedElems,
		},
		{
			"../data/coverage-test.gtf.gz",
			elems,
		},
		{
			"../data/coverage-test-shuffled.gtf.gz",
			elems,
		},
	} {
		m := CreateIndex(i.f, chrLens)
		index := m.Get("chr1")
		res := make(map[string]int)
		for _, s := range QueryIndex(index, 0, 248956422) {
			f := s.(*Feature)
			res[f.Element()]++
		}
		for k, v := range i.expected {
			if v != res[k] {
				t.Errorf("(%s) Different number of %s features. Expected: %d, got %d", t.Name(), k, v, res[k])
			}
		}
	}
}
