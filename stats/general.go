package stats

import (
	"math"

	"github.com/guigolab/bamstats/sam"
)

// MappedReadsStats represents statistics for mapped reads
type MappedReadsStats struct {
	Total    uint64 `json:"total,omitempty"`
	Unmapped uint64 `json:"unmapped,omitempty"`
	Mapped   TagMap `json:"mapped,omitempty"`
}

// MappingsStats represents statistics for mappings
type MappingsStats struct {
	MappedReadsStats
	Mappings MultimapStats `json:"mappings"`
}

// MappedPairsStats represents statistcs for mapped read-pairs
type MappedPairsStats struct {
	MappedReadsStats
	InsertSizes TagMap `json:"insert_sizes,omitempty"`
}

// MultimapStats represents statistics for multi-maps
type MultimapStats struct {
	Ratio float64 `json:"ratio"`
	Count uint64  `json:"count"`
}

// GeneralStats represents general mapping statistics
type GeneralStats struct {
	Reads MappingsStats    `json:"reads,omitempty"`
	Pairs MappedPairsStats `json:"pairs,omitempty"`
}

// Merge updates counts from a channel of Stats instances.
func (s *GeneralStats) Merge(others chan Stats) {
	for other := range others {
		if other, ok := other.(*GeneralStats); ok {
			s.Update(other)
		}
	}
}

// Update updates all counts from a Stats instance.
func (s *GeneralStats) Update(other Stats) {
	if other, ok := other.(*GeneralStats); ok {
		s.Reads.Update(other.Reads)
		s.Pairs.Update(other.Pairs)
		s.Pairs.MappedReadsStats.UpdateUnmapped()
	}
}

// Finalize updates dependent counts of a Stats instance.
func (s *GeneralStats) Finalize() {
	s.Reads.MappedReadsStats.UpdateUnmapped()
	s.Pairs.MappedReadsStats.UpdateUnmapped()
	s.Reads.UpdateMappingsRatio()
}

// Update updates all counts from another MappedReadStats instance.
func (s *MappedReadsStats) Update(other MappedReadsStats) {
	s.Total += other.Total
	s.Unmapped += other.Unmapped
	s.Mapped.Update(other.Mapped)
}

// UpdateUnmapped updates the count of unmapped pairs
func (s *MappedReadsStats) UpdateUnmapped() {
	s.Unmapped = s.Total - s.Mapped.Total()
}

// Update updates all counts from another MappingsStats instance.
func (s *MappingsStats) Update(other MappingsStats) {
	s.MappedReadsStats.Update(other.MappedReadsStats)
	s.Mappings.Count += other.Mappings.Count
	s.UpdateMappingsRatio()
}

// Update updates all counts from another MappedPairsStats instance.
func (s *MappedPairsStats) Update(other MappedPairsStats) {
	s.MappedReadsStats.Update(other.MappedReadsStats)
	s.InsertSizes.Update(other.InsertSizes)
}

// FilterInsertSizes filters out insert size lengths having support below the given percentage of total read-pairs.
func (s *MappedPairsStats) FilterInsertSizes(percent float64) {
	for k, v := range s.InsertSizes {
		if float64(v) < float64(s.Total)*(percent/100) {
			delete(s.InsertSizes, k)
		}
	}
}

// UpdateMappingsRatio updates ration of mappings vs total mapped reads.
func (s *MappingsStats) UpdateMappingsRatio() {
	s.Mappings.Ratio = float64(s.Mappings.Count) / float64(s.Mapped.Total())
}

// Unique returns the number of uniquely mapped reads.
func (s *MappedReadsStats) Unique() uint64 {
	return s.Mapped[1]
}

// NewGeneralStats creates a new instance of GeneralStats
func NewGeneralStats() *GeneralStats {
	ms := GeneralStats{}
	ms.Pairs = *NewMappedPairsStats()
	ms.Reads.MappedReadsStats = *NewMappedReadsStats()
	return &ms
}

// NewMappedReadsStats creates a new instance of MappedReadsStats
func NewMappedReadsStats() *MappedReadsStats {
	s := MappedReadsStats{}
	s.Mapped = make(TagMap)
	return &s
}

// NewMappedPairsStats creates a new instance of MappedPairsStats
func NewMappedPairsStats() *MappedPairsStats {
	s := MappedPairsStats{}
	s.MappedReadsStats = *NewMappedReadsStats()
	s.InsertSizes = make(TagMap)
	return &s
}

// Collect collects general mapping statistics from a sam.Record.
func (s *GeneralStats) Collect(r *sam.Record) {
	NH, hasNH := r.Tag([]byte("NH"))
	if !hasNH {
		NH, _ = sam.ParseAux([]byte("NH:i:0"))
	}
	NHKey := int(NH.Value().(uint8))
	if r.IsUnmapped() {
		s.Reads.Total++
		s.Reads.Unmapped++
		return
	}
	s.Reads.Mappings.Count++
	if r.IsPrimary() {
		s.Reads.Total++
		s.Reads.Mapped[NHKey]++
		if r.IsFirstOfValidPair() {
			s.Pairs.Total++
			s.Pairs.Mapped[NHKey]++
			isLen := int(math.Abs(float64(r.TempLen)))
			s.Pairs.InsertSizes[isLen]++
		}
	}
}
