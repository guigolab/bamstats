package sam

import (
	"bytes"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/biogo/hts/sam"
	"github.com/guigolab/bamstats/annotation"
)

func sliceEq(a, b []location) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestSplit(t *testing.T) {
	for i, s := range []struct {
		line     []byte
		expected bool
		blocks   []location
	}{
		{
			[]byte("r001	99	ref	7	30	8M2I4M1D3M	=	37	39	TTAGATAAAGGATACTG	*\n"),
			false,
			[]location{location{"ref", 6, 22}},
		},
		{
			[]byte("r002	0	ref	9	30	3S6M1N1I4M	*	0	0	AAAAGATAAGGATA	*\n"),
			true,
			[]location{location{"ref", 8, 14}, location{"ref", 15, 19}},
		},
		{
			[]byte("r003	0	ref	9	30	5S6M	*	0	0	GCCTAAGCTAA	*	SA:Z:ref,29,-,6H5M,17,0;\n"),
			false,
			[]location{location{"ref", 8, 14}},
		},
		{
			[]byte("r004	0	ref	16	30	6M14N5M	*	0	0	ATAGCTTCAGC	*\n"),
			true,
			[]location{location{"ref", 15, 21}, location{"ref", 35, 40}},
		},
		{
			[]byte("r003	2064	ref	29	17	6H5M	*	0	0	TAGGC	*	SA:Z:ref,9,+,5S6M,30,1;\n"),
			false,
			[]location{location{"ref", 28, 33}},
		},
		{
			[]byte("r001	147	ref	37	30	9M	=	7	-39	CAGCGGCAT	*	NM:i:1\n"),
			false,
			[]location{location{"ref", 36, 45}},
		},
	} {
		sr, err := sam.NewReader(bytes.NewReader(s.line))
		checkTest(err, t)
		r, err := sr.Read()
		checkTest(err, t)
		split := isSplit(r)
		if split != s.expected {
			t.Errorf("(isSplit) [%d] %s: expected %v, got %v", i, r.Name, s.expected, split)
		}
		blocks := getBlocks(r)
		logrus.Info(blocks)
		if !sliceEq(blocks, s.blocks) {
			t.Errorf("(getBlocks) [%d] %s: expected %v, got %v", i, r.Name, s.blocks, blocks)
		}
	}
}

func TestFlags(t *testing.T) {
	for i, s := range []struct {
		line  []byte
		flags [8]bool
	}{
		{
			[]byte("r001	99	ref	7	30	8M2I4M1D3M	=	37	39	TTAGATAAAGGATACTG	*\n"),
			[8]bool{true, false, true, true, true, false, false, true},
		},
		{
			[]byte("r002	4	*	0	0	*	*	0	0	*	*\n"),
			[8]bool{true, true, false, false, false, false, false, false},
		},
		{
			[]byte("r003	9	ref	9	30	5S6M	*	0	0	GCCTAAGCTAA	*	SA:Z:ref,29,-,6H5M,17,0;\n"),
			[8]bool{true, false, true, false, false, false, true, false},
		},
		{
			[]byte("r004	256	ref	16	30	6M14N5M	*	0	0	ATAGCTTCAGC	*\n"),
			[8]bool{false, false, false, false, false, false, false, false},
		},
		{
			[]byte("r003	149	*	0	0	*	*	0	0	*	*\n"),
			[8]bool{true, true, true, false, false, true, false, false},
		},
		{
			[]byte("r001	147	ref	37	30	9M	=	7	-39	CAGCGGCAT	*	NM:i:1\n"),
			[8]bool{true, false, true, true, false, true, false, false},
		},
	} {
		sr, err := sam.NewReader(bytes.NewReader(s.line))
		checkTest(err, t)
		r, err := sr.Read()
		checkTest(err, t)
		flags := [8]bool{isPrimary(r), isUnmapped(r), isPaired(r), isProperlyPaired(r), isRead1(r), isRead2(r), hasMateUnmapped(r), isFirstOfValidPair(r)}
		if flags != s.flags {
			t.Errorf("(flags) [%d] %s: expected %v, got %v", i, r.Name, s.flags, flags)
		}
	}
}
