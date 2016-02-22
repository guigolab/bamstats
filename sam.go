package bamstats

import (
	"github.com/biogo/hts/sam"
)

func isSplit(r *sam.Record) bool {
	for _, op := range r.Cigar {
		if op.Type() == sam.CigarSkipped {
			return true
		}
	}
	return false
}

func isPrimary(r *sam.Record) bool {
	return r.Flags&sam.Secondary == 0
}

func getBlocks(r *sam.Record) []location {
	blocks := make([]location, 0, 10)
	ref := r.Ref.Name()
	start := r.Pos
	end := r.Pos
	var con sam.Consume
	for _, co := range r.Cigar {
		coType := co.Type()
		if coType == sam.CigarSkipped {
			blocks = append(blocks, location{ref, start, end})
			start = end + co.Len()
			end = start
			continue
		}
		con = co.Type().Consumes()
		end += co.Len() * con.Reference
		// if con.Query != 0 {
		// 	end = max(end, start)
		// }
	}
	blocks = append(blocks, location{ref, start, end})
	return blocks
}
