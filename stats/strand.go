package stats

import (
	"fmt"
	"math"

	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/sam"
)

var (
	strandMap = map[string]int8{
		"+": +1,
		"-": -1,
	}

	specLabelMap = map[string][]string{
		"SingleEnd": []string{
			"++,--",
			"+-,-+",
		},
		"PairedEnd": []string{
			"1++,1--,2+-,2-+",
			"1+-,1-+,2++,2--",
		},
	}

	strandnessMap = map[string]string{
		"++,--":           "SENSE",
		"+-,-+":           "ANTISENSE",
		"1++,1--,2+-,2-+": "MATE1_SENSE",
		"1+-,1-+,2++,2--": "MATE2_SENSE",
	}
)

// SingleStrandStats represents strand statistics for single end reads
type SingleStrandStats struct {
	MapPlusGenePlus   uint64 `json:"++"`
	MapPlusGeneMinus  uint64 `json:"+-"`
	MapMinusGeneMinus uint64 `json:"--"`
	MapMinusGenePlus  uint64 `json:"-+"`
}

// Update updates all counts from another StrandStats instance.
func (s *SingleStrandStats) Update(other *SingleStrandStats) {
	s.MapPlusGenePlus += other.MapPlusGenePlus
	s.MapPlusGeneMinus += other.MapPlusGeneMinus
	s.MapMinusGenePlus += other.MapMinusGenePlus
	s.MapMinusGeneMinus += other.MapMinusGeneMinus
}

// PairedStrandStats represents strand statistics for paired end reads
type PairedStrandStats struct {
	FirstMapPlusGenePlus    uint64 `json:"1++"`
	FirstMapPlusGeneMinus   uint64 `json:"1+-"`
	FirstMapMinusGeneMinus  uint64 `json:"1--"`
	FirstMapMinusGenePlus   uint64 `json:"1-+"`
	SecondMapPlusGenePlus   uint64 `json:"2++"`
	SecondMapPlusGeneMinus  uint64 `json:"2+-"`
	SecondMapMinusGeneMinus uint64 `json:"2--"`
	SecondMapMinusGenePlus  uint64 `json:"2-+"`
}

// Update updates all counts from another StrandStats instance.
func (s *PairedStrandStats) Update(other *PairedStrandStats) {
	s.FirstMapPlusGenePlus += other.FirstMapPlusGenePlus
	s.FirstMapPlusGeneMinus += other.FirstMapPlusGeneMinus
	s.FirstMapMinusGenePlus += other.FirstMapMinusGenePlus
	s.FirstMapMinusGeneMinus += other.FirstMapMinusGeneMinus
	s.SecondMapPlusGenePlus += other.SecondMapPlusGenePlus
	s.SecondMapPlusGeneMinus += other.SecondMapPlusGeneMinus
	s.SecondMapMinusGenePlus += other.SecondMapMinusGenePlus
	s.SecondMapMinusGeneMinus += other.SecondMapMinusGeneMinus
}

// StrandStats represents strand statistics
type StrandStats struct {
	Total      uint64  `json:"total"`
	Failed     float64 `json:"failed"`
	threshold  float64
	mapQ       byte
	Strandness string             `json:"strandness"`
	Reads      *SingleStrandStats `json:"reads,omitempty"`
	Pairs      *PairedStrandStats `json:"pairs,omitempty"`
	index      *annotation.RtreeMap
}

// Collect collects strand statistics from a sam.Record.
func (s *StrandStats) Collect(record *sam.Record) {
	NH, hasNH := record.Tag([]byte("NH"))
	multiMap := hasNH && NH.Value().(uint8) > 1
	if s.index == nil || !record.IsPrimary() || record.IsUnmapped() || record.IsDuplicate() || record.IsQCFail() || multiMap || record.MapQ < s.mapQ {
		return
	}
	rtree := s.index.Get(record.Ref.Name())
	if rtree == nil || rtree.Size() == 0 {
		return
	}
	// var results []rtreego.Spatial
	check := make(map[int8]uint8)
	for _, mappingLocation := range record.GetBlocks() {
		rtree := s.index.Get(mappingLocation.Chrom())
		if rtree == nil || rtree.Size() == 0 {
			return
		}
		for _, r := range annotation.QueryIndex(rtree, mappingLocation.Start(), mappingLocation.End()) {
			if f, ok := r.(*annotation.Feature); ok {
				if f.Element() == "gene" {
					continue
				}
				start := math.Max(mappingLocation.Start(), f.Start())
				end := math.Min(mappingLocation.End(), f.End())
				if end <= start {
					continue
				}
				// results = append(results, r)
				f := r.(*annotation.Feature)
				if record.Strand() == strandMap[f.Strand()] {
					check[1]++
				} else {
					check[-1]++
				}
			}
		}
	}
	// results := annotation.QueryIndex(rtree, loc.Start(), loc.End())
	if check[1] == 0 && check[-1] == 0 {
		return
	}
	if check[1] > 0 && check[-1] > 0 {
		s.Failed++
	}
	s.Total++
	if check[-1] == 0 {
		s.IncrementSense(record)
	}
	if check[1] == 0 {
		s.IncrementAntisense(record)
	}
}

