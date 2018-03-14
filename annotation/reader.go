package annotation

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"unsafe"

	"github.com/Sirupsen/logrus"
	"github.com/dhconnelly/rtreego"
	"github.com/guigolab/bamstats/utils"
)

const (
	PEEKLEN = 4096
)

type FeatureReader struct {
	r            *bufio.Reader
	format       Format
	exons, genes [3]*Feature
	line         int
	chrLens      map[string]int
}

func NewFeatureReader(r io.Reader, chrs map[string]int) *FeatureReader {
	br := buffReader(r)
	format := scanFormat(br, PEEKLEN)
	return &FeatureReader{
		r:       br,
		format:  format,
		chrLens: chrs,
	}
}

// CheckBytes peeks at a buffered stream and checks if the first read bytes match.
func CheckBytes(b *bufio.Reader, buf []byte) (bool, error) {
	m, err := b.Peek(len(buf))
	if err != nil {
		return false, err
	}
	for i := range buf {
		if m[i] != buf[i] {
			return false, nil
		}
	}
	return true, nil
}

// IsGzip returns true buffered Reader has the gzip magic
func isGzip(b *bufio.Reader) (bool, error) {
	return CheckBytes(b, []byte{0x1f, 0x8b})
}

// IsGzip returns true buffered Reader has the gzip magic
func isBzip2(b *bufio.Reader) (bool, error) {
	return CheckBytes(b, []byte{0x42, 0x5a})
}

func buffReader(r io.Reader) *bufio.Reader {

	br := bufio.NewReader(r)
	if isGz, err := isGzip(br); err != nil {
		log.Fatal(err)
	} else if isGz {
		rdr, err := gzip.NewReader(br)
		utils.Check(err)
		br = bufio.NewReader(rdr)
	} else if isBz, err := isBzip2(br); err != nil {
		log.Fatal(err)
	} else if isBz {
		rdr := bzip2.NewReader(br)
		br = bufio.NewReader(rdr)
	}

	return br
}

func isTab(r rune) bool {
	return r == '\t'
}

func isNewLine(r rune) bool {
	return r == '\n'
}

func scanFormat(r *bufio.Reader, n int) (format Format) {
	b, err := r.Peek(n)
	if err != nil {
		if err != io.EOF {
			panic(err)
		}
	}
	lines := bytes.FieldsFunc(b, isNewLine)
scan:
	for i, line := range lines {
		if line[0] == '#' {
			continue
		}
		if i == len(lines)-1 && !isNewLine(rune(line[len(line)-1])) {
			log.Fatal("Cannot guess type. Try increasing the peek buffer.")
		}
		switch c := bytes.Count(line, []byte{'\t'}); c + 1 {
		case 4:
			format = BED
			break scan
		case 9:
			format = GTF
			break scan
		default:
			format = UNDEF
		}
	}
	return
}

// This function cannot be used to create strings that are expected to persist.
func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func (r *FeatureReader) Read() (f *Feature, err error) {
	switch r.format {
	case BED:
		f, err = readBed(r)
	case GTF:
		f, err = readGtf(r)
	default:
		err = fmt.Errorf("FeatureReader, %s format error", r.format)
	}
	return
}

func skip(line []byte) bool {
	if len(line) == 0 {
		return true
	}
	if bytes.HasPrefix(line, []byte{'#'}) {
		return true
	}
	return false
}

func parseInterval(b, e []byte) (begin, end float64) {
	var err error
	begin, err = strconv.ParseFloat(unsafeString(b), 64)
	if err != nil {
		return -1, -1
	}
	end, err = strconv.ParseFloat(unsafeString(e), 64)
	if err != nil {
		return -1, -1
	}
	return
}

func parseFeature(chr, element []byte, begin, end float64) (*Feature, error) {
	loc := rtreego.Point{begin}
	size := end - begin
	rect, err := rtreego.NewRect(loc, []float64{size})
	if err != nil {
		return nil, err
	}
	return NewFeature(chr, element, rect), nil
}

func readBed(r *FeatureReader) (f *Feature, err error) {
	var line []byte
	for {
		line, err = r.r.ReadBytes('\n')
		//r.line++
		if err != nil {
			if err == io.EOF {
				return f, err
			}
			return nil, &csv.ParseError{Err: err}
		}
		line = bytes.TrimSpace(line)
		if skip(line) { // ignore blank lines and comment lines
			continue
		} else {
			break
		}
	}
	fields := bytes.Split(line, []byte{'\t'})
	chr := fields[0]
	start := fields[1]
	end := fields[2]
	element := fields[3]

	s, e := parseInterval(start, end)

	return parseFeature(chr, element, s, e)
}

