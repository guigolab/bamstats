package bamstats

import (
	log "github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/brentp/bix"
	"os"
)

type ElementStats struct {
	ExonIntron int `json:"exonic_intronic"`
	Intron     int `json:"intron"`
	Exon       int `json:"exon"`
	Intergenic int `json:"intergenic"`
	Total      int `json:"total"`
}

type ReadStats struct {
	Total      ElementStats `json:"Total reads"`
	Continuous ElementStats `json:"Continuous read"`
	Split      ElementStats `json:"Split reads"`
}

func updateCount(r *sam.Record, elems map[string]uint8, st *ElementStats) {
	exons, hasExon := elems["exon"]
	introns, hasIntron := elems["intron"]
	st.Total++
	if _, isIntergenic := elems["intergenic"]; isIntergenic {
		st.Intergenic++
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

func Coverage(bamFile string, annotation string, cpu int) ReadStats {
	stats := ReadStats{}
	f, err := os.Open(bamFile)
	defer f.Close()
	check(err)
	anno, err := bix.New(annotation)
	check(err)
	br, err := bam.NewReader(f, cpu)
	check(err)
	for {
		record, err := br.Read()
		if err != nil {
			break
		}
		if !isPrimary(record) {
			continue
		}
		elements := map[string]uint8{}
		log.Debug(record.Name)
		for _, mappingPosition := range getBlocks(record) {
			log.Debug(mappingPosition)
			eBuf, err := anno.Query(mappingPosition)
			check(err)
			getElements(mappingPosition, eBuf, elements)
		}
		stats.Total.Total++
		if isSplit(record) {
			updateCount(record, elements, &stats.Split)
		} else {
			updateCount(record, elements, &stats.Continuous)
		}
	}
	return stats
}
