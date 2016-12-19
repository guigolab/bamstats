package stats

import (
	"github.com/guigolab/bamstats/annotation"
	"github.com/guigolab/bamstats/sam"
)

// Stats represents mapping statistics.
type Stats interface {
	Update(other Stats)
	Merge(others chan Stats)
	Collect(record *sam.Record, index *annotation.RtreeMap)
	Finalize()
}

// StatsMap is a map of Stats instances with string keys.
type StatsMap map[string]Stats

// Merge merges instances of StatsMap
func (sm *StatsMap) Merge(stats chan StatsMap) {
	for s := range stats {
		for key, stat := range *sm {
			if otherStat, ok := s[key]; ok {
				stat.Update(otherStat)
			}
		}
	}
}

func NewStatsMap(general, coverage, uniq bool) StatsMap {
	m := make(StatsMap)
	if general {
		m["general"] = NewGeneralStats()
	}
	if coverage {
		m["coverage"] = NewCoverageStats()
	}
	if uniq {
		cs := NewCoverageStats()
		cs.uniq = true
		m["coverageUniq"] = cs
	}
	return m
}
