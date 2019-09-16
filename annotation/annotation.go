package annotation

import (
	"bufio"
	"log"
	"math"
	"os"
	"runtime/debug"
	"sort"
	"sync"

	"github.com/dhconnelly/rtreego"
	"github.com/sirupsen/logrus"
)

var (
	debugElementsFile = "bamstats-coverage.elements.bed"
	dumpElementsEnv   = "BAMSTATS_DUMP_ELEMENTS"
)

type chunk struct {
	chr   string
	feats chan rtreego.Spatial
}

type tree struct {
	chr  string
	tree *rtreego.Rtree
}

// RtreeMap is a map of pointers to Rtree with string keys.
// type RtreeMap map[string]*rtreego.Rtree
type RtreeMap map[string]*rtreego.Rtree

// Get returns the pointer to an Rtree for the specified chromosome and create a new Rtree if not present.
func (t RtreeMap) Get(chr string) *rtreego.Rtree {
	v, ok := t[chr]
	if ok {
		return v
	}
	return nil
}

// Len returns the number of elements in the map.
func (t RtreeMap) Len() int {
	return len(t)
}

func scan(scanner *Scanner, regions chan chunk) {
	defer func() {
		close(regions)
		if r := recover(); r != nil {
			logrus.Debugf("%s", debug.Stack())
			os.Exit(1)
		}
	}()

	regMap := make(map[string]chan rtreego.Spatial)
	var chr string
	for scanner.Next() {
		feature := scanner.Feat()
		if feature == nil {
			continue
		}
		chr = feature.Chr()
		_, ok := regMap[chr]
		if !ok {
			regMap[chr] = make(chan rtreego.Spatial)
			regions <- chunk{chr, regMap[chr]}
		}
		regMap[chr] <- feature
	}
	for ref := range regMap {
		close(regMap[ref])
	}
	if scanner.Error() != nil {
		logrus.Panic(scanner.Error())
	}
	nChroms := len(regMap)
	if nChroms == 0 {
		logrus.Panic("Error reading annotation file: no chromosomes found")
	}
	logrus.Infof("Annotation scanned: %d chromosomes found", len(regMap))
}

func writeElements(items <-chan rtreego.Spatial, done chan<- struct{}) {
	var w *bufio.Writer
	out, _ := os.Create(debugElementsFile)
	w = bufio.NewWriter(out)
	logrus.Debugf("Writing index elements to %s", out.Name())
	itemSlice := NewFeatureSlice(chan2slice(items))
	sort.Sort(itemSlice)
	for _, item := range itemSlice {
		w.WriteString(item.Out())
		w.WriteRune('\n')
	}
	w.Flush()
	done <- struct{}{}
}

func mergeIntervals(in []rtreego.Spatial) []*Feature {
	intervals := NewFeatureSlice(in)
	sort.Sort(intervals)
	var out []*Feature
	var x *Feature
	for n, f := range intervals {
		if n == 0 {
			x = f
		}
		if n > 0 {
			if f.Start() <= x.End() {
				start := math.Min(x.Start(), f.Start())
				end := math.Max(f.End(), x.End())
				loc := rtreego.Point{start}
				size := end - start
				rect, err := rtreego.NewRect(loc, []float64{size})
				if err != nil {
					log.Panic(err)
				}
				x.SetBounds(rect)
			} else {
				out = append(out, x)
				x = f
			}
		}
		if n == len(intervals)-1 {
			out = append(out, x)
		}
	}
	return out
}

func interleaveFeatures(features []*Feature, start, end float64, element string, updated []byte, extremes bool) []*Feature {
	var fs FeatureSlice

	for i, f := range features {
		fs = append(fs, f)
		if extremes {
			if i == 0 && f.Start() > start {
				n, _ := parseFeature(f.chr, updated, start, f.Start())
				fs = append(fs, n)
			}
			if i == len(features)-1 && f.End() < end {
				n, _ := parseFeature(f.chr, updated, f.End(), end)
				fs = append(fs, n)
			}
		}
		if i > 0 {
			g := features[i-1]
			n, _ := parseFeature(f.chr, updated, g.End(), f.Start())
			fs = append(fs, n)
		}
	}
	sort.Sort(fs)
	return fs
}

