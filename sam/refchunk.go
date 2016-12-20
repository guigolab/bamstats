package sam

import (
	"github.com/biogo/hts/bgzf"
	"github.com/biogo/hts/sam"
)

type RefChunk struct {
	Ref    *sam.Reference
	Chunks []bgzf.Chunk
}

func NewRefChunk(ref *sam.Reference, chunk bgzf.Chunk) *RefChunk {
	return &RefChunk{ref, []bgzf.Chunk{chunk}}
}