// IncrementSense increment counts for sense reads
func (s *StrandStats) IncrementSense(record *sam.Record) {
	if record.IsPaired() {
		if s.Pairs == nil {
			s.Pairs = &PairedStrandStats{}
		}
		if record.IsRead1() {
			if record.Strand() > 0 {
				s.Pairs.FirstMapPlusGenePlus++
			}
			if record.Strand() < 0 {
				s.Pairs.FirstMapMinusGeneMinus++
			}
		}
		if record.IsRead2() {
			if record.Strand() > 0 {
				s.Pairs.SecondMapPlusGenePlus++
			}
			if record.Strand() < 0 {
				s.Pairs.SecondMapMinusGeneMinus++
			}
		}
		return
	}
	if s.Reads == nil {
		s.Reads = &SingleStrandStats{}
	}
	if record.Strand() > 0 {
		s.Reads.MapPlusGenePlus++
	}
	if record.Strand() < 0 {
		s.Reads.MapMinusGeneMinus++
	}
}

// IncrementAntisense increment counts for antisense reads
func (s *StrandStats) IncrementAntisense(record *sam.Record) {
	if record.IsPaired() {
		if s.Pairs == nil {
			s.Pairs = &PairedStrandStats{}
		}
		if record.IsRead1() {
			if record.Strand() > 0 {
				s.Pairs.FirstMapPlusGeneMinus++
			}
			if record.Strand() < 0 {
				s.Pairs.FirstMapMinusGenePlus++
			}
		}
		if record.IsRead2() {
			if record.Strand() > 0 {
				s.Pairs.SecondMapPlusGeneMinus++
			}
			if record.Strand() < 0 {
				s.Pairs.SecondMapMinusGenePlus++
			}
		}
		return
	}
	if s.Reads == nil {
		s.Reads = &SingleStrandStats{}
	}
	if record.Strand() > 0 {
		s.Reads.MapPlusGeneMinus++
	}
	if record.Strand() < 0 {
		s.Reads.MapMinusGenePlus++
	}
}

// Type returns the stats type
func (s *StrandStats) Type() string {
	return "strand"
}

// Update updates all counts from another Stats instance.
func (s *StrandStats) Update(other Stats) {
	if other, ok := other.(*StrandStats); ok {
		if other.Reads != nil {
			if s.Reads == nil {
				s.Reads = &SingleStrandStats{}
			}
			s.Reads.Update(other.Reads)
		}
		if other.Pairs != nil {
			if s.Pairs == nil {
				s.Pairs = &PairedStrandStats{}
			}
			s.Pairs.Update(other.Pairs)
		}
		s.Total += other.Total
		s.Failed += other.Failed
		s.Finalize()
	}
}

// Merge update counts from a channel of Stats instances.
func (s *StrandStats) Merge(others chan Stats) {
	for other := range others {
		s.Update(other)
	}
}

// Summary returns specs as string
func (s *StrandStats) Summary() string {
	spec1Label := specLabelMap[s.ReadType()][0]
	spec2Label := specLabelMap[s.ReadType()][1]
	return fmt.Sprintf(`
Total %d usable reads
This is %s data
Fraction of reads failed to determine: %.4f
Fraction of reads explained by "%s": %.4f
Fraction of reads explained by "%s": %.4f
`, s.Total, s.ReadType(), 1-(s.Spec1()+s.Spec2()), spec1Label, s.Spec1(), spec2Label, s.Spec2())
}

// Spec1 returns the sum of counts for SENSE/MATE1_SENSE reads
func (s *StrandStats) Spec1() float64 {
	var sum uint64
	if s.Reads != nil {
		sum = s.Reads.MapPlusGenePlus +
			s.Reads.MapMinusGeneMinus
	} else {
		if s.Pairs != nil {
			sum = s.Pairs.FirstMapPlusGenePlus +
				s.Pairs.FirstMapMinusGeneMinus +
				s.Pairs.SecondMapPlusGeneMinus +
				s.Pairs.SecondMapMinusGenePlus
		}
	}
	return float64(sum) / float64(s.Total)
}

// Spec2 returns the sum of counts for ANTISENSE/MATE2_SENSE reads
func (s *StrandStats) Spec2() float64 {
	var sum uint64
	if s.Reads != nil {
		sum = s.Reads.MapPlusGeneMinus +
			s.Reads.MapMinusGenePlus
	} else {
		if s.Pairs != nil {
			sum = s.Pairs.FirstMapPlusGeneMinus +
				s.Pairs.FirstMapMinusGenePlus +
				s.Pairs.SecondMapPlusGenePlus +
				s.Pairs.SecondMapMinusGeneMinus
		}
	}
	return float64(sum) / float64(s.Total)
}

// ReadType returns the read type
func (s *StrandStats) ReadType() string {
	if s.Reads == nil && s.Pairs == nil {
		return ""
	}
	if s.Reads != nil && s.Pairs != nil {
		return "Mixed"
	}
	if s.Reads != nil {
		return "SingleEnd"
	}
	return "PairedEnd"
}

// Finalize updates dependent counts of a CoverageStats instance.
func (s *StrandStats) Finalize() {
	specLabels, ok := specLabelMap[s.ReadType()]
	if !ok {
		return
	}
	if s.Spec1() > s.threshold {
		s.Strandness = strandnessMap[specLabels[0]]
	} else {
		if s.Spec2() > s.threshold {
			s.Strandness = strandnessMap[specLabels[1]]
		}
	}
}

// NewStrandStats returns a new instance of a StrandStats object
func NewStrandStats(index *annotation.RtreeMap, threshold float64, mapq byte) *StrandStats {
	return &StrandStats{
		index:      index,
		threshold:  threshold,
		mapQ:       mapq,
		Strandness: "NONE",
	}
}
