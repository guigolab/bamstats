package annotation

import "io"

// Scanner represent a scanner of features
type Scanner struct {
	r    *FeatureReader
	feat *Feature
	err  error
}

// NewScanner returns a new instance of a Scanner
func NewScanner(r io.Reader, chrs map[string]int) *Scanner {
	return &Scanner{
		r: NewFeatureReader(r, chrs),
	}
}

// Next reads the next feature
func (s *Scanner) Next() bool {
	if s.err != nil {
		return false
	}
	s.feat, s.err = s.r.Read()
	return s.err == nil
}

// Error returns the first non-EOF error that was encountered by the Scanner.
func (s *Scanner) Error() error {
	if s.err == io.EOF {
		return nil
	}
	return s.err
}

// Feat returns the current read feature
func (s *Scanner) Feat() *Feature {
	return s.feat
}
