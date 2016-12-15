// Package bamstats provides functions for computing several mapping statistics on a BAM file.
package bamstats

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/bgzf"
	bgzfidx "github.com/biogo/hts/bgzf/index"
	"github.com/biogo/hts/sam"
)

func init() {
	log.SetLevel(log.WarnLevel)
}

// Stats represents mapping statistics.
type Stats interface {
	Update(other Stats)
	Merge(others chan Stats)
	Collect(record *sam.Record, index *RtreeMap)
	Finalize()
}

type IndexWorkerData struct {
	Ref   *sam.Reference
	Chunk bgzf.Chunk
}

// StatsMap is a map of Stats instances with string keys.
type StatsMap map[string]Stats

var wg sync.WaitGroup

// Merge merges instances of StatsMap
func (sm *StatsMap) Merge(stats chan StatsMap) {
	for s := range stats {
		for key, stat := range *sm {
			if otherStat, ok := s[key]; ok {
				stat.Update(otherStat)
			}
		}
	}
}

func getStatsMap(stats []Stats) StatsMap {
	m := make(StatsMap)
	for _, s := range stats {
		s.Finalize()
		switch s.(type) {
		case *GeneralStats:
			m["general"] = s
		case *CoverageStats:
			if s.(*CoverageStats).uniq {
				m["coverageUniq"] = s
			} else {
				m["coverage"] = s
			}
		}
	}
	return m
}

func workerIndex(id int, fname string, in chan IndexWorkerData, out chan StatsMap, index *RtreeMap, uniq bool) {
	defer wg.Done()
	logger := log.WithFields(log.Fields{
		"Worker": id,
	})
	logger.Debug("Starting")
	f, err := os.Open(fname)
	defer f.Close()
	if err != nil {
		panic(err)
	}
	br, err := bam.NewReader(f, 1)
	defer br.Close()
	if err != nil {
		panic(err)
	}
	stats := []Stats{NewGeneralStats()}
	if index != nil {
		stats = append(stats, NewCoverageStats())
		if uniq {
			cs := NewCoverageStats()
			cs.uniq = true
			stats = append(stats, cs)
		}
	}
	for data := range in {
		logger.WithFields(log.Fields{
			"Reference": data.Ref.Name(),
			"Length":    data.Ref.Len(),
		}).Debugf("Reading reference")
		it, err := bam.NewIterator(br, []bgzf.Chunk{data.Chunk})
		defer it.Close()
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			it.Close()
			panic(err)
		}
		for it.Next() {
			for _, s := range stats {
				s.Collect(it.Record(), index)
			}
		}
	}
	logger.Debug("Done")

	out <- getStatsMap(stats)
}

func worker(id int, in chan *sam.Record, out chan StatsMap, index *RtreeMap, uniq bool) {
	defer wg.Done()
	logger := log.WithFields(log.Fields{
		"worker": id,
	})
	logger.Debug("Starting")
	stats := []Stats{NewGeneralStats()}
	if index != nil {
		stats = append(stats, NewCoverageStats())
		if uniq {
			cs := NewCoverageStats()
			cs.uniq = true
			stats = append(stats, cs)
		}
	}
	for record := range in {
		for _, s := range stats {
			s.Collect(record, index)
		}
	}
	logger.Debug("Done")

	out <- getStatsMap(stats)
}

func readBAMWithIndex(bamFile string, index *RtreeMap, cpu int, maxBuf int, reads int, uniq bool) (chan StatsMap, error) {
	f, err := os.Open(bamFile)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	br, err := bam.NewReader(f, 1)
	defer br.Close()
	if err != nil {
		return nil, err
	}
	log.Infof("Opening BAM index %s", bamFile+".bai")
	i, err := os.Open(bamFile + ".bai")
	defer i.Close()
	if err != nil {
		return nil, err
	}
	bai, err := bam.ReadIndex(i)
	if err != nil {
		return nil, err
	}
	h := br.Header()
	nRefs := len(h.Refs())
	stats := make(chan StatsMap, cpu)
	chunks := make(chan IndexWorkerData, cpu)
	nWorkers := cpu
	if cpu > nRefs {
		log.WithFields(log.Fields{
			"References": nRefs,
		}).Warnf("Limiting the number of workers to the number of BAM references")
		nWorkers = nRefs
	}
	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go workerIndex(i+1, bamFile, chunks, stats, index, uniq)
	}
	for _, ref := range h.Refs() {
		refChunks, _ := bai.Chunks(ref, 0, ref.Len())
		if err != nil {
			if err != io.EOF && err != bgzfidx.ErrInvalid {
				log.Error(err)
			}
			return nil, err
		}
		if len(refChunks) > 0 {
			if len(refChunks) > 1 {
				log.Debugf("%v: %v chunks", ref.Name(), len(refChunks))
			}
			for _, chunk := range refChunks {
				chunks <- IndexWorkerData{ref, chunk}
			}
		}
	}
	close(chunks)

	return stats, nil
}

func readBAM(bamFile string, index *RtreeMap, cpu int, maxBuf int, reads int, uniq bool) (chan StatsMap, error) {
	f, err := os.Open(bamFile)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	br, err := bam.NewReader(f, cpu)
	defer br.Close()
	if err != nil {
		return nil, err
	}
	input := make([]chan *sam.Record, cpu)
	stats := make(chan StatsMap, cpu)
	for i := 0; i < cpu; i++ {
		wg.Add(1)
		input[i] = make(chan *sam.Record, maxBuf)
		go worker(i+1, input[i], stats, index, uniq)
	}
	c := 0
	for {
		if reads > -1 && c == reads {
			break
		}
		record, err := br.Read()
		if err != nil {
			break
		}
		input[c%cpu] <- record
		c++
	}
	for i := 0; i < cpu; i++ {
		close(input[i])
	}
	return stats, nil
}

// Process process the input BAM file and collect different mapping stats.
func Process(bamFile string, annotation string, cpu int, maxBuf int, reads int, uniq bool) (StatsMap, error) {
	var index *RtreeMap
	if bamFile == "" {
		return nil, errors.New("Please specify a BAM input file")
	}
	if annotation != "" {
		log.Infof("Creating index for %s", annotation)
		start := time.Now()
		index = CreateIndex(annotation, cpu)
		log.Infof("Index done in %v", time.Since(start))
	}
	start := time.Now()
	log.Infof("Collecting stats for %s", bamFile)
	process := readBAMWithIndex
	if _, err := os.Stat(bamFile + ".bai"); os.IsNotExist(err) || cpu == 1 {
		log.Warning("Not using BAM index")
		process = readBAM
	}
	stats, err := process(bamFile, index, cpu, maxBuf, reads, uniq)
	if err != nil {
		return nil, err
	}
	go func() {
		wg.Wait()
		close(stats)
		log.Infof("Stats done in %v", time.Since(start))
	}()
	st := <-stats
	st.Merge(stats)
	return st, nil
}
