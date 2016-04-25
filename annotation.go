package bamstats

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/dhconnelly/rtreego"
)

// RtreeMap is a map of pointers to Rtree with string keys.
type RtreeMap map[string]*rtreego.Rtree

// Feature represents an annotated element.
type Feature struct {
	location *rtreego.Rect
	chr      string
	Element  string
}

// Chr returns the chromosome of the feature
func (f *Feature) Chr() string {
	return f.chr
}

// Start returns the start position of the feature
func (f *Feature) Start() float64 {
	return f.location.PointCoord(0)
}

// End returns the end position of the feature
func (f *Feature) End() float64 {
	return f.location.LengthsCoord(0) + f.Start()
}

// Bounds returns the location of the feature. It is used within the Rtree.
func (f *Feature) Bounds() *rtreego.Rect {
	return f.location
}

// Get returns the pointer to an Rtree for the specified chromosome.
func (t RtreeMap) Get(chr string) *rtreego.Rtree {
	if _, ok := t[chr]; !ok {
		t[chr] = rtreego.NewTree(2, 25, 50)
	}
	return t[chr]
}

// CreateIndex creates the Rtree indices for the specified annotation file. It builds a Rtree
// for each chromosome and returns a RtreeMap having the chromosome names as keys.
func CreateIndex(fname string) *RtreeMap {

	f, err := os.Open(fname)
	defer f.Close()
	check(err)
	reader := bufio.NewScanner(f)

	trees := make(RtreeMap)

	for reader.Scan() {
		line := strings.Split(reader.Text(), "\t")
		chr := line[0]
		element := line[3]
		rt := trees.Get(chr)
		begin, err := strconv.ParseFloat(line[1], 64)
		if err != nil {
			log.Panic("Cannot convert to float64")
		}
		end, err := strconv.ParseFloat(line[2], 64)
		if err != nil {
			log.Panic("Cannot convert to float64")
		}
		loc := rtreego.Point{begin, begin}
		size := end - begin
		rect, err := rtreego.NewRect(loc, []float64{size, size})
		if err != nil {
			log.Panic(err)
		}
		rt.Insert(&Feature{rect, chr, element})
	}

	return &trees
}

// QueryIndex perform a SearchIntersect on the specified index given a start and end position.
func QueryIndex(index *rtreego.Rtree, begin, end float64) []rtreego.Spatial {
	// Create the bounding box for the query:
	bb, _ := rtreego.NewRect(rtreego.Point{begin, begin}, []float64{end - begin, end - begin})

	// Get a slice of the objects in rt that intersect bb:
	return index.SearchIntersect(bb)
}
