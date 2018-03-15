package annotation

import (
	"fmt"

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
	return s[i].Start() < s[j].Start()
}

// Feature represents an annotated element.
type Feature struct {
	location     *rtreego.Rect
	chr, element []byte
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

// Bounds returns the location of the feature. It is used within the Rtree.
func (f *Feature) Bounds() *rtreego.Rect {
	return f.location
}

// SetBounds set a new location of the feature
func (f *Feature) SetBounds(newLocation *rtreego.Rect) {
	f.location = newLocation
}

// String returns the string representation of a Feature
func (f *Feature) String() string {
	return fmt.Sprintf("%s:%.0f-%.0f:%s", f.Chr(), f.Start(), f.End(), f.Element())
}

// Out returns the string representation of a Feature
func (f *Feature) Out() string {
	return fmt.Sprintf("%s\t%.0f\t%.0f\t%s", f.Chr(), f.Start(), f.End(), f.Element())
}

// Clone returns a clone of f
func (f *Feature) Clone() *Feature {
	return NewFeature(f.chr, f.element, f.location)
}

// NewFeature returns a new instance of a Feature
func NewFeature(chr, element []byte, rect *rtreego.Rect) *Feature {
	return &Feature{
		rect,
		chr,
		element,
	}
}
