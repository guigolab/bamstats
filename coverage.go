package bamstats

import (
	"math"

	log "github.com/Sirupsen/logrus"
	"github.com/biogo/hts/sam"
)

type ElementStats struct {
	ExonIntron int `json:"exonic_intronic"`
	Intron     int `json:"intron"`
	Exon       int `json:"exon"`
	Intergenic int `json:"intergenic"`
	Other      int `json:"others"`
	Total      int `json:"total"`
}

type ReadStats struct {
	Total      ElementStats `json:"Total reads"`
	Continuous ElementStats `json:"Continuous read"`
	Split      ElementStats `json:"Split reads"`
}

func (s *ReadStats) Update(other Stats) {
	if other, ok := other.(*ReadStats); ok {
		s.Continuous.Update(other.Continuous)
		s.Split.Update(other.Split)
		s.UpdateTotal()
	}
}

func (s *ReadStats) UpdateTotal() {
	s.Total.ExonIntron = s.Continuous.ExonIntron + s.Split.ExonIntron
	s.Total.Exon = s.Continuous.Exon + s.Split.Exon
	s.Total.Intron = s.Continuous.Intron + s.Split.Intron
	s.Total.Intergenic = s.Continuous.Intergenic + s.Split.Intergenic
	s.Total.Other = s.Continuous.Other + s.Split.Other
	s.Total.Total = s.Continuous.Total + s.Split.Total
}

func (s *ReadStats) Merge(others chan Stats) {
	for other := range others {
		if other, ok := other.(*ReadStats); ok {
			s.Update(other)
		}
	}
}

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

func (s *ReadStats) Collect(record *sam.Record, index *RtreeMap) {
	if !isPrimary(record) || isUnmapped(record) {
		return
	}
	elements := map[string]uint8{}
	log.Debug(record.Name)
	for _, mappingPosition := range getBlocks(record) {
		log.Debug(mappingPosition)
		results := QueryIndex(index.Get(mappingPosition.Chrom()), float64(mappingPosition.Start()), float64(mappingPosition.End()), math.MaxFloat64)
		getElements(mappingPosition, &results, elements)
	}
	if isSplit(record) {
		updateCount(record, elements, &s.Split)
	} else {
		updateCount(record, elements, &s.Continuous)
	}
}
