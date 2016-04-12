package bamstats

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/sam"
)

type TagMap map[int]int

type MappedReadsStats struct {
	Total    int           `json:"total"`
	Mapped   TagMap        `json:"mapped"`
	Unmapped int           `json:"unmapped"`
	Mappings MultimapStats `json:"mappings"`
}

type MappedPairsStats struct {
	Read1  TagMap `json:"read1"`
	Read2  TagMap `json:"read2"`
	Mapped int    `json:"mapped"`
}

type MultimapStats struct {
	Count int     `json:"count"`
	Ratio float64 `json:"ratio"`
}

type MappingStats struct {
	Reads MappedReadsStats `json:"reads"`
	Pairs MappedPairsStats `json:"pairs"`
}

func (s *MappingStats) Merge(others chan MappingStats) {
	for other := range others {
		s.Update(other)
	}
}

func (s *MappingStats) Update(other MappingStats) {
	s.Reads.Update(other.Reads)
	s.Pairs.Update(other.Pairs)
}

func (s *MappedReadsStats) Update(other MappedReadsStats) {
	s.Total += other.Total
	s.Unmapped += other.Unmapped
	s.Mappings.Count += other.Mappings.Count
	s.Mapped.Update(other.Mapped)
	s.UpdateMappingsRatio()
}

func (s *MappedReadsStats) UpdateMappingsRatio() {
	s.Mappings.Ratio = float64(s.Mappings.Count) / float64(s.Mapped.Total())
}

func (s *MappedReadsStats) Unique() int {
	return s.Mapped[1]
}

func (s *MappedPairsStats) Update(other MappedPairsStats) {
	s.Mapped += other.Mapped
	s.Read1.Update(other.Read1)
	s.Read2.Update(other.Read2)
}

func (s TagMap) Update(other TagMap) {
	for k := range s {
		s[k] += other[k]
	}
	for k := range other {
		if _, ok := s[k]; !ok {
			s[k] += other[k]
		}
	}
}

func (s TagMap) Total() (sum int) {
	for _, v := range s {
		sum += v
	}
	return
}

func NewMappingStats() *MappingStats {
	ms := MappingStats{}
	ms.Pairs.Read1 = make(TagMap)
	ms.Pairs.Read2 = make(TagMap)
	ms.Reads.Mapped = make(TagMap)
	return &ms
}

func (tm TagMap) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{'{', '\n'})
	var keys []int
	for k := range tm {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	l := len(keys)
	for i, k := range keys {
		fmt.Fprintf(buf, "\t\"%d\": \"%v\"", k, tm[k])
		if i < l-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.Write([]byte{'}', '\n'})
	return buf.Bytes(), nil
}

func (s *MappingStats) Collect(r *sam.Record) {
	NH, hasNH := r.Tag([]byte("NH"))
	if !hasNH {
		NH, _ = sam.ParseAux([]byte("NH:i:0"))
	}
	NHKey := int(NH.Value().(uint8))
	if isPrimary(r) {
		s.Reads.Total++
		if isUnmapped(r) {
			s.Reads.Unmapped++
			return
		}
		s.Reads.Mappings.Count++
		s.Reads.Mapped[NHKey]++
		if isPaired(r) {
			if isRead1(r) {
				s.Pairs.Read1[NHKey]++
				if isProperlyPaired(r) && !hasMateUnmapped(r) {
					s.Pairs.Mapped++
				}
			}
			if isRead2(r) {
				s.Pairs.Read2[NHKey]++
			}
		}
	} else {
		if !isUnmapped(r) {
			s.Reads.Mappings.Count++
		}
	}
}

var wgg sync.WaitGroup

func gworker(in chan *sam.Record, out chan MappingStats) {
	defer wgg.Done()
	stats := NewMappingStats()
	for record := range in {
		stats.Collect(record)
	}
	log.Debug("Worker DONE!")
	out <- *stats
}

func cProc(br *bam.Reader, cpu int, maxBuf int) *MappingStats {
	input := make([]chan *sam.Record, cpu)
	stats := make(chan MappingStats, cpu)
	for i := 0; i < cpu; i++ {
		wgg.Add(1)
		input[i] = make(chan *sam.Record, maxBuf)
		go gworker(input[i], stats)
	}
	c := 0
	for {
		record, err := br.Read()
		if err != nil {
			break
		}
		input[c] <- record
		c = (c + 1) % cpu
	}
	for i := 0; i < cpu; i++ {
		close(input[i])
	}
	wgg.Wait()
	close(stats)
	st := <-stats
	st.Merge(stats)
	return &st
}

func lProc(br *bam.Reader) *MappingStats {
	st := NewMappingStats()
	for {
		record, err := br.Read()
		if err != nil {
			break
		}
		st.Collect(record)
	}
	return st
}

func General(bamFile string, cpu int, maxBuf int) *MappingStats {
	f, err := os.Open(bamFile)
	defer f.Close()
	check(err)
	br, err := bam.NewReader(f, cpu)
	defer br.Close()
	check(err)
	return cProc(br, cpu, maxBuf)
}
