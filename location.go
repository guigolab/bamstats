package bamstats

import (
	"math"

	"github.com/dhconnelly/rtreego"
)

type location struct {
	chrom      string
	start, end float64
}

func (s location) Chrom() string {
	return s.chrom
}
func (s location) Start() float64 {
	return s.start
}
func (s location) End() float64 {
	return s.end
}

func getElements(loc location, buf *[]rtreego.Spatial, elems map[string]uint8) {
	for _, feature := range *buf {
		if feature, ok := feature.(*Feature); ok {
			start := math.Max(loc.Start(), feature.Start())
			end := math.Min(loc.End(), feature.End())
			if end <= start {
				continue
			}
			if feature.Element != "gene" {
				elems[feature.Element]++
			}
		}
	}
}
