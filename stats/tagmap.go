package stats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// TagMap represents a map of sam tags with integer keys
type TagMap map[int]int

// Update updates all counts from another TagMap instance.
func (tm TagMap) Update(other TagMap) {
	for k := range tm {
		tm[k] += other[k]
	}
	for k := range other {
		if _, ok := tm[k]; !ok {
			tm[k] += other[k]
		}
	}
}

// Total returns the total number of reads in the TagMap
func (tm TagMap) Total() (sum int) {
	for _, v := range tm {
		sum += v
	}
	return
}

// MarshalJSON returns a JSON representation of a TagMap, numerically sorting the keys.
func (tm TagMap) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{'{', '\n'})
	var keys []int
	for k := range tm {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	l := len(keys)
	for i, k := range keys {
		fmt.Fprintf(buf, "\t\"%d\": %v", k, tm[k])
		if i < l-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.Write([]byte{'}', '\n'})
	return buf.Bytes(), nil
}

// UnmarshalJSON parse a JSON representation of a TagMap.
func (tm *TagMap) UnmarshalJSON(b []byte) (err error) {
	smap, imap := make(map[string]int), TagMap{}
	if err = json.Unmarshal(b, &smap); err == nil {
		for key, value := range smap {
			// JSON objects have string key - need to convert to int
			if intKey, err := strconv.Atoi(key); err == nil {
				imap[intKey] = value
			}
		}
		*tm = imap
	}
	return
}
