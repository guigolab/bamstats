package sam

import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/bgzf"
	"github.com/biogo/hts/sam"
	"github.com/guigolab/bamstats/config"
)

type Reader struct {
	*bam.Reader
	FileName string
	Workers  int
	Index    *bam.Index
	Refs     []*sam.Reference
	Channels []interface{}
	cfg      *config.Config
	unmapped uint64
}

func NewReader(bamFile string, cfg *config.Config) (*Reader, error) {
	r, err := NewBamReader(bamFile, cfg)
	if err != nil {
		return nil, err
	}
	h := r.Header()
	index, unmapped := readIndex(bamFile, r, cfg.Cpu)
	workers := cfg.Cpu
	if index != nil {
		nRefs := index.NumRefs()
		if cfg.Cpu > nRefs {
			log.WithFields(log.Fields{
				"References": nRefs,
			}).Warnf("Limiting the number of workers to the number of BAM references")
			workers = nRefs
		}
	}
	chans := make([]interface{}, workers)
	for i := 0; i < workers; i++ {
		if index == nil {
			chans[i] = make(chan *Record, cfg.MaxBuf)
		} else {
			chans[i] = make(chan *Iterator, cfg.MaxBuf)
		}
	}
	return &Reader{
		r,
		bamFile,
		workers,
		index,
		h.Refs(),
		chans,
		cfg,
		unmapped,
	}, nil
}

func NewBamReader(bamFile string, cfg *config.Config) (*bam.Reader, error) {
	f, err := os.Open(bamFile)
	if err != nil {
		return nil, err
	}
	r, err := bam.NewReader(f, cfg.Cpu)
	return r, err
}

func readIndex(bamFile string, br *bam.Reader, cpu int) (*bam.Index, uint64) {
	if _, err := os.Stat(bamFile + ".bai"); err == nil && cpu > 1 {
		log.Infof("Opening BAM index %s", bamFile+".bai")
		i, err := os.Open(bamFile + ".bai")
		defer i.Close()
		if err != nil {
			panic(err)
		}
		bai, err := bam.ReadIndex(i)
		if err != nil {
			panic(err)
		}
		unmapped, _ := bai.Unmapped()
		return bai, unmapped
	} else {
		return nil, 0
	}
}

func (r *Reader) readChromosomes() error {
	var err error
	c := 0
	for _, ref := range r.Refs {
		refStats, ok := r.Index.ReferenceStats(ref.ID())
		if !ok {
			continue
		}
		refChunks := refStats.Chunk

		r.Channels[c%r.Workers].(chan *Iterator) <- r.readChunk(NewRefChunk(ref, []bgzf.Chunk{refChunks}))

		c++
	}
	for i := 0; i < r.Workers; i++ {
		close(r.Channels[i].(chan *Iterator))
	}
	return err
}

func (r *Reader) scan() error {
	c := 0
	reads := r.cfg.Reads
	for {
		if reads > -1 && c == reads {
			break
		}
		record, err := r.Reader.Read()
		if err != nil {
			break
		}
		rec := NewRecord(record)
		if rec.IsUnmapped() {
			r.unmapped++
			continue
		}
		r.Channels[c%r.Workers].(chan *Record) <- rec
		if rec.IsPrimary() {
			c++
		}
	}
	for i := 0; i < r.Workers; i++ {
		close(r.Channels[i].(chan *Record))
	}
	return nil
}

func (r *Reader) Read() {
	if r.Index == nil {
		r.scan()
	} else {
		r.readChromosomes()
	}
}

func (r *Reader) readChunk(data *RefChunk) *Iterator {
	br, err := NewBamReader(r.FileName, r.cfg)
	if err != nil {
		panic(err)
	}
	reads := -1
	if r.cfg.Reads > -1 {
		reads = r.cfg.Reads / len(r.Refs)
		rem := r.cfg.Reads % len(r.Refs)
		if data.Ref.ID() == 0 {
			reads += rem
		}
	}
	count, _ := r.Index.ReferenceStats(data.Ref.ID())
	log.WithFields(log.Fields{
		"Reference": data.Ref.Name(),
		"Length":    data.Ref.Len(),
		"Refs":      len(r.Refs),
		"Reads":     reads,
		"Mapped":    count.Mapped,
		"Unmapped":  count.Unmapped,
	}).Debugf("Reading reference")
	it, err := NewIterator(br, data, reads)
	if err != nil {
		if err != io.EOF {
			log.Println(err)
		}
		it.Close()
		panic(err)
	}
	return it
}

func (r *Reader) Clone() *Reader {
	reader, err := NewReader(r.FileName, r.cfg)
	if err != nil {
		panic(err)
	}
	return reader
}

func (r *Reader) Unmapped() uint64 {
	return r.unmapped
}
