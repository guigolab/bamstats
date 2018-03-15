package annotation

// Format type
type Format int

// exported formats
const (
	UNDEF Format = iota - 1
	BED
	GTF
)

// String return the string representation of a Format
func (f Format) String() string {
	switch f {
	case BED:
		return "BED"
	case GTF:
		return "GTF"
	default:
		return "UNKNOWN"
	}
}
