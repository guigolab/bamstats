package bamstats

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"testing"
)

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
	index := createIndex(bufio.NewScanner(bytes.NewReader(elements)))
	l := len(*index)
	if l != 16 {
		t.Errorf("(createIndex) expected length 15, got %v", l)
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
	index := createIndex(bufio.NewScanner(bytes.NewReader(elements)))
	for _, item := range []struct {
		query          location
		expectedLength int
	}{
		{location{"chr1", 17145, 17234}, 3},
	} {
		results := QueryIndex(index.Get(item.query.Chrom()), item.query.Start(), item.query.End())

		l := len(results)
		if l != item.expectedLength {
			t.Errorf("(QueryIndex) expected %v, got %v results", item.expectedLength, l)
		}
	}
}
