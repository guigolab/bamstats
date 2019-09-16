package annotation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dhconnelly/rtreego"
)

// FeatureSlice represents a slice of Feature, sortable by start position
type FeatureSlice []*Feature

func (s FeatureSlice) Len() int {
	return len(s)
}
func (s FeatureSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s FeatureSlice) Less(i, j int) bool {
	var ci, cj interface{}
	var err error
	chri := strings.Replace(s[i].Chr(), "chr", "", -1)
	chrj := strings.Replace(s[j].Chr(), "chr", "", -1)
	ci, err = strconv.Atoi(chri)
	if err != nil {
		ci = chri
	}
	cj, err = strconv.Atoi(chrj)
	if err != nil {
		cj = chrj
	}

	m := map[byte]int8{
		'X': 0,
		'Y': 1,
		'M': 2,
	}

	if _, ok := cj.(string); ok {
		js := []byte(cj.(string))[0]
		if _, ok := ci.(string); ok {
			is := []byte(ci.(string))[0]
			if is != js {
				return m[is] < m[js]
			}
		} else {
			return true
		}
	}
	if _, ok := cj.(int); ok {
		if _, ok := ci.(int); ok {
			if ci.(int) != cj.(int) {
				return ci.(int) < cj.(int)
			}
		} else {
			return false
		}
	}
	return s[i].Start() < s[j].Start()
}

// NewFeatureSlice returns a new FeatureSlice instance from a slice of rtreego.Spatial
func NewFeatureSlice(intervals []rtreego.Spatial) FeatureSlice {
	var fs FeatureSlice
	for _, i := range intervals {
		fs = append(fs, i.(*Feature))
	}
	return fs
}

// Feature represents an annotated element.
type Feature struct {
	location     *rtreego.Rect
	chr, element []byte
	score        int
	strand       byte
	tags         map[string][]byte
}

// Chr returns the chromosome of the feature
func (f *Feature) Chr() string {
	return string(f.chr)
}

// Start returns the start position of the feature
func (f *Feature) Start() float64 {
	return f.location.PointCoord(0)
}

// End returns the end position of the feature
func (f *Feature) End() float64 {
	return f.location.LengthsCoord(0) + f.Start()
}

// Element returns the element of the feature
func (f *Feature) Element() string {
	return string(f.element)
}

// Score returns the element of the feature
func (f *Feature) Score() string {
	if f.score < 0 {
		return "."
	}
	return strconv.Itoa(f.score)
}

// Strand returns the element of the feature
func (f *Feature) Strand() string {
	return string(f.strand)
}

// Bounds returns the location of the feature. It is used within the Rtree.
func (f *Feature) Bounds() *rtreego.Rect {
	return f.location
}

// SetBounds set a new location of the feature
func (f *Feature) SetBounds(newLocation *rtreego.Rect) {
	f.location = newLocation
}

// SetTags set feture tags
func (f *Feature) SetTags(tags map[string][]byte) {
	f.tags = tags
}

// Tag get a tag value from f
func (f *Feature) Tag(key string) string {
	return string(f.tags[key])
}

// String returns the string representation of a Feature
func (f *Feature) String() string {
	return fmt.Sprintf("%s:%.0f-%.0f:%s:%s:%s", f.Chr(), f.Start(), f.End(), f.Element(), f.Score(), f.Strand())
}

// Out returns the string representation of a Feature
func (f *Feature) Out() string {
	return fmt.Sprintf("%s\t%.0f\t%.0f\t%s\t%s\t%s", f.Chr(), f.Start(), f.End(), f.Element(), f.Score(), f.Strand())
}

// Clone returns a clone of f
func (f *Feature) Clone() *Feature {
	return NewFeature(f.chr, f.element, f.score, f.strand, f.location)
}

// NewFeature returns a new instance of a Feature
func NewFeature(chr []byte, element []byte, score int, strand byte, rect *rtreego.Rect) *Feature {
	return &Feature{
		rect,
		chr,
		element,
		score,
		strand,
		nil,
	}
}
