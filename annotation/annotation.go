package annotation

import (
	"bufio"
	"log"
	"math"
	"os"
	"sort"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/biogo/hts/bam"
	"github.com/dhconnelly/rtreego"
	"github.com/guigolab/bamstats/utils"
)

var (
	debugElementsFile = "bamstats-coverage.elements.bed"
)

type chunk struct {
	chr   string
	feats chan rtreego.Spatial
}

// RtreeMap is a map of pointers to Rtree with string keys.
// type RtreeMap map[string]*rtreego.Rtree
type RtreeMap struct {
	sync.Map
}

// Get returns the pointer to an Rtree for the specified chromosome and create a new Rtree if not present.
func (t *RtreeMap) Get(chr string) *rtreego.Rtree {
	v, ok := t.Load(chr)
	if ok {
		return v.(*rtreego.Rtree)
	}
	return nil
}

// Len returns the number of elements in the map.
func (t *RtreeMap) Len() int {
	var length int
	t.Range(func(_, _ interface{}) bool {
		length++

		return true
	})
	return length
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

func mergeIntervals(intervals []*Feature) []*Feature {
	sort.Sort(FeatureSlice(intervals))
	out := make([]*Feature, 0)
	x, intervals := intervals[0], intervals[1:]
	for n, i := range intervals {
		if i.Start() <= x.End() {
			start := math.Min(x.Start(), i.Start())
			end := math.Max(i.End(), x.End())
			loc := rtreego.Point{start}
			size := end - start
			rect, err := rtreego.NewRect(loc, []float64{size})
			if err != nil {
				log.Panic(err)
			}
			x.SetBounds(rect)
		} else {
			out = append(out, x)
			x = i
		}
		if n == len(intervals)-1 {
			out = append(out, x)
		}
	}
	return out
}

func updateIndex(index *rtreego.Rtree, length float64, feature, updated string, extremes bool) {
	elemFilter := func(elem string) rtreego.Filter {
		return func(results []rtreego.Spatial, object rtreego.Spatial) (refuse, abort bool) {
			f := object.(*Feature)
			if f.Element() != elem {
				return true, false
			}

			return false, false
		}
	}

	r, _ := rtreego.NewRect(rtreego.Point{0}, []float64{length})
	features := index.SearchIntersect(r, elemFilter(feature))

	var w *bufio.Writer
	var out *os.File
	if feature == "gene" {
		out, _ = os.Create(debugElementsFile)
	} else {
		out, _ = os.OpenFile(debugElementsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	}
	w = bufio.NewWriter(out)
	logrus.Debugf("Writing %s elements to %s", feature, out.Name())

	debPrint := func(f *Feature) {
		w.WriteString(f.Out())
		w.WriteRune('\n')
	}

	convert := func(s []rtreego.Spatial) []*Feature {
		var f []*Feature
		for _, i := range s {
			f = append(f, i.(*Feature))
		}
		return f
	}

	merged := mergeIntervals(convert(features))
	for i, f := range merged {
		debPrint(f)
		if extremes {
			if i == 0 {
				n, _ := parseFeature(f.chr, []byte(updated), 0, f.Start())
				debPrint(n)
				index.Insert(n)
			}
			if i == len(merged)-1 {
				n, _ := parseFeature(f.chr, []byte(updated), f.End(), length)
				debPrint(n)
				index.Insert(n)
			}
		}
		if i > 0 {
			g := merged[i-1]
			// if feature == "exon" && f.Tag("gene_id") != g.Tag("gene_id") {
			// 	continue
			// }
			n, _ := parseFeature(f.chr, []byte(updated), g.End(), f.Start())
			debPrint(n)
			index.Insert(n)
		}
	}
	w.Flush()
}

func createTree(trees *RtreeMap, chr string, length float64, feats chan rtreego.Spatial, wg *sync.WaitGroup) {
	wg.Add(1)
	chan2slice := func(c chan rtreego.Spatial) []rtreego.Spatial {
		var s []rtreego.Spatial
		for item := range c {
			s = append(s, item)
		}
		return s
	}
	fs := chan2slice(feats)
	trees.Store(chr, rtreego.NewTree(1, 25, 50, fs...))
	if length > 0 {
		updateIndex(trees.Get(chr), length, "gene", "intergenic", true)
		updateIndex(trees.Get(chr), length, "exon", "intron", false)
	}
	wg.Done()
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

	return createIndex(scanner)
}

func createIndex(scanner *Scanner) *RtreeMap {
	var trees RtreeMap
	regions := make(chan chunk)
	debugElements := make(chan string)

	if logrus.GetLevel() == logrus.DebugLevel {
		go writeElements(debugElements)
	}

	go scan(scanner, regions, debugElements)

	var wg sync.WaitGroup
	for chunk := range regions {
		chr := chunk.chr
		feats := chunk.feats
		length := float64(scanner.r.chrLens[chr])
		go createTree(&trees, chr, length, feats, &wg)
	}
	wg.Wait()

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
