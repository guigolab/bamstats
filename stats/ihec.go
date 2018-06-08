package stats

import (
	"github.com/dhconnelly/rtreego"
	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/sam"
)

// IHECstats represents statistics for mapped reads
type IHECstats struct {
	Intergenic uint64 `json:"intergenic"`
	RRNA       uint64 `json:"rRNA"`
	index      *annotation.RtreeMap
}

// Merge updates counts from a channel of Stats instances.
func (s *IHECstats) Merge(others chan Stats) {
	for other := range others {
		if other, ok := other.(*IHECstats); ok {
			s.Update(other)
		}
	}
}

// Update updates all counts from a Stats instance.
func (s *IHECstats) Update(other Stats) {
	if other, isIHEC := other.(*IHECstats); isIHEC {
		s.Intergenic += other.Intergenic
		s.RRNA += other.RRNA
	}
}

// Finalize updates dependent counts of a Stats instance.
func (s *IHECstats) Finalize() {
}

// Collect collects general mapping statistics from a sam.Record.
func (s *IHECstats) Collect(record *sam.Record) {
	elements := map[string]uint8{}
	if s.index == nil || !record.IsPrimary() || record.IsUnmapped() {
		return
	}
	mappingLocation := annotation.NewLocation(record.Ref.Name(), record.Start(), record.End())
	rtree := s.index.Get(mappingLocation.Chrom())
	if rtree == nil || rtree.Size() == 0 {
		return
	}

	results := annotation.QueryIndex(rtree, mappingLocation.Start(), mappingLocation.End())
	var filteredResults []rtreego.Spatial
	for _, r := range results {
		if r, ok := r.(*annotation.Feature); ok {
			// if r.Element() == "intergenic" {
			// 	if r.End()-r.Start() <= 1000 {
			// 		continue
			// 	}
			// 	if !(mappingLocation.End() >= r.Start()+500 && mappingLocation.Start() <= r.End()-500) {
			// 		continue
			// 	}
			// }
			filteredResults = append(filteredResults, r)
		}
	}
	mappingLocation.GetElements(&filteredResults, elements)

	// if _, isIntergenic := elements["intergenic"]; isIntergenic && len(elements) > 1 {
	// 	fmt.Println(elements)
	// }

	updateIHECcount(elements, s)
}

// NewIHECstats creates a new instance of IHECstats
func NewIHECstats(index *annotation.RtreeMap) *IHECstats {
	return &IHECstats{
		index: index,
	}
}

func updateIHECcount(elems map[string]uint8, st *IHECstats) {

	if len(elems) == 0 {
		return
	}

	if _, isRRNA := elems["rRNA"]; isRRNA {
		st.RRNA++
		return
	}

	if _, isIntergenic := elems["intergenic"]; isIntergenic {
		st.Intergenic++
		return
	}
}
