// Package bamstats provides functions for computing several mapping statistics on a BAM file.
package bamstats

import (
	// "io"
	// "os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/config"
	"github.com/guigolab/bamstats/sam"
	"github.com/guigolab/bamstats/stats"
	"github.com/guigolab/bamstats/utils"
)

func init() {
	log.SetLevel(log.WarnLevel)
}

// var wg sync.WaitGroup

func worker(id int, in interface{}, out chan stats.StatsMap, index *annotation.RtreeMap, cfg *config.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	logger := log.WithFields(log.Fields{
		"worker": id,
	})
	logger.Debug("Starting")

	sm := stats.NewStatsMap(true, (index != nil), cfg.Uniq)

	switch in.(type) {
	case chan *sam.Record:
		c := in.(chan *sam.Record)
		for record := range c {
			for _, s := range sm {
				s.Collect(record, index)
			}
		}
	case chan *sam.Iterator:
		iterators := in.(chan *sam.Iterator)
		for it := range iterators {
			for it.Next() {
				for _, s := range sm {
					s.Collect(it.Record(), index)
				}
			}
		}
	}
	logger.Debug("Done")

	out <- sm
}

func process(bamFile string, index *annotation.RtreeMap, cpu int, maxBuf int, reads int, uniq bool) (chan stats.StatsMap, error) {

	var wg sync.WaitGroup

	conf := config.NewConfig(cpu, maxBuf, reads, uniq)

	br, err := sam.NewReader(bamFile, conf)
	defer br.Close()
	if err != nil {
		return nil, err
	}
	st := make(chan stats.StatsMap, cpu)
	for i := 0; i < br.Workers; i++ {
		id := i + 1
		wg.Add(1)
		go worker(id, br.Channels[i], st, index, conf, &wg)
	}

	br.Read()

	go waitProcess(st, &wg)

	return st, nil
}

func waitProcess(st chan stats.StatsMap, wg *sync.WaitGroup) {
	wg.Wait()
	close(st)
}

// Process process the input BAM file and collect different mapping stats.
func Process(bamFile string, anno string, cpu int, maxBuf int, reads int, uniq bool) (stats.StatsMap, error) {
	var index *annotation.RtreeMap
	if anno != "" {
		log.Infof("Creating index for %s", anno)
		start := time.Now()
		index = annotation.CreateIndex(anno, cpu)
		log.Infof("Index done in %v", time.Since(start))
	}
	start := time.Now()
	log.Infof("Collecting stats for %s", bamFile)
	stats, err := process(bamFile, index, cpu, maxBuf, reads, uniq)
	if err != nil {
		return nil, err
	}
	st := <-stats
	st.Merge(stats)
	log.Infof("Stats done in %v", time.Since(start))
	return st, nil
}

func WriteOutput(output string, st stats.StatsMap) {
	out := utils.NewOutput(output)
	utils.OutputJSON(out, st)
}
