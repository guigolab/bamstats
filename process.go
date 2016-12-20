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
	bgzfidx "github.com/biogo/hts/bgzf/index"
	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/sam"
	"github.com/guigolab/bamstats/stats"
	"github.com/guigolab/bamstats/utils"
)

func init() {
	log.SetLevel(log.WarnLevel)
}

var wg sync.WaitGroup

func workerIndex(id int, fname string, in chan *sam.RefChunk, out chan stats.StatsMap, index *annotation.RtreeMap, uniq bool) {
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
	sm := stats.NewStatsMap(true, (index != nil), uniq)
	for data := range in {
		logger.WithFields(log.Fields{
			"Reference": data.Ref.Name(),
			"Length":    data.Ref.Len(),
		}).Debugf("Reading reference")
		it, err := bam.NewIterator(br, data.Chunks)
		defer it.Close()
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			it.Close()
			panic(err)
		}
		for it.Next() {
			for _, s := range sm {
				s.Collect(sam.NewRecord(it.Record()), index)
			}
		}
	}
	logger.Debug("Done")

	out <- sm
}

func worker(id int, in chan *sam.Record, out chan stats.StatsMap, index *annotation.RtreeMap, uniq bool) {
	defer wg.Done()
	logger := log.WithFields(log.Fields{
		"worker": id,
	})
	logger.Debug("Starting")

	sm := stats.NewStatsMap(true, (index != nil), uniq)
	for record := range in {
		for _, s := range sm {
			s.Collect(record, index)
		}
	}
	logger.Debug("Done")

	out <- sm
}

func readBAMWithIndex(bamFile string, index *annotation.RtreeMap, cpu int, maxBuf int, reads int, uniq bool) (chan stats.StatsMap, error) {
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
	stats := make(chan stats.StatsMap, cpu)
	chunks := make(chan *sam.RefChunk, cpu)
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
				chunks <- sam.NewRefChunk(ref, chunk)
			}
		}
	}
	close(chunks)

	return stats, nil
}

func readBAM(bamFile string, index *annotation.RtreeMap, cpu int, maxBuf int, reads int, uniq bool) (chan stats.StatsMap, error) {
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
	stats := make(chan stats.StatsMap, cpu)
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
		input[c%cpu] <- sam.NewRecord(record)
		c++
	}
	for i := 0; i < cpu; i++ {
		close(input[i])
	}
	return stats, nil
}

// Process process the input BAM file and collect different mapping stats.
func Process(bamFile string, anno string, cpu int, maxBuf int, reads int, uniq bool) (stats.StatsMap, error) {
	var index *annotation.RtreeMap
	if bamFile == "" {
		return nil, errors.New("Please specify a BAM input file")
	}
	if anno != "" {
		log.Infof("Creating index for %s", anno)
		start := time.Now()
		index = annotation.CreateIndex(anno, cpu)
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

func WriteOutput(output string, st stats.StatsMap) {
	out := utils.NewOutput(output)
	utils.OutputJSON(out, st)
}
