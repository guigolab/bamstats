package bamstats

import (
	"bufio"
	"encoding/gob"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/dhconnelly/rtreego"
)

type RtreeMap map[string]*rtreego.Rtree

type Feature struct {
	location *rtreego.Rect
	chr      string
	Element  string
}

func (f *Feature) Chr() string {
	return f.chr
}

func (f *Feature) Start() float64 {
	return f.location.PointCoord(0)
}

func (f *Feature) End() float64 {
	return f.location.LengthsCoord(0) + f.Start()
}

func (s *Feature) Bounds() *rtreego.Rect {
	return s.location
}

func (t RtreeMap) Get(chr string) *rtreego.Rtree {
	if _, ok := t[chr]; !ok {
		t[chr] = rtreego.NewTree(2, 25, 50)
	}
	return t[chr]
}

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

func QueryIndex(index *rtreego.Rtree, begin, end, max float64) []rtreego.Spatial {
	bb, _ := rtreego.NewRect(rtreego.Point{begin, begin}, []float64{end - begin, end - begin})

	// Get a slice of the objects in rt that intersect bb:
	return index.SearchIntersect(bb)
}

func ReadIndex(fname string) {
	f, err := os.Open(fname)
	defer f.Close()
	if err != nil {
		log.Fatal("error reading file: ", fname)
	}
	trees := make(RtreeMap)
	dec := gob.NewDecoder(f)
	err = dec.Decode(&trees)
	if err != nil {
		log.Fatal("decoding error: ", err)
	}
}

func WriteIndex(fname string) {
	trees := CreateIndex(fname)

	// writing
	f, err := os.Create(fname)
	defer f.Close()
	if err != nil {
		log.Fatal("error creating file: ", fname)
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(trees)
	if err != nil {
		log.Fatal("encode error:", err)
	}
}

// func main() {

// 	trees := CreateIndex(bufio.NewScanner(os.Stdin))

// 	QueryIndex(trees.Get("chr1"), 14711, 14800, 248956422)
// }
