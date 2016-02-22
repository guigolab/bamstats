package bamstats

import (
	log "github.com/Sirupsen/logrus"
	"github.com/brentp/irelate/interfaces"
	"github.com/brentp/irelate/parsers"
)

type location struct {
	chrom string
	start int
	end   int
}

func (s location) Chrom() string {
	return s.chrom
}
func (s location) Start() uint32 {
	return uint32(s.start)
}
func (s location) End() uint32 {
	return uint32(s.end)
}

func getElements(pos location, buf interfaces.RelatableIterator, elems map[string]uint8) {
	for {
		feature, err := buf.Next()
		if err != nil {
			break
		}
		start := max(pos.Start(), feature.Start())
		end := min(pos.End(), feature.End())
		if end <= start {
			continue
		}
		log.Debug(feature)
		if interval, ok := feature.(*parsers.Interval); ok {
			t := string(interval.Fields[3])
			if t != "gene" {
				elems[t]++
			}
		}
	}
}
