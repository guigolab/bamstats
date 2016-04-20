package bamstats

import "github.com/biogo/hts/sam"

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

func isUnmapped(r *sam.Record) bool {
	return r.Flags&sam.Unmapped == sam.Unmapped
}

func isPaired(r *sam.Record) bool {
	return r.Flags&sam.Paired == sam.Paired
}

func isProperlyPaired(r *sam.Record) bool {
	return r.Flags&sam.ProperPair == sam.ProperPair
}

func isRead1(r *sam.Record) bool {
	return r.Flags&sam.Read1 == sam.Read1
}

func isRead2(r *sam.Record) bool {
	return r.Flags&sam.Read2 == sam.Read2
}

func hasMateUnmapped(r *sam.Record) bool {
	return r.Flags&sam.MateUnmapped == sam.MateUnmapped
}

func isFirstOfValidPair(r *sam.Record) bool {
	return isPaired(r) && isRead1(r) && isProperlyPaired(r) && !hasMateUnmapped(r)
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
