package annotation

import (
	"fmt"
	"math"

	"github.com/dhconnelly/rtreego"
)

// Location represents a genomic region
type Location struct {
	chrom      string
	start, end float64
}

// NewLocation returns a new Location instance
func NewLocation(chrom string, start, end int) *Location {
	return &Location{chrom, float64(start), float64(end)}
}

// Chrom returns the location chromosome
func (loc *Location) Chrom() string {
	return loc.chrom
}

// Start returns the locations start position
func (loc *Location) Start() float64 {
	return loc.start
}

// End returns the location end position
func (loc *Location) End() float64 {
	return loc.end
}

func (loc *Location) String() string {
	return fmt.Sprintf("%s:%d-%d", loc.chrom, int(loc.start), int(loc.end))
}

// GetElements returns all elements overlapping with buf
func (loc *Location) GetElements(buf []rtreego.Spatial, elems map[string]uint8, tags ...string) {
	for _, feature := range buf {
		if feature, ok := feature.(*Feature); ok {
			start := math.Max(loc.Start(), feature.Start())
			end := math.Min(loc.End(), feature.End())
			if end <= start {
				continue
			}

			if feature.Element() != "gene" {
				elems[feature.Element()]++
			} else {
				for _, t := range tags {
					elems[feature.Tag(t)]++
				}
			}
		}
	}
}
