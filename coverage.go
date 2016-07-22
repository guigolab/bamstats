package bamstats

import "github.com/biogo/hts/sam"

// ElementStats represents mappings statistics for genomic elements
type ElementStats struct {
	ExonIntron int `json:"exonic_intronic"`
	Intron     int `json:"intron"`
	Exon       int `json:"exon"`
	Intergenic int `json:"intergenic"`
	Other      int `json:"others"`
	Total      int `json:"total"`
}

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
	s.Total.ExonIntron = s.Continuous.ExonIntron + s.Split.ExonIntron
	s.Total.Exon = s.Continuous.Exon + s.Split.Exon
	s.Total.Intron = s.Continuous.Intron + s.Split.Intron
	s.Total.Intergenic = s.Continuous.Intergenic + s.Split.Intergenic
	s.Total.Other = s.Continuous.Other + s.Split.Other
	s.Total.Total = s.Continuous.Total + s.Split.Total
}

// Update updates all counts from another ElementsStats instance.
func (s *ElementStats) Update(other ElementStats) {
	s.ExonIntron += other.ExonIntron
	s.Exon += other.Exon
	s.Intron += other.Intron
	s.Intergenic += other.Intergenic
	s.Other += other.Other
	s.Total += other.Total
}

func updateCount(r *sam.Record, elems map[string]uint8, st *ElementStats) {
	exons, hasExon := elems["exon"]
	introns, hasIntron := elems["intron"]
	st.Total++
	if _, isIntergenic := elems["intergenic"]; isIntergenic {
		if len(elems) > 1 {
			st.Other++
		} else {
			st.Intergenic++
		}
		return
	}
	if hasExon && !hasIntron && exons > 0 {
		st.Exon++
		return
	}
	if hasIntron && !hasExon && introns > 0 {
		st.Intron++
		return
	}
	st.ExonIntron++
}

// Collect collects genome coverage statistics from a sam.Record.
func (s *CoverageStats) Collect(record *sam.Record, index *RtreeMap) {
	if index == nil || !isPrimary(record) || isUnmapped(record) {
		return
	}
	if s.uniq && !isUniq(record) {
		return
	}
	elements := map[string]uint8{}
	for _, mappingLocation := range getBlocks(record) {
		results := QueryIndex(index.Get(mappingLocation.Chrom()), mappingLocation.Start(), mappingLocation.End())
		getElements(mappingLocation, &results, elements)
	}
	if isSplit(record) {
		updateCount(record, elements, &s.Split)
	} else {
		updateCount(record, elements, &s.Continuous)
	}
}

// NewCoverageStats create a new instance of CoverageStats.
func NewCoverageStats() *CoverageStats {
	return &CoverageStats{}
}
