package stats

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/sam"
)

// general element constants
const (
	Exon       = "exon"
	ExonIntron = "exonic_intronic"
	Intergenic = "intergenic"
	Intron     = "intron"
	Other      = "others"
	Total      = "total"
)

var (
	allElems = []string{
		ExonIntron,
		Intron,
		Exon,
		Intergenic,
		Other,
		Total,
	}
)

func getElemSet() map[string]struct{} {
	elemSet := make(map[string]struct{})
	for _, el := range allElems {
		elemSet[el] = struct{}{}
	}
	return elemSet
}

// ElementStats represents mappings statistics for genomic elements
type ElementStats map[string]uint64

// Keys returns map keys
func (s ElementStats) Keys() []string {
	var keys []string
	elemSet := getElemSet()
	for k := range s {
		if _, found := elemSet[k]; !found {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return append(keys, allElems...)
}

// MergeKeys combine keys from two ElementsStats instances
func (s ElementStats) MergeKeys(other ElementStats) []string {
	elemSet := getElemSet()
	var keys []string
	for _, k := range s.Keys() {
		if _, found := elemSet[k]; !found {
			keys = append(keys, k)
		}
	}
	for _, k := range other.Keys() {
		if _, found := elemSet[k]; !found {
			if _, hasThis := s[k]; !hasThis {
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return append(keys, allElems...)
}

// CoverageStats represents genome coverage statistics for continuos, split and total mapped reads.
type CoverageStats struct {
	Total      ElementStats `json:"total"`
	Continuous ElementStats `json:"continuous"`
	Split      ElementStats `json:"split"`
	Uniq       bool         `json:"-"`
	index      *annotation.RtreeMap
}

// Update updates all counts from a Stats instance.
func (s *CoverageStats) Update(other Stats) {
	if other, ok := other.(*CoverageStats); ok {
		s.Continuous.Update(other.Continuous)
		s.Split.Update(other.Split)
		s.Finalize()
	}
}

// Merge update counts from a channel of Stats instances.
func (s *CoverageStats) Merge(others chan Stats) {
	for other := range others {
		if other, ok := other.(*CoverageStats); ok {
			s.Update(other)
		}
	}
}

// Finalize updates dependent counts of a CoverageStats instance.
func (s *CoverageStats) Finalize() {
	for _, elem := range s.Continuous.MergeKeys(s.Split) {
		s.Total[elem] = s.Continuous[elem] + s.Split[elem]
	}
}

// Update updates all counts from another ElementsStats instance.
func (s ElementStats) Update(other ElementStats) {
	for k := range other {
		s[k] += other[k]
	}
}

// MarshalJSON implements JSON Marshaller interface
func (s ElementStats) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{'{', '\n'})
	keys := s.Keys()
	l := len(keys)
	for i, k := range keys {
		// TODO: write 0 values elements? Update test output in case
		// if s[k] == 0 {
		// 	continue
		// }
		fmt.Fprintf(buf, "\t\"%s\": %d", k, s[k])
		if i < l-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.Write([]byte{'}', '\n'})
	return buf.Bytes(), nil
}

func updateCount(r *sam.Record, elems map[string]uint8, st ElementStats) {
	if len(elems) == 0 {
		return
	}
	exons, hasExon := elems["exon"]
	introns, hasIntron := elems["intron"]
	st[Total]++
	if _, isIntergenic := elems["intergenic"]; isIntergenic {
		if len(elems) > 1 {
			st[Other]++
		} else {
			st[Intergenic]++
		}
		return
	}
	if hasExon && !hasIntron && exons > 0 {
		st[Exon]++
		return
	}
	if hasIntron && !hasExon && introns > 0 {
		st[Intron]++
		return
	}
	if hasIntron && hasExon && introns > 0 && exons > 0 {
		st[ExonIntron]++
		return
	}
	var keys []string
	for k := range elems {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for i, e := range keys {
		if i > 0 {
			buf.WriteByte('_')
		}
		buf.WriteString(e)
	}
	st[buf.String()]++
}

// Collect collects genome coverage statistics from a sam.Record.
func (s *CoverageStats) Collect(record *sam.Record) {
	if s.index == nil || !record.IsPrimary() || record.IsUnmapped() {
		return
	}
	if s.Uniq && !record.IsUniq() {
		return
	}
	elements := map[string]uint8{}
	for _, mappingLocation := range record.GetBlocks() {
		rtree := s.index.Get(mappingLocation.Chrom())
		if rtree.Size() == 0 {
			return
		}
		results := annotation.QueryIndex(rtree, mappingLocation.Start(), mappingLocation.End())
		mappingLocation.GetElements(&results, elements)
	}
	if record.IsSplit() {
		updateCount(record, elements, s.Split)
	} else {
		updateCount(record, elements, s.Continuous)
	}
}

// NewCoverageStats create a new instance of CoverageStats.
func NewCoverageStats(index *annotation.RtreeMap, uniq bool) *CoverageStats {
	return &CoverageStats{
		Total:      make(ElementStats),
		Continuous: make(ElementStats),
		Split:      make(ElementStats),
		Uniq:       uniq,
		index:      index,
	}
}
