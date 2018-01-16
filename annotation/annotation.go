package annotation

import (
	"bufio"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/dhconnelly/rtreego"
	"github.com/guigolab/bamstats/utils"
)

const (
	DEBUG_ELEMENTS_FILE = "bamstats-coverage.elements.bed"
)

// RtreeMap is a map of pointers to Rtree with string keys.
type RtreeMap map[string]*rtreego.Rtree

// Get returns the pointer to an Rtree for the specified chromosome and create a new Rtree if not present.
func (t RtreeMap) Get(chr string) *rtreego.Rtree {
	if _, ok := t[chr]; !ok {
		t[chr] = rtreego.NewTree(1, 25, 50)
	}
	return t[chr]
}

func insertInTree(sem chan bool, rt *rtreego.Rtree, feats []*Feature) {
	defer func() { <-sem }()
	for _, feat := range feats {
		rt.Insert(feat)
	}
}

func getChrLens(bamFile string, cpu int) (chrs map[string]int) {
	bf, err := os.Open(bamFile)
	utils.Check(err)
	br, err := bam.NewReader(bf, cpu)
	utils.Check(err)
	refs := br.Header().Refs()
	chrs = make(map[string]int, len(refs))
	for _, r := range refs {
		chrs[r.Name()] = r.Len()
	}
	return
}

// CreateIndex creates the Rtree indices for the specified annotation file. It builds a Rtree
// for each chromosome and returns a RtreeMap having the chromosome names as keys.
func CreateIndex(annoFile, bamFile string, cpu int) *RtreeMap {
	f, err := os.Open(annoFile)
	utils.Check(err)
	chrLens := getChrLens(bamFile, cpu)
	scanner := NewScanner(f, chrLens)

	return createIndex(scanner, cpu)
}

func writeElements() {

}

func createIndex(scanner *Scanner, cpu int) *RtreeMap {
	var w *bufio.Writer
	trees := make(RtreeMap)
	regions := make(map[string][]*Feature)
	if logrus.GetLevel() == logrus.DebugLevel {
		out, _ := os.Create(DEBUG_ELEMENTS_FILE)
		w = bufio.NewWriter(out)
	}
	for scanner.Next() {
		feature := scanner.Feat()
		if feature == nil {
			continue
		}
		if logrus.GetLevel() == logrus.DebugLevel {
			w.WriteString(feature.Out())
			w.WriteRune('\n')
		}
		chr := feature.Chr()
		_, ok := regions[chr]
		if !ok {
			var p []*Feature
			regions[chr] = p
		}
		regions[chr] = append(regions[chr], feature)
	}
	if logrus.GetLevel() == logrus.DebugLevel {
		w.Flush()
	}
	if scanner.Error() != nil {
		logrus.Panic(scanner.Error())
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
	bb, _ := rtreego.NewRect(rtreego.Point{begin}, []float64{size})

	// Get a slice of the objects in rt that intersect bb:
	return index.SearchIntersect(bb)
}
