package bamstats

import (
	log "github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/brentp/bix"
	"os"
  "sync"
)

var wg sync.WaitGroup

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

func (s *ReadStats) Update(other ReadStats) {
  s.Continuous.Update(other.Continuous)
  s.Split.Update(other.Split)
  s.UpdateTotal(other)
}

func (s *ReadStats) UpdateTotal(other ReadStats) {
  s.Total.Update(other.Continuous)
  s.Total.Update(other.Split)
}

func (s *ReadStats) Merge(others chan ReadStats) {
  for other := range others {
    s.Update(other)
  }
}

func (s *ElementStats) Update(other ElementStats) {
  s.ExonIntron += other.ExonIntron
  s.Exon += other.Exon
  s.Intron += other.Intron
  s.Intergenic += other.Intergenic
  s.Total += other.Total
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

func worker(in chan *sam.Record, out chan ReadStats, anno *bix.Bix) {
  defer wg.Done()
  stats := ReadStats{}
  for record := range in {
    elements := map[string]uint8{}
		log.Debug(record.Name)
		for _, mappingPosition := range getBlocks(record) {
			log.Debug(mappingPosition)
			eBuf, err := anno.Query(mappingPosition)
			check(err)
			getElements(mappingPosition, eBuf, elements)
		}
		if isSplit(record) {
			updateCount(record, elements, &stats.Split)
		} else {
			updateCount(record, elements, &stats.Continuous)
		}
	}
  out <- stats
}

func Coverage(bamFile string, annotation string, cpu int) ReadStats {
	f, err := os.Open(bamFile)
	defer f.Close()
	check(err)
	anno, err := bix.New(annotation, cpu)
	check(err)
	br, err := bam.NewReader(f, cpu)
	check(err)
  input := make([]chan *sam.Record, cpu)
  stats := make(chan ReadStats, cpu)
  for i := 0; i < cpu; i++ {
    wg.Add(1)
    input[i] = make(chan *sam.Record)
    go worker(input[i], stats, anno)
  }
  c := 0
	for {
    record, err := br.Read()
		if err != nil {
			break
		}
		if !isPrimary(record) {
			continue
		}
    input[c] <- record
    c = (c+1)%cpu
  }
  for i := 0; i < cpu; i++ {
    close(input[i])
  }
  wg.Wait()
  close(stats)
  st := ReadStats{}
  st.Merge(stats)
  return st
}
