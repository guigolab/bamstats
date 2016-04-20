package bamstats

import (
	"math"

	log "github.com/Sirupsen/logrus"
	I "github.com/brentp/irelate/interfaces"
	"github.com/brentp/irelate/parsers"
	"github.com/dhconnelly/rtreego"
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

func getElements(pos I.IPosition, buf I.RelatableIterator, elems map[string]uint8) {
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

func getElements1(pos I.IPosition, buf []I.Relatable, elems map[string]uint8) {
	for _, feature := range buf {
		// start := max(pos.Start(), feature.Start())
		// end := min(pos.End(), feature.End())
		// if end <= start {
		// 	continue
		// }
		log.Debug(feature)
		if interval, ok := feature.(*parsers.Interval); ok {
			t := string(interval.Fields[3])
			if t != "gene" {
				elems[t]++
			}
		}
	}
}

func getElements2(pos I.IPosition, buf *[]rtreego.Spatial, elems map[string]uint8) {
	for _, feature := range *buf {
		if feature, ok := feature.(*Feature); ok {
			start := math.Max(float64(pos.Start()), feature.Start())
			end := math.Min(float64(pos.End()), feature.End())
			if end <= start {
				continue
			}
			if feature.Element != "gene" {
				elems[feature.Element]++
			}
		}
	}
}
