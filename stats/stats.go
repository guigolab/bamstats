package stats

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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
	Type() string
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
func (sm Map) Add(s Stats) {
	sm[s.Type()] = s
}

// OutputJSON writes sm to the wrtier as JSON
func (sm Map) OutputJSON(writer io.Writer) error {
	b, err := json.MarshalIndent(sm, "", "\t")
	if err != nil {
		return err
	}
	writer.Write(b)
	if w, ok := writer.(*bufio.Writer); ok {
		w.Flush()
	}
	return nil
}

//NewMap creates and instance of a stats.Map
func NewMap(stats ...Stats) Map {
	m := make(Map)
	for _, s := range stats {
		m.Add(s)
	}
	return m
}
