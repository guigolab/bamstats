// Package bamstats provides functions for computing several mapping statistics on a BAM file.
package bamstats

import (
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
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

func worker(id int, in interface{}, out chan stats.Map, index *annotation.RtreeMap, cfg *config.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	logger := log.WithFields(log.Fields{
		"worker": id,
	})
	logger.Debug("Starting")

	sm := makeStatsMap(index, cfg)

	collectStats(in, sm)

	logger.Debug("Done")

	out <- sm
}

func collectStats(in interface{}, sm stats.Map) {
	switch in.(type) {
	case chan *sam.Record:
		c := in.(chan *sam.Record)
		for record := range c {
			for _, s := range sm {
				s.Collect(record)
			}
		}
	case chan *sam.Iterator:
		iterators := in.(chan *sam.Iterator)
		for it := range iterators {
			for it.Next() {
				for _, s := range sm {
					s.Collect(it.Record())
				}
			}
		}
	}
}

func process(bamFile string, index *annotation.RtreeMap, cpu int, maxBuf int, reads int, uniq bool) (stats.Map, error) {

	var wg sync.WaitGroup

	conf := config.NewConfig(cpu, maxBuf, reads, uniq)

	br, err := sam.NewReader(bamFile, conf)
	defer br.Close()
	if err != nil {
		return nil, err
	}
	statChan := make(chan stats.Map, cpu)
	for i := 0; i < br.Workers; i++ {
		id := i + 1
		wg.Add(1)
		go worker(id, br.Channels[i], statChan, index, conf, &wg)
	}

	go br.Read()

	go waitProcess(statChan, &wg)
	stat := <-statChan
	stat.Merge(statChan)
	for k, v := range stat {
		v.Finalize()
		if k == "general" {
			s := v.(*stats.GeneralStats)
			s.Reads.Total += br.Unmapped()
		}
	}

	return stat, nil
}

func waitProcess(st chan stats.Map, wg *sync.WaitGroup) {
	wg.Wait()
	close(st)
}

func getChrLens(bamFile string, cpu int) (chrs map[string]int) {
	bf, err := os.Open(bamFile)
	utils.Check(err)
	br, err := bam.NewReader(bf, cpu)
	utils.Check(err)
	refs := br.Header().Refs()
	chrs = make(map[string]int, len(refs))
	for _, r := range refs {
		chrs[r.Name()] = r.Len()
	}
	return
}

// Process process the input BAM file and collect different mapping stats.
func Process(bamFile string, anno string, cpu int, maxBuf int, reads int, uniq bool) (stats.Map, error) {
	var index *annotation.RtreeMap
	if anno != "" {
		log.Infof("Creating index for %s", anno)
		start := time.Now()
		chrLens := getChrLens(bamFile, cpu)
		index = annotation.CreateIndex(anno, chrLens)
		log.Infof("Index done in %v", time.Since(start))
	}
	start := time.Now()
	log.Infof("Collecting stats for %s", bamFile)
	allStats, err := process(bamFile, index, cpu, maxBuf, reads, uniq)
	if err != nil {
		return nil, err
	}
	log.Infof("Stats done in %v", time.Since(start))
	return allStats, nil
}

func WriteOutput(output string, st stats.Map) {
	out := utils.NewOutput(output)
	utils.OutputJSON(out, st)
}

func makeStatsMap(index *annotation.RtreeMap, cfg *config.Config) stats.Map {
	m := make(stats.Map)
	m.Add("general", stats.NewGeneralStats())
	if index != nil {
		m.Add("coverage", stats.NewCoverageStats(index, false))
		if cfg.Uniq {
			m.Add("coverageUniq", stats.NewCoverageStats(index, true))
		}
		m.Add("rnaseq", stats.NewIHECstats(index))
	}
	return m
}
