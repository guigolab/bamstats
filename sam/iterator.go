package sam

import "github.com/biogo/hts/bam"

type Iterator struct {
	*bam.Iterator
	MaxReads, Reads int
	chr             string
}

func NewIterator(br *bam.Reader, data *RefChunk, reads int) (*Iterator, error) {
	it, err := bam.NewIterator(br, data.Chunks)
	if err != nil {
		return nil, err
	}
	return &Iterator{it, reads, 0, data.Ref.Name()}, nil
}

func (i *Iterator) Next() bool {
	cont := true
	for i.Iterator.Next() {
		if i.chr != i.Record().Ref.Name() {
			continue
		}
		if i.MaxReads >= 0 {
			cont = (i.Reads < i.MaxReads)
		}
		if i.Record().IsPrimary() {
			i.Reads++
		}
		return cont
	}
	return false
}

func (i *Iterator) Record() *Record {
	return NewRecord(i.Iterator.Record())
}
