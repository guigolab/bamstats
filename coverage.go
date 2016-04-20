package bamstats

import (
	"bufio"
	"math"
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
	"github.com/brentp/bix"
	"github.com/brentp/irelate"
	I "github.com/brentp/irelate/interfaces"
	"github.com/brentp/irelate/parsers"
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

func updateCount1(r *parsers.Bam, elems map[string]uint8, st *ElementStats) {
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

func worker1(b I.RelatableIterator, anno I.Queryable, out chan ReadStats) {
	// defer wg.Done()
	stats := ReadStats{}
	for record := range irelate.PIRelate(80000, 25000, b, false, func(a I.Relatable) {}, anno) {
		elements := map[string]uint8{}
		if record, ok := record.(*parsers.Bam); ok {
			getElements1(record, record.Related(), elements)
			if isSplit1(record) {
				updateCount1(record, elements, &stats.Split)
			} else {
				updateCount1(record, elements, &stats.Continuous)
			}
		}
	}
	out <- stats
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

func worker2(in chan *sam.Record, out chan ReadStats, trees *RtreeMap) {
	defer wg.Done()
	stats := ReadStats{}
	for record := range in {
		elements := map[string]uint8{}
		log.Debug(record.Name)
		for _, mappingPosition := range getBlocks(record) {
			log.Debug(mappingPosition)
			results := QueryIndex(trees.Get(mappingPosition.Chrom()), float64(mappingPosition.Start()), float64(mappingPosition.End()), math.MaxFloat64)
			getElements2(mappingPosition, &results, elements)
		}
		if isSplit(record) {
			updateCount(record, elements, &stats.Split)
		} else {
			updateCount(record, elements, &stats.Continuous)
		}
	}
	out <- stats
}

func Coverage1(bamFile string, annotation string, cpu int) ReadStats {
	// f, err := os.Open(bamFile)
	// defer f.Close()
	// check(err)
	// br, err := bam.NewReader(f, cpu)
	// if err != nil {
	// 	return ReadStats{}
	// }
	// // hdr := br.Header()
	b, err := parsers.NewBamQueryable(bamFile)
	defer b.Close()
	check(err)
	anno, err := bix.New(annotation)
	defer anno.Close()
	check(err)
	stats := make(chan ReadStats, cpu)

	q, err := b.Query(location{"chr1", 0, 12000})
	check(err)
	worker1(q, anno, stats)
	close(stats)
	st := ReadStats{}
	st.Merge(stats)
	return st
}

func Coverage2(bamFile string, annotation string, cpu int) ReadStats {
	f, err := os.Open(bamFile)
	defer f.Close()
	check(err)
	anno, err := os.Open(annotation)
	defer anno.Close()
	check(err)
	start := time.Now()
	log.Info("Creating index for ", annotation)
	trees := CreateIndex(bufio.NewScanner(anno))
	elapsed := time.Since(start)
	log.Infof("Indexing done in %v", elapsed)
	start = time.Now()
	br, err := bam.NewReader(f, cpu)
	defer br.Close()
	check(err)
	input := make([]chan *sam.Record, cpu)
	stats := make(chan ReadStats, cpu)
	for i := 0; i < cpu; i++ {
		wg.Add(1)
		input[i] = make(chan *sam.Record, 1000000)
		go worker2(input[i], stats, trees)
	}
	c := 0
	for {
		record, err := br.Read()
		if err != nil {
			break
		}
		if !isPrimary(record) || isUnmapped(record) {
			continue
		}
		input[c%cpu] <- record
		c++
	}
	for i := 0; i < cpu; i++ {
		close(input[i])
	}
	go func() {
		wg.Wait()
		close(stats)
	}()
	st := ReadStats{}
	st.Merge(stats)
	elapsed = time.Since(start)
	log.Infof("Stats done in %v", elapsed)
	return st
}

func Coverage(bamFile string, annotation string, cpu int) ReadStats {
	f, err := os.Open(bamFile)
	defer f.Close()
	check(err)
	anno, err := bix.New(annotation, cpu)
	defer anno.Close()
	check(err)
	br, err := bam.NewReader(f, cpu)
	defer br.Close()
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
		c = (c + 1) % cpu
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
