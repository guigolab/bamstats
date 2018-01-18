// Copyright ©2011-2013 The bíogo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filter

import (
	"github.com/biogo/biogo/alphabet"
	"github.com/biogo/biogo/index/kmerindex"
	"github.com/biogo/biogo/seq/linear"

	"sort"
)

const (
	diagonalPadding = 2
)

// A Merger aggregates and clips an ordered set of trapezoids.
type Merger struct {
	target, query              *linear.Seq
	filterParams               *Params
	maxIGap                    int
	leftPadding, bottomPadding int
	binWidth                   int
	selfComparison             bool
	freeTraps, trapList        *trapezoid
	trapOrder, tail            *trapezoid
	eoTerm                     *trapezoid
	trapCount                  int
	valueToCode                alphabet.Index
}

// Create a new Merger using the provided kmerindex, query sequence, filter parameters and maximum inter-segment gap length.
// If selfCompare is true only the upper diagonal of the comparison matrix is examined.
func NewMerger(ki *kmerindex.Index, query *linear.Seq, filterParams *Params, maxIGap int, selfCompare bool) *Merger {
	tubeWidth := filterParams.TubeOffset + filterParams.MaxError
	binWidth := tubeWidth - 1
	leftPadding := diagonalPadding + binWidth

	eoTerm := &trapezoid{Trapezoid: Trapezoid{
		Left:   query.Len() + 1 + leftPadding,
		Right:  query.Len() + 1,
		Bottom: -1,
		Top:    query.Len() + 1,
	}}

	return &Merger{
		target:         ki.Seq(),
		filterParams:   filterParams,
		maxIGap:        maxIGap,
		query:          query,
		selfComparison: selfCompare,
		bottomPadding:  ki.K() + 2,
		leftPadding:    leftPadding,
		binWidth:       binWidth,
		eoTerm:         eoTerm,
		trapOrder:      eoTerm,
		valueToCode:    ki.Seq().Alpha.LetterIndex(),
	}
}

// Merge a filter hit into the collection.
func (m *Merger) MergeFilterHit(h *Hit) {
	Left := -h.Diagonal
	if m.selfComparison && Left <= m.filterParams.MaxError {
		return
	}
	Top := h.To
	Bottom := h.From

	var temp, free *trapezoid
	for base := m.trapOrder; ; base = temp {
		temp = base.next
		switch {
		case Bottom-m.bottomPadding > base.Top:
			if free == nil {
				m.trapOrder = temp
			} else {
				free.join(temp)
			}
			m.trapList = base.join(m.trapList)
			m.trapCount++
		case Left-diagonalPadding > base.Right:
			free = base
		case Left+m.leftPadding >= base.Left:
			if Left+m.binWidth > base.Right {
				base.Right = Left + m.binWidth
			}
			if Left < base.Left {
				base.Left = Left
			}
			if Top > base.Top {
				base.Top = Top
			}

			if free != nil && free.Right+diagonalPadding >= base.Left {
				free.Right = base.Right
				if free.Bottom > base.Bottom {
					free.Bottom = base.Bottom
				}
				if free.Top < base.Top {
					free.Top = base.Top
				}

				free.join(temp)
				m.freeTraps = base.join(m.freeTraps)
			} else if temp != nil && temp.Left-diagonalPadding <= base.Right {
				base.Right = temp.Right
				if base.Bottom > temp.Bottom {
					base.Bottom = temp.Bottom
				}
				if base.Top < temp.Top {
					base.Top = temp.Top
				}
				base.join(temp.next)
				m.freeTraps = temp.join(m.freeTraps)
				temp = base.next
			}

			return
		default:
			if m.freeTraps == nil {
				m.freeTraps = &trapezoid{}
			}
			if free == nil {
				m.trapOrder = m.freeTraps
			} else {
				free.join(m.freeTraps)
			}

			free, m.freeTraps = m.freeTraps.decapitate()
			free.join(base)

			free.Top = Top
			free.Bottom = Bottom
			free.Left = Left
			free.Right = Left + m.binWidth

			return
		}
	}
}

func (m *Merger) clipVertical() {
	for base := m.trapList; base != nil; base = base.next {
		lagPosition := base.Bottom - m.maxIGap + 1
		if lagPosition < 0 {
			lagPosition = 0
		}
		lastPosition := base.Top + m.maxIGap
		if lastPosition > m.query.Len() {
			lastPosition = m.query.Len()
		}

		var pos int
		for pos = lagPosition; pos < lastPosition; pos++ {
			if m.valueToCode[m.query.Seq[pos]] >= 0 {
				if pos-lagPosition >= m.maxIGap {
					if lagPosition-base.Bottom > 0 {
						if m.freeTraps == nil {
							m.freeTraps = &trapezoid{}
						}

						m.freeTraps = m.freeTraps.prependFrontTo(base)

						base.Top = lagPosition
						base = base.next
						base.Bottom = pos
						m.trapCount++
					} else {
						base.Bottom = pos
					}
				}
				lagPosition = pos + 1
			}
		}
		if pos-lagPosition >= m.maxIGap {
			base.Top = lagPosition
		}
	}
}

func (m *Merger) clipTrapezoids() {
	for base := m.trapList; base != nil; base = base.next {
		if base.Top-base.Bottom < m.bottomPadding-2 {
			continue
		}

		aBottom := base.Bottom - base.Right
		aTop := base.Top - base.Left

		lagPosition := aBottom - m.maxIGap + 1
		if lagPosition < 0 {
			lagPosition = 0
		}
		lastPosition := aTop + m.maxIGap
		if lastPosition > m.target.Len() {
			lastPosition = m.target.Len()
		}

		lagClip := aBottom
		var pos int
		for pos = lagPosition; pos < lastPosition; pos++ {
			if m.valueToCode[m.target.Seq[pos]] >= 0 {
				if pos-lagPosition >= m.maxIGap {
					if lagPosition > lagClip {
						if m.freeTraps == nil {
							m.freeTraps = &trapezoid{}
						}

						m.freeTraps = m.freeTraps.prependFrontTo(base)

						base.clip(lagPosition, lagClip)

						base = base.next
						m.trapCount++
					}
					lagClip = pos
				}
				lagPosition = pos + 1
			}
		}

		if pos-lagPosition < m.maxIGap {
			lagPosition = aTop
		}

		base.clip(lagPosition, lagClip)

		m.tail = base
	}
}

// Finalise the merged collection and return a sorted slice of Trapezoids.
func (m *Merger) FinaliseMerge() Trapezoids {
	var next *trapezoid
	for base := m.trapOrder; base != m.eoTerm; base = next {
		next = base.next
		m.trapList = base.join(m.trapList)
		m.trapCount++
	}

	m.clipVertical()
	m.clipTrapezoids()

	if m.tail != nil {
		m.freeTraps = m.tail.join(m.freeTraps)
	}

	traps := make(Trapezoids, m.trapCount)
	for i, z := 0, m.trapList; i < m.trapCount; i++ {
		traps[i] = z.Trapezoid
		z, z.next = z.next, nil
	}

	sort.Sort(traps)

	return traps
}