func selectFeature(f *Feature, list *[3]*Feature, element []byte, extremes bool, chrLens map[string]int) (feat *Feature, err error) {
	if (*list)[0] != nil && ((*list)[1] == nil || (*list)[0].Chr() != (*list)[1].Chr()) {
		if extremes {
			feat, err = parseFeature([]byte(f.Chr()), element, f.End(), float64(chrLens[f.Chr()]))
			(*list)[0] = nil // previous <- nil
			return
		}
		(*list)[0] = nil // previous <- nil
	}
	if (*list)[0] != nil {
		// return new element between previous and current
		feat, err = parseFeature([]byte(f.Chr()), element, (*list)[0].End(), (*list)[1].Start())
		(*list)[2] = (*list)[1] // next <- current
		(*list)[1] = f          // current <- feature just read
		(*list)[0] = nil        // no previous

	} else {
		if extremes && isFirst(*list) && (*list)[1] != nil && (*list)[1].Start() > 0 {
			// return new element between 0 and current
			feat, err = parseFeature([]byte(f.Chr()), element, 0, (*list)[1].Start())
			(*list)[2] = (*list)[1] // next <- current
			(*list)[1] = f          // current <- feature just read
			(*list)[0] = nil        // no previous
		} else {
			(*list)[0] = (*list)[1]          // previous <- current
			feat, (*list)[1] = (*list)[1], f // return current, current <- feature just read
		}
	}
	return
}

func isFirst(list [3]*Feature) bool {
	return list[0] == nil && list[2] == nil
}

func readGtf(r *FeatureReader) (f *Feature, err error) {
	var line []byte
	var fields [][]byte
	var element []byte
	for {
		if r.genes[2] != nil {
			f, r.genes[2], r.genes[0], err = r.genes[2], nil, r.genes[2], nil
			if r.genes[1] == nil || r.genes[1].Chr() != r.genes[0].Chr() {
				f, err = selectFeature(r.genes[0], &r.genes, []byte("intergenic"), true, r.chrLens)
			}
			break
		}
		if r.exons[2] != nil {
			f, r.exons[2], r.exons[0], err = r.exons[2], nil, r.exons[2], nil
			if r.exons[1] != nil && r.exons[1].Chr() != r.exons[0].Chr() {
				r.exons[0] = nil
			}
			break
		}
		line, err = r.r.ReadBytes('\n')
		//r.line++
		if err != nil {
			if err == io.EOF {
				if r.genes[1] != nil {
					f, err = selectFeature(r.genes[1], &r.genes, []byte("intergenic"), true, r.chrLens)
					r.genes[1] = nil
					break
				}
				if r.exons[1] != nil {
					f, err = selectFeature(r.exons[1], &r.exons, []byte("intron"), false, r.chrLens)
					if f.Element() == "intron" && r.genes[0] != nil && r.genes[1] != nil {
						if f.Start() == r.genes[0].End() || f.End() == r.genes[1].Start() || f.End() == r.genes[0].Start() {
							f = nil
						}
					}
					r.exons[1] = nil
					break
				}
				return
			}
			return nil, &csv.ParseError{Err: err}
		}
		line = bytes.TrimSpace(line)
		if skip(line) { // ignore blank lines and comment lines
			continue
		} else {
			fields = bytes.Split(line, []byte{'\t'})
			element = fields[2]
			chr := fields[0]
			start := fields[3]
			end := fields[4]
			if _, ok := r.chrLens[string(chr)]; !ok {
				continue
			}
			if bytes.Equal(element, []byte("gene")) {
				s, e := parseInterval(start, end)
				f, err = parseFeature(chr, element, s-1, e)
				if r.genes[1] == nil {
					r.genes[1] = f
				} else {
					if f.Chr() != r.genes[1].Chr() {
						tmp := f
						f, err = selectFeature(r.genes[1], &r.genes, []byte("intergenic"), true, r.chrLens)
						r.genes[1] = tmp
						break
					}
					if f.Start() < r.genes[1].Start() {
						logrus.Fatal("Annotation is not sorted")
					}
					if f.Start() <= r.genes[1].End() {
						r.genes[1], err = parseFeature(chr, element, r.genes[1].Start(), float64(utils.Max(int(f.End()), int(r.genes[1].End()))))
					} else {
						f, err = selectFeature(f, &r.genes, []byte("intergenic"), true, r.chrLens)
						break
					}
				}
			} else if bytes.Equal(element, []byte("exon")) {
				s, e := parseInterval(start, end)
				f, err = parseFeature(chr, element, s-1, e)
				if r.exons[1] == nil {
					r.exons[1] = f
				} else {
					if f.Chr() != r.exons[1].Chr() {
						tmp := f
						f, err = selectFeature(r.exons[1], &r.exons, []byte("intron"), false, r.chrLens)
						if f.Element() == "intron" {
							if (r.genes[0] != nil && r.genes[1] != nil && f.Start() == r.genes[0].End() || f.End() == r.genes[1].Start()) || (r.genes[0] != nil && f.End() == r.genes[0].Start()) {
								f = nil
							}
						}
						r.exons[1] = tmp
						break
					}
					if f.Start() < r.exons[1].Start() {
						logrus.Fatal("Annotation is not sorted")
					}
					if f.Start() <= r.exons[1].End() {
						r.exons[1], err = parseFeature(chr, element, r.exons[1].Start(), float64(utils.Max(int(f.End()), int(r.exons[1].End()))))
					} else {
						f, err = selectFeature(f, &r.exons, []byte("intron"), false, r.chrLens)
						if f.Element() == "intron" {
							if (r.genes[0] != nil && r.genes[1] != nil && f.Start() == r.genes[0].End() || f.End() == r.genes[1].Start()) || (r.genes[0] != nil && f.End() == r.genes[0].Start()) {
								f = nil
							}
						}
						break
					}
				}
			}
		}
	}
	return
}

