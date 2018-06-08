package annotation

import (
	"bufio"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/dhconnelly/rtreego"
	"github.com/guigolab/bamstats/utils"
)

var (
	debugElementsFile = "bamstats-coverage.elements.bed"
)

// RtreeMap is a map of pointers to Rtree with string keys.
type RtreeMap map[string]*rtreego.Rtree

type chunk struct {
	chr   string
	feats chan rtreego.Spatial
}

// Get returns the pointer to an Rtree for the specified chromosome and create a new Rtree if not present.
func (t RtreeMap) Get(chr string) *rtreego.Rtree {
	return t[chr]
}

func createTrees(trees RtreeMap, regions chan chunk) {
	chan2slice := func(c chan rtreego.Spatial) []rtreego.Spatial {
		var s []rtreego.Spatial
		for item := range c {
			s = append(s, item)
		}
		return s
	}

	for chunk := range regions {
		trees[chunk.chr] = rtreego.NewTree(1, 25, 50, chan2slice(chunk.feats)...)
		logrus.Debugf("Done %s", chunk.chr)
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

func writeElements(items <-chan string) {
	var w *bufio.Writer
	out, _ := os.Create(debugElementsFile)
	w = bufio.NewWriter(out)
	logrus.Debugf("Writing index elements to %s", out.Name())
	for item := range items {
		w.WriteString(item)
		w.WriteRune('\n')
	}
	w.Flush()
}

func scan(scanner *Scanner, regions chan chunk, elems chan string) {
	regMap := make(map[string]chan rtreego.Spatial)
	var chr, lastChr string
	for scanner.Next() {
		feature := scanner.Feat()
		if feature == nil {
			continue
		}
		if logrus.GetLevel() == logrus.DebugLevel {
			elems <- feature.Out()
		}
		if len(chr) == 0 {
			lastChr = feature.Chr()
		}
		chr = feature.Chr()
		if lastChr != chr {
			close(regMap[lastChr])
			lastChr = chr
		}
		_, ok := regMap[chr]
		if !ok {
			regMap[chr] = make(chan rtreego.Spatial)
			regions <- chunk{chr, regMap[chr]}
		}
		regMap[chr] <- feature
	}
	close(regMap[lastChr])
	close(regions)
	close(elems)
	if scanner.Error() != nil {
		logrus.Panic(scanner.Error())
	}
}

func createIndex(scanner *Scanner, cpu int) *RtreeMap {
	trees := make(RtreeMap)
	regions := make(chan chunk)
	debugElements := make(chan string)

	if logrus.GetLevel() == logrus.DebugLevel {
		go writeElements(debugElements)
	}

	go scan(scanner, regions, debugElements)

	createTrees(trees, regions)

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
