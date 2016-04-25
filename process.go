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

type Stats interface {
	Update(other Stats)
	Merge(others chan Stats)
	Collect(record *sam.Record, index *RtreeMap)
}

var wg sync.WaitGroup

func getStats(stats []Stats) Stats {
	general := stats[0].(*GeneralStats)
	general.Coverage = nil
	if len(stats) > 1 {
		general.Coverage = stats[1].(*CoverageStats)
		general.Coverage.UpdateTotal()
	}
	return general
}

func worker(in chan *sam.Record, out chan Stats, index *RtreeMap) {
	defer wg.Done()
	stats := []Stats{NewGeneralStats()}
	if index != nil {
		stats = append(stats, NewCoverageStats())
	}
	for record := range in {
		for _, s := range stats {
			s.Collect(record, index)
		}
	}
	log.Debug("Worker DONE!")

	out <- getStats(stats)
}

func readBAM(bamFile string, index *RtreeMap, cpu int, maxBuf int, reads int) (chan Stats, error) {
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
	stats := make(chan Stats, cpu)
	for i := 0; i < cpu; i++ {
		wg.Add(1)
		input[i] = make(chan *sam.Record, maxBuf)
		go worker(input[i], stats, index)
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

func Process(bamFile string, annotation string, cpu int, maxBuf int, reads int) (*Stats, error) {
	var index *RtreeMap
	if bamFile == "" {
		return nil, errors.New("Please specify a BAM input file")
	}
	if annotation != "" {
		log.Infof("Creating index for %s", annotation)
		start := time.Now()
		index = CreateIndex(annotation)
		log.Infof("Index done in %v", time.Since(start))
	}
	start := time.Now()
	log.Infof("Collecting stats for %s", bamFile)
	stats, err := readBAM(bamFile, index, cpu, maxBuf, reads)
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
	return &st, nil
}
