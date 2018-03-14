package stats

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/sam"
)

// general element constants
const (
	Exon       = "exon"
	Intron     = "intron"
	Intergenic = "intergenic"
	ExonIntron = "exonic_intronic"
	Total      = "total"
	Other      = "others"
)

var (
	// allElems = []string{Exon, Intron, Intergenic, ExonIntron, Other, Total}
	allElems = []string{
		ExonIntron,
		Intron,
		Exon,
		Intergenic,
		Other,
		Total,
	}
)

// ElementStats represents mappings statistics for genomic elements
type ElementStats map[string]uint64

// type ElementStats struct {
// 	ExonIntron uint64 `json:"exonic_intronic"`
// 	Intron     uint64 `json:"intron"`
// 	Exon       uint64 `json:"exon"`
// 	Intergenic uint64 `json:"intergenic"`
// 	Other      uint64 `json:"others"`
// 	Total      uint64 `json:"total"`
// }

// CoverageStats represents genome coverage statistics for continuos, split and total mapped reads.
type CoverageStats struct {
	Total      ElementStats `json:"total"`
	Continuous ElementStats `json:"continuous"`
	Split      ElementStats `json:"split"`
	uniq       bool
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
	for _, elem := range allElems {
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
	l := len(allElems)
	for i, k := range allElems {
		val, err := json.Marshal(s[k])
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(buf, "\t\"%s\": %s", k, val)
		if i < l-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.Write([]byte{'}', '\n'})
	return buf.Bytes(), nil
}

func updateCount(r *sam.Record, elems map[string]uint8, st ElementStats) {
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
	st[ExonIntron]++
}

// func updateCount2(r *sam.Record, elems map[string]uint8, st ElementStats) {
// 	exons, hasExon := elems["exon"]
// 	introns, hasIntron := elems["intron"]
// 	st.Total++
// 	if _, isIntergenic := elems["intergenic"]; isIntergenic {
// 		if len(elems) > 1 {
// 			st.Other++
// 		} else {
// 			st.Intergenic++
// 		}
// 		return
// 	}
// 	if hasExon && !hasIntron && exons > 0 {
// 		st.Exon++
// 		return
// 	}
// 	if hasIntron && !hasExon && introns > 0 {
// 		st.Intron++
// 		return
// 	}
// 	st.ExonIntron++
// }

// Collect collects genome coverage statistics from a sam.Record.
func (s *CoverageStats) Collect(record *sam.Record, index *annotation.RtreeMap) {
	if index == nil || !record.IsPrimary() || record.IsUnmapped() {
		return
	}
	if s.uniq && !record.IsUniq() {
		return
	}
	elements := map[string]uint8{}
	for _, mappingLocation := range record.GetBlocks() {
		rtree := index.Get(mappingLocation.Chrom())
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
func NewCoverageStats() *CoverageStats {
	return &CoverageStats{
		Total:      make(ElementStats),
		Continuous: make(ElementStats),
		Split:      make(ElementStats),
	}
}
