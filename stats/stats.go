package stats

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/guigolab/bamstats/sam"
)

type fraction float64

func (m fraction) String() string {
	return fmt.Sprintf("%.6g", float64(m))
}

func (m fraction) MarshalJSON() ([]byte, error) {
	v, err := strconv.ParseFloat(m.String(), 64)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// Stats represents mapping statistics.
type Stats interface {
	Update(other Stats)
	Merge(others chan Stats)
	Collect(record *sam.Record)
	Finalize()
}

// Map is a map of Stats instances with string keys.
type Map map[string]Stats

// Merge merges instances of StatsMap
func (sm *Map) Merge(stats chan Map) {
	for s := range stats {
		for key, stat := range *sm {
			if otherStat, ok := s[key]; ok {
				stat.Update(otherStat)
			}
		}
	}
}

// Add adds a new Stats object to sm
func (sm Map) Add(key string, s Stats) {
	sm[key] = s
}

// NewMap creates and instance of a stats.Map
// func NewMap(general, coverage, uniq bool) Map {
// 	m := make(Map)
// 	if general {
// 		m["general"] = NewGeneralStats()
// 	}
// 	if coverage {
// 		m["coverage"] = NewCoverageStats()
// 	}
// 	if uniq {
// 		cs := NewCoverageStats()
// 		cs.Uniq = true
// 		m["coverageUniq"] = cs
// 	}
// 	return m
// }
