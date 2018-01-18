// Copyright ©2011-2013 The bíogo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate ./genCode.sh

// Package align provide basic sequence alignment types and helpers.
package align

import (
	"github.com/biogo/biogo/alphabet"
	"github.com/biogo/biogo/feat"
	"github.com/biogo/biogo/seq"

	"errors"
	"fmt"
)

type AlphabetSlicer interface {
	Alphabet() alphabet.Alphabet
	Slice() alphabet.Slice
}

// An Aligner aligns the sequence data of two type-matching Slicers, returning an ordered
// slice of features describing matching and mismatching segments. The sequences to be aligned
// must have a valid gap letter in the first position of their alphabet; the alphabets
// {DNA,RNA}{gapped,redundant} and Protein provided by the alphabet package satisfy this.
type Aligner interface {
	Align(reference, query AlphabetSlicer) ([]feat.Pair, error)
}

// A Linear is a basic linear gap penalty alignment description.
// It is a square scoring matrix with the first column and first row specifying gap penalties.
type Linear [][]int

// An Affine is a basic affine gap penalty alignment description.
type Affine struct {
	Matrix  Linear
	GapOpen int
}

var (
	_ Aligner = SW{}
	_ Aligner = NW{}
)

const (
	diag = iota
	up
	left

	gap = 0

	minInt = -int(^uint(0)>>1) - 1
)

var (
	ErrMismatchedTypes     = errors.New("align: mismatched sequence types")
	ErrMismatchedAlphabets = errors.New("align: mismatched alphabets")
	ErrNoAlphabet          = errors.New("align: no alphabet")
	ErrNotGappedAlphabet   = errors.New("align: alphabet does not have gap at position 0")
	ErrTypeNotHandled      = errors.New("align: sequence type not handled")
	ErrMatrixNotSquare     = errors.New("align: scoring matrix is not square")
)

type ErrMatrixWrongSize struct {
	Size int // size of the matrix
	Len  int // length of the alphabet
}

func (e ErrMatrixWrongSize) Error() string {
	return fmt.Sprintf("align: scoring matrix size %d does not match alphabet length %d", e.Size, e.Len)
}

func max3(a, b, c int) int {
	if b > a {
		a = b
	}
	if c > a {
		return c
	}
	return a
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func add(a, b int) int {
	if a == minInt || b == minInt {
		return minInt
	}
	return a + b
}

type feature struct {
	start, end int
	loc        feat.Feature
}

func (f feature) Name() string {
	if f.loc != nil {
		return f.loc.Name()
	}
	return ""
}
func (f feature) Description() string {
	if f.loc != nil {
		return f.loc.Description()
	}
	return ""
}
func (f feature) Location() feat.Feature { return f.loc }
func (f feature) Start() int             { return f.start }
func (f feature) End() int               { return f.end }
func (f feature) Len() int               { return f.end - f.start }

type featPair struct {
	a, b  feature
	score int
}

func (fp *featPair) Features() [2]feat.Feature { return [2]feat.Feature{fp.a, fp.b} }
func (fp *featPair) Score() int                { return fp.score }
func (fp *featPair) Invert()                   { fp.a, fp.b = fp.b, fp.a }
func (fp *featPair) String() string {
	switch {
	case fp.a.start == fp.a.end:
		return fmt.Sprintf("-/%s[%d,%d)=%d",
			fp.b.Name(), fp.b.start, fp.b.end,
			fp.score)
	case fp.b.start == fp.b.end:
		return fmt.Sprintf("%s[%d,%d)/-=%d",
			fp.a.Name(), fp.a.start, fp.a.end,
			fp.score)
	}
	return fmt.Sprintf("%s[%d,%d)/%s[%d,%d)=%d",
		fp.a.Name(), fp.a.start, fp.a.end,
		fp.b.Name(), fp.b.start, fp.b.end,
		fp.score)
}

// Format returns a [2]alphabet.Slice representing the formatted alignment of a and b described by the
// list of feature pairs in f, with gap used to fill gaps in the alignment.
func Format(a, b seq.Slicer, f []feat.Pair, gap alphabet.Letter) [2]alphabet.Slice {
	var as, aln [2]alphabet.Slice
	for i, s := range [2]seq.Slicer{a, b} {
		as[i] = s.Slice()
		aln[i] = as[i].Make(0, 0)
	}
	for _, fs := range f {
		fc := fs.Features()
		for i := range aln {
			if fc[i].Len() == 0 {
				switch aln[i].(type) {
				case alphabet.Letters:
					aln[i] = aln[i].Append(alphabet.Letters(gap.Repeat(fc[1-i].Len())))
				case alphabet.QLetters:
					aln[i] = aln[i].Append(alphabet.QLetters(alphabet.QLetter{L: gap}.Repeat(fc[1-i].Len())))
				}
			} else {
				aln[i] = aln[i].Append(as[i].Slice(fc[i].Start(), fc[i].End()))
			}
		}
	}
	return aln
}
