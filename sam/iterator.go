package sam

import "github.com/biogo/hts/bam"

type Iterator struct {
	*bam.Iterator
	MaxReads, Reads int
	Chr             string
}

func NewIterator(br *bam.Reader, data *RefChunk, reads int) (*Iterator, error) {
	it, err := bam.NewIterator(br, data.Chunks)
	if err != nil {
		return nil, err
	}
	return &Iterator{it, reads, 0, data.Ref.Name()}, nil
}

func (i *Iterator) Next() bool {
	next := i.Iterator.Next()
	cont := true
	if next && (i.MaxReads >= 0) {
		if i.Record().IsPrimary() {
			i.Reads++
		}
		cont = (i.Reads < i.MaxReads)
	}
	return next && cont
}

func (i *Iterator) Record() *Record {
	return NewRecord(i.Iterator.Record())
}
