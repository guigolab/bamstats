package annotation

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/bamstats/utils"
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

func getFileReader(f *os.File, fname string) *bufio.Scanner {
	var r io.Reader = f

	switch path.Ext(fname) {
	case ".gz":
		zipReader, err := gzip.NewReader(f)
		utils.Check(err)
		r = zipReader
	case ".bz2":
		r = bzip2.NewReader(f)
	}
	return bufio.NewScanner(r)
}

// CreateIndex creates the Rtree indices for the specified annotation file. It builds a Rtree
// for each chromosome and returns a RtreeMap having the chromosome names as keys.
func CreateIndex(fname string, cpu int) *RtreeMap {
	f, err := os.Open(fname)
	defer f.Close()
	utils.Check(err)

	reader := getFileReader(f, fname)

	return createIndex(reader, cpu)
}

func insertInTree(sem chan bool, rt *rtreego.Rtree, feats []*Feature) {
	defer func() { <-sem }()
	for _, feat := range feats {
		rt.Insert(feat)
	}
}

func createIndex(reader *bufio.Scanner, cpu int) *RtreeMap {
	trees := make(RtreeMap)
	regions := make(map[string][]*Feature)

	for reader.Scan() {
		line := strings.Split(reader.Text(), "\t")
		chr := line[0]
		_, ok := regions[chr]
		if !ok {
			var p []*Feature
			regions[chr] = p
		}
		element := line[3]
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
		regions[chr] = append(regions[chr], &Feature{rect, chr, element})
	}

	sem := make(chan bool, cpu)
	for chr := range regions {
		sem <- true
		go insertInTree(sem, trees.Get(chr), regions[chr])
	}
	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	return &trees
}

// QueryIndex perform a SearchIntersect on the specified index given a start and end position.
func QueryIndex(index *rtreego.Rtree, begin, end float64) []rtreego.Spatial {
	size := end - begin
	// Create the bounding box for the query:
	bb, _ := rtreego.NewRect(rtreego.Point{begin, begin}, []float64{size, size})

	// Get a slice of the objects in rt that intersect bb:
	return index.SearchIntersect(bb)
}
