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
	"strconv"
	"unsafe"

	"github.com/dhconnelly/rtreego"
)

const (
	peekLen = 4096
)

// FeatureReader is a struct for readinf features
type FeatureReader struct {
	r            *bufio.Reader
	format       Format
	exons, genes [3]*Feature
	line         int
	chrLens      map[string]int
}

// NewFeatureReader returns a new instance of FeatureReader
func NewFeatureReader(r io.Reader, chrs map[string]int) *FeatureReader {
	br := buffReader(r)
	format := scanFormat(br, peekLen)
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
		if err != nil {
			return nil
		}
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

func parseTags(b []byte) map[string][]byte {
	m := make(map[string][]byte)
	var k string
	for i, tag := range bytes.Split(b, []byte(" ")) {
		if i%2 == 0 {
			k = string(tag)
		} else {
			m[k] = bytes.Trim(tag, `";`)
			if k == "gene_type" {
				break
			}
		}
	}
	return m
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

func readGtf(r *FeatureReader) (f *Feature, err error) {
	var line []byte
	var fields [][]byte
	var element []byte
	for {
		line, err = r.r.ReadBytes('\n')
		//r.line++
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, &csv.ParseError{Err: err}
		}
		line = bytes.TrimSpace(line)
		if skip(line) { // ignore blank lines and comment lines
			continue
		} else {
			fields = bytes.Split(line, []byte{'\t'})
			elem := string(fields[2])
			if elem != "gene" && elem != "exon" {
				continue
			}
			element = fields[2]
			chr := fields[0]
			start := fields[3]
			end := fields[4]
			tags := parseTags(fields[8])
			if _, ok := r.chrLens[string(chr)]; !ok {
				continue
			}
			s, e := parseInterval(start, end)
			f, err = parseFeature(chr, element, s-1, e)
			f.SetTags(tags)
			break
		}
	}
	return
}
