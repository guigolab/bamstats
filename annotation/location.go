package annotation

import (
	"math"

	"github.com/dhconnelly/rtreego"
)

type Location struct {
	chrom      string
	start, end float64
}

func NewLocation(chrom string, start, end float64) *Location {
	return &Location{chrom, start, end}
}

func (s *Location) Chrom() string {
	return s.chrom
}
func (s *Location) Start() float64 {
	return s.start
}
func (s *Location) End() float64 {
	return s.end
}

func (loc *Location) GetElements(buf *[]rtreego.Spatial, elems map[string]uint8) {
	for _, feature := range *buf {
		if feature, ok := feature.(*Feature); ok {
			start := math.Max(loc.Start(), feature.Start())
			end := math.Min(loc.End(), feature.End())
			if end <= start {
				continue
			}
			if feature.Element() != "gene" {
				elems[feature.Element()]++
			}
		}
	}
}
