package sam

import (
	"github.com/biogo/hts/sam"
	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/utils"
)

type Record struct {
	*sam.Record
}

// Export original sam.Record functions
var (
	NewTag = sam.NewTag
	NewAux = sam.NewAux
)

func NewRecord(r *sam.Record) *Record {
	return &Record{r}
}

func (r *Record) IsUniq() bool {
	NH, hasNH := r.Tag([]byte("NH"))
	if !hasNH {
		return false
	}
	NHval := NH.Value().(uint8)
	return NHval == 1
}

func (r *Record) IsSplit() bool {
	for _, op := range r.Cigar {
		if op.Type() == sam.CigarSkipped {
			return true
		}
	}
	return false
}

func (r *Record) IsPrimary() bool {
	return r.Flags&sam.Secondary == 0
}

func (r *Record) IsUnmapped() bool {
	return r.Flags&sam.Unmapped == sam.Unmapped
}

func (r *Record) IsPaired() bool {
	return r.Flags&sam.Paired == sam.Paired
}

func (r *Record) IsProperlyPaired() bool {
	return r.Flags&sam.ProperPair == sam.ProperPair
}

func (r *Record) IsRead1() bool {
	return r.Flags&sam.Read1 == sam.Read1
}

func (r *Record) IsRead2() bool {
	return r.Flags&sam.Read2 == sam.Read2
}

func (r *Record) HasMateUnmapped() bool {
	return r.Flags&sam.MateUnmapped == sam.MateUnmapped
}

func (r *Record) IsFirstOfValidPair() bool {
	return r.IsPaired() && r.IsRead1() && r.IsProperlyPaired() && !r.HasMateUnmapped()
}

func (r *Record) IsDuplicate() bool {
	return r.Flags&sam.Duplicate == sam.Duplicate
}

func (r *Record) IsQCFail() bool {
	return r.Flags&sam.QCFail == sam.QCFail
}

func (r *Record) GetBlocks() []*annotation.Location {
	blocks := make([]*annotation.Location, 0, 10)
	ref := r.Ref.Name()
	start := r.Pos
	end := r.Pos
	var con sam.Consume
	for _, co := range r.Cigar {
		coType := co.Type()
		if coType == sam.CigarSkipped {
			blocks = append(blocks, annotation.NewLocation(ref, start, end))
			start = end + co.Len()
			end = start
			continue
		}
		con = co.Type().Consumes()
		end += co.Len() * con.Reference
		if con.Query != 0 {
			end = utils.Max(end, start)
		}
	}
	blocks = append(blocks, annotation.NewLocation(ref, start, end))
	return blocks
}
