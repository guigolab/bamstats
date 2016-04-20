package bamstats

import (
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
	Collect(record *sam.Record, trees *RtreeMap)
}

var wg sync.WaitGroup

func worker(in chan *sam.Record, out chan Stats, trees *RtreeMap) {
	defer wg.Done()
	var stats Stats
	if trees != nil {
		stats = &ReadStats{}
	} else {
		stats = NewGeneralStats()
	}
	for record := range in {
		stats.Collect(record, trees)
	}
	log.Debug("Worker DONE!")
	out <- stats
}

func ReadBAM(bamFile string, trees *RtreeMap, cpu int, maxBuf int, reads int) chan Stats {
	f, err := os.Open(bamFile)
	defer f.Close()
	check(err)
	br, err := bam.NewReader(f, cpu)
	defer br.Close()
	check(err)
	input := make([]chan *sam.Record, cpu)
	stats := make(chan Stats, cpu)
	for i := 0; i < cpu; i++ {
		wg.Add(1)
		input[i] = make(chan *sam.Record, maxBuf)
		go worker(input[i], stats, trees)
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
	return stats
}

func Process(bamFile string, annotation string, cpu int, maxBuf int, reads int) *Stats {
	var trees *RtreeMap
	if annotation != "" {
		log.Infof("Creating index for %s", annotation)
		start := time.Now()
		trees = CreateIndex(annotation)
		log.Infof("Index done in %v", time.Since(start))
	}
	start := time.Now()
	log.Infof("Collecting stats for %s", bamFile)
	stats := ReadBAM(bamFile, trees, cpu, maxBuf, reads)
	go func() {
		wg.Wait()
		close(stats)
	}()
	log.Infof("Stats done in %v", time.Since(start))
	st := <-stats
	st.Merge(stats)
	return &st
}
