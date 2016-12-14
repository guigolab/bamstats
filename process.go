// Package bamstats provides functions for computing several mapping statistics on a BAM file.
package bamstats

import (
	"errors"
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
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

func worker(in chan *sam.Record, out chan StatsMap, index *RtreeMap, uniq bool) {
	defer wg.Done()
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
	log.Debug("Worker DONE!")

	out <- getStatsMap(stats)
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
		go worker(input[i], stats, index, uniq)
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
	stats, err := readBAM(bamFile, index, cpu, maxBuf, reads, uniq)
	if err != nil {
		return nil, err
	}
	go func() {
		wg.Wait()
		close(stats)
	}()
	log.Infof("Stats done in %v", time.Since(start))
	st := <-stats
	st.Merge(stats)
	return st, nil
}
