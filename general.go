package bamstats

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"github.com/biogo/hts/sam"
)

type TagMap map[int]int

type MappedReadsStats struct {
	Total    int    `json:"total,omitempty"`
	Unmapped int    `json:"unmapped,omitempty"`
	Mapped   TagMap `json:"mapped,omitempty"`
}

type MappingsStats struct {
	MappedReadsStats
	Continuous int           `json:"continuous"`
	Split      int           `json:"split"`
	Mappings   MultimapStats `json:"mappings"`
}

type MappedPairsStats struct {
	MappedReadsStats
	InsertSizes TagMap `json:"insert_sizes,omitempty"`
}

type MultimapStats struct {
	Ratio float64 `json:"ratio"`
	Count int     `json:"count"`
}

type GeneralStats struct {
	Reads    MappingsStats    `json:"reads,omitempty"`
	Pairs    MappedPairsStats `json:"pairs,omitempty"`
	Coverage *CoverageStats   `json:"coverage,omitempty"`
}

func (s *GeneralStats) Merge(others chan Stats) {
	for other := range others {
		if other, ok := other.(*GeneralStats); ok {
			s.Update(other)
		}
	}
}

func (s *GeneralStats) Update(other Stats) {
	if other, ok := other.(*GeneralStats); ok {
		s.Reads.Update(other.Reads)
		s.Pairs.Update(other.Pairs)
		if s.Coverage != nil {
			s.Coverage.Update(other.Coverage)
		}
		if len(s.Pairs.Mapped) > 0 {
			s.Pairs.Total = s.Reads.Total / 2
			s.Pairs.Unmapped = s.Pairs.Total - s.Pairs.Mapped.Total()
		}
	}
}

func (s *MappedReadsStats) Update(other MappedReadsStats) {
	s.Total += other.Total
	s.Unmapped += other.Unmapped
	s.Mapped.Update(other.Mapped)
}

func (s *MappingsStats) Update(other MappingsStats) {
	s.MappedReadsStats.Update(other.MappedReadsStats)
	s.Continuous += other.Continuous
	s.Split += other.Split
	s.Mappings.Count += other.Mappings.Count
	s.UpdateMappingsRatio()
}

func (s *MappedPairsStats) Update(other MappedPairsStats) {
	s.MappedReadsStats.Update(other.MappedReadsStats)
	s.InsertSizes.Update(other.InsertSizes)
}

func (s *MappedPairsStats) FilterInsertSizes(percent float64) {
	for k, v := range s.InsertSizes {
		if float64(v) < float64(s.Total)*(percent/100) {
			delete(s.InsertSizes, k)
		}
	}
}

func (s *MappingsStats) UpdateMappingsRatio() {
	s.Mappings.Ratio = float64(s.Mappings.Count) / float64(s.Mapped.Total())
}

func (s *MappedReadsStats) Unique() int {
	return s.Mapped[1]
}

func (tm TagMap) Update(other TagMap) {
	for k := range tm {
		tm[k] += other[k]
	}
	for k := range other {
		if _, ok := tm[k]; !ok {
			tm[k] += other[k]
		}
	}
}

func (tm TagMap) Total() (sum int) {
	for _, v := range tm {
		sum += v
	}
	return
}

func NewGeneralStats() *GeneralStats {
	ms := GeneralStats{}
	ms.Pairs = *NewMappedPairsStats()
	ms.Reads.MappedReadsStats = *NewMappedReadsStats()
	return &ms
}

func NewMappedReadsStats() *MappedReadsStats {
	s := MappedReadsStats{}
	s.Mapped = make(TagMap)
	return &s
}

func NewMappedPairsStats() *MappedPairsStats {
	s := MappedPairsStats{}
	s.MappedReadsStats = *NewMappedReadsStats()
	s.InsertSizes = make(TagMap)
	return &s
}

func (tm TagMap) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{'{', '\n'})
	var keys []int
	for k := range tm {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	l := len(keys)
	for i, k := range keys {
		fmt.Fprintf(buf, "\t\"%d\": \"%v\"", k, tm[k])
		if i < l-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.Write([]byte{'}', '\n'})
	return buf.Bytes(), nil
}

func (s *GeneralStats) Collect(r *sam.Record, index *RtreeMap) {
	NH, hasNH := r.Tag([]byte("NH"))
	if !hasNH {
		NH, _ = sam.ParseAux([]byte("NH:i:0"))
	}
	NHKey := int(NH.Value().(uint8))
	if isUnmapped(r) {
		s.Reads.Total++
		s.Reads.Unmapped++
		return
	}
	s.Reads.Mappings.Count++
	if isPrimary(r) {
		s.Reads.Total++
		s.Reads.Mapped[NHKey]++
		if isSplit(r) {
			s.Reads.Split++
		} else {
			s.Reads.Continuous++
		}
		if isFirstOfValidPair(r) {
			s.Pairs.Mapped[NHKey]++
			isLen := int(math.Abs(float64(r.TempLen)))
			s.Pairs.InsertSizes[isLen]++
		}
	}
}
