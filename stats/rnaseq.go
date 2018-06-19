package stats

import (
	"github.com/dhconnelly/rtreego"
	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/sam"
)

// RNAseqMetrics represents statistics for mapped reads
type RNAseqMetrics struct {
	Mapped     fraction `json:"fraction_mapped,omitempty"`
	Intergenic fraction `json:"fraction_intergenic,omitempty"`
	RRNA       fraction `json:"fraction_rrna,omitempty"`
	Duplicates fraction `json:"fraction_duplicates,omitempty"`
}

// RNAseqStats represents statistics for mapped reads
type RNAseqStats struct {
	total, mapped, duplicates uint64
	Intergenic                uint64         `json:"intergenic"`
	RRNA                      uint64         `json:"rRNA"`
	Metrics                   *RNAseqMetrics `json:"metrics,omitempty"`
	index                     *annotation.RtreeMap
}

// Merge updates counts from a channel of Stats instances.
func (s *RNAseqStats) Merge(others chan Stats) {
	for other := range others {
		if other, ok := other.(*RNAseqStats); ok {
			s.Update(other)
		}
	}
}

// Update updates all counts from a Stats instance.
func (s *RNAseqStats) Update(other Stats) {
	if other, isIHEC := other.(*RNAseqStats); isIHEC {
		s.Intergenic += other.Intergenic
		s.RRNA += other.RRNA
		s.duplicates += other.duplicates
		s.total += other.total
		s.mapped += other.mapped
	}
}

// Finalize updates dependent counts of a Stats instance.
func (s *RNAseqStats) Finalize() {
	if s.total > 0 {
		s.Metrics.Mapped = fraction(s.mapped) / fraction(s.total)
		if s.mapped > 0 {
			s.Metrics.Intergenic = fraction(s.Intergenic) / fraction(s.mapped)
			s.Metrics.RRNA = fraction(s.RRNA) / fraction(s.mapped)
			s.Metrics.Duplicates = fraction(s.duplicates) / fraction(s.mapped)
		}
	}
}

// Collect collects general mapping statistics from a sam.Record.
func (s *RNAseqStats) Collect(record *sam.Record) {
	elements := map[string]uint8{}
	if s.index == nil || !record.IsPrimary() {
		return
	}
	s.total++
	if record.IsUnmapped() {
		return
	}
	s.mapped++
	if record.IsDuplicate() {
		s.duplicates++
	}
	mappingLocation := annotation.NewLocation(record.Ref.Name(), record.Start(), record.End())
	rtree := s.index.Get(mappingLocation.Chrom())
	if rtree == nil || rtree.Size() == 0 {
		return
	}

	results := annotation.QueryIndex(rtree, mappingLocation.Start(), mappingLocation.End())

	mappingLocation.GetElements(filterElements(results, mappingLocation.Start(), mappingLocation.End(), 500), elements, "gene_type")

	updateIHECcount(elements, s)
}

// NewIHECstats creates a new instance of IHECstats
func NewIHECstats(index *annotation.RtreeMap) *RNAseqStats {
	return &RNAseqStats{
		index:   index,
		Metrics: &RNAseqMetrics{},
	}
}

func filterElements(elements []rtreego.Spatial, start, end, offset float64) []rtreego.Spatial {
	var filteredElements []rtreego.Spatial
	for _, r := range elements {
		if r, ok := r.(*annotation.Feature); ok {
			if r.Element() == "intergenic" {
				if r.End()-r.Start() < 2*offset {
					continue
				}
				if end <= r.Start()+offset || start > r.End()-offset {
					continue
				}
			}
			filteredElements = append(filteredElements, r)
		}
	}
	return filteredElements
}

func updateIHECcount(elems map[string]uint8, st *RNAseqStats) {

	if len(elems) == 0 {
		return
	}

	rRNAs := []string{
		"rRNA",
		"Mt_rRNA",
	}

	for _, gt := range rRNAs {
		if _, isRRNA := elems[gt]; isRRNA {
			st.RRNA++
		}
	}

	if _, isIntergenic := elems["intergenic"]; isIntergenic {
		st.Intergenic++
	}

}