func mergeIntervals(intervals []*Feature) []*Feature {
	sort.Sort(FeatureSlice(intervals))
	out := make([]*Feature, 0)
	x, intervals := intervals[0], intervals[1:]
	for n, i := range intervals {
		if i.Start() <= x.End() {
			loc := rtreego.Point{x.Start()}
			size := float64(utils.Max(int(i.End()), int(x.End()))) - x.Start()
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

func insertBetweenIntervals(intervals []*Feature, element []byte, extremes bool) []*Feature {
	sort.Sort(FeatureSlice(intervals))
	out := make([]*Feature, 0)
	var start, end float64
	for n, i := range intervals {
		if n == 0 && extremes || n > 0 {
			end = i.Start()
			loc := rtreego.Point{start}
			size := end - start
			rect, err := rtreego.NewRect(loc, []float64{size})
			if err != nil {
				log.Panic(err)
			}
			out = append(out, &Feature{rect, i.chr, element})
		}
		start = i.End()
		out = append(out, i)
	}
	return out
}

// func createIndex(scanner *bufio.Scanner, cpu int) *RtreeMap {
// 	trees := make(RtreeMap)
// 	regions := make(map[string][]*Feature)

// 	for scanner.Scan() {
// 		if scanner.Text()[0] == byte('#') {
// 			// line := strings.Split(scanner.Text(), "\t")
// 			continue
// 		}
// 		isTab := func(r rune) bool {
// 			return r == '\t'
// 		}
// 		line := strings.FieldsFunc(scanner.Text(), isTab)
// 		feature := parseBed(line)
// 		if feature == nil {
// 			continue
// 		}
// 		chr := feature.Chr()
// 		_, ok := regions[chr]
// 		if !ok {
// 			var p []*Feature
// 			regions[chr] = p
// 		}
// 		regions[chr] = append(regions[chr], feature)
// 	}

// 	sem := make(chan bool, cpu)
// 	for chr := range regions {
// 		sem <- true
// 		go insertInTree(sem, trees.Get(chr), regions[chr])
// 	}
// 	for i := 0; i < cap(sem); i++ {
// 		sem <- true
// 	}

// 	return &trees
// }

// func createIndexFromGtf(scanner *bufio.Scanner, cpu int) *RtreeMap {
// 	trees := make(RtreeMap)
// 	genes, exons := make(map[string][]*Feature), make(map[string][]*Feature)

// 	for scanner.Scan() {
// 		// line := strings.Split(scanner.Text(), "\t")
// 		if scanner.Bytes() == nil {
// 			continue
// 		}
// 		if scanner.Text()[0] == byte('#') {
// 			continue
// 		}
// 		isTab := func(r rune) bool {
// 			return r == '\t'
// 		}
// 		line := strings.FieldsFunc(scanner.Text(), isTab)
// 		feature := parseBed(line)
// 		if feature == nil {
// 			continue
// 		}
// 		chr := feature.Chr()
// 		if feature.Element == "exon" {
// 			_, ok := exons[chr]
// 			if !ok {
// 				var p []*Feature
// 				exons[chr] = p
// 			}
// 			exons[chr] = append(exons[chr], feature)
// 		} else if feature.Element == "gene" {
// 			_, ok := genes[chr]
// 			if !ok {
// 				var p []*Feature
// 				genes[chr] = p
// 			}
// 			genes[chr] = append(genes[chr], feature)
// 		}
// 	}

// 	sem := make(chan bool, cpu)
// 	for chr := range genes {
// 		sem <- true
// 		go func(chr string) {
// 			exons := insertBetweenIntervals(mergeIntervals(exons[chr]), "intron", false)
// 			genes := insertBetweenIntervals(mergeIntervals(genes[chr]), "intergenic", true)
// 			regions := append(exons, genes...)
// 			insertInTree(sem, trees.Get(chr), regions)
// 		}(chr)
// 	}
// 	for i := 0; i < cap(sem); i++ {
// 		sem <- true
// 	}

// 	return &trees
// }