func updateIndex(index *rtreego.Rtree, start, end float64, feature, updated string, extremes bool, elems chan rtreego.Spatial) *rtreego.Rtree {
	if end-start <= 0 {
		return index
	}

	var features []rtreego.Spatial

	genes := QueryIndexByElement(index, start, end, feature)
	for _, i := range genes {
		f := i.(*Feature)
		features = append(features, f)
	}
	mergedGenes := mergeIntervals(genes)
	for _, f := range interleaveFeatures(mergedGenes, start, end, feature, []byte(updated), extremes) {
		if f.Element() == updated {
			features = append(features, f)
			if os.Getenv(dumpElementsEnv) != "" {
				elems <- f
			}
		}
		exons := QueryIndexByElement(index, f.Start(), f.End(), "exon")
		for _, i := range exons {
			f := i.(*Feature)
			features = append(features, f)
		}
		mergedExons := mergeIntervals(exons)
		for _, g := range interleaveFeatures(mergedExons, f.Start(), f.End(), "exon", []byte("intron"), false) {
			if g.Element() == "intron" {
				features = append(features, g)
				if os.Getenv(dumpElementsEnv) != "" {
					elems <- g
				}
			}
		}
	}
	return rtreego.NewTree(1, 25, 50, features...)
}

func chan2slice(c <-chan rtreego.Spatial) []rtreego.Spatial {
	var s []rtreego.Spatial
	for item := range c {
		s = append(s, item)
	}
	return s
}

func createTree(trees chan *tree, chr string, length float64, feats chan rtreego.Spatial, wg *sync.WaitGroup, elems chan rtreego.Spatial) {
	wg.Add(1)
	featSlice := chan2slice(feats)
	tmpIndex := rtreego.NewTree(1, 25, 50, featSlice...)
	t := tree{chr, updateIndex(tmpIndex, 0, length, "gene", "intergenic", true, elems)}
	trees <- &t
	if os.Getenv(dumpElementsEnv) != "" && length == 0 {
		for _, f := range featSlice {
			elems <- f.(*Feature)
		}
	}
	wg.Done()
}

// CreateIndex creates the Rtree indices for the specified annotation file. It builds a Rtree
// for each chromosome and returns a RtreeMap having the chromosome names as keys.
func CreateIndex(annoFile string, chrLens map[string]int) *RtreeMap {
	f, err := os.Open(annoFile)
	if err != nil {
		return nil
	}
	scanner := NewScanner(f, chrLens)

	return createIndex(scanner)
}

func createIndex(scanner *Scanner) *RtreeMap {
	trees := make(RtreeMap)
	regions := make(chan chunk)
	treeChan := make(chan *tree)
	debugElements := make(chan rtreego.Spatial)
	debugElemsDone := make(chan struct{})

	if os.Getenv(dumpElementsEnv) != "" {
		go writeElements(debugElements, debugElemsDone)
	}

	go scan(scanner, regions)

	var wg sync.WaitGroup
	for chunk := range regions {
		chr := chunk.chr
		feats := chunk.feats
		length := float64(scanner.r.chrLens[chr])
		go createTree(treeChan, chr, length, feats, &wg, debugElements)
	}

	go func() {
		wg.Wait()
		close(treeChan)
		close(debugElements)
	}()

	for t := range treeChan {
		trees[t.chr] = t.tree
	}

	if os.Getenv(dumpElementsEnv) != "" {
		<-debugElemsDone
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

// QueryIndexByElement perform a SearchIntersect on the specified index given a start and end position and an element.
func QueryIndexByElement(index *rtreego.Rtree, begin, end float64, element string) []rtreego.Spatial {
	elementFilter := func(elem string) rtreego.Filter {
		return func(results []rtreego.Spatial, object rtreego.Spatial) (refuse, abort bool) {
			f := object.(*Feature)
			if f.Element() != elem {
				return true, false
			}

			return false, false
		}
	}

	size := end - begin
	// Create the bounding box for the query:
	bb, _ := rtreego.NewRect(rtreego.Point{begin}, []float64{size})

	// Get a slice of the objects in rt that intersect bb:
	return index.SearchIntersect(bb, elementFilter(element))
}
