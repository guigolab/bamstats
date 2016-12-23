package stats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// TagMap represents a map of sam tags with integer keys
type TagMap map[interface{}]uint64

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
func (tm TagMap) Total() (sum uint64) {
	for _, v := range tm {
		sum += v
	}
	return
}

// MarshalJSON returns a JSON representation of a TagMap, numerically sorting the keys.
func (tm TagMap) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{'{', '\n'})
	l := len(tm)
	for i, k := range tm.SortedKeys().([]interface{}) {
		fmt.Fprintf(buf, "\t\"%d\": %v", k, tm[k])
		if i < l-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.Write([]byte{'}', '\n'})
	return buf.Bytes(), nil
}

func (tm TagMap) SortedKeys() interface{} {
	l := len(tm)
	keys := make([]interface{}, l)
	i := 0
	for k := range tm {
		keys[i] = k
		break
	}
	switch keys[0].(type) {
	case int:
		convKeys := make([]int, l)
		for k := range tm {
			convKeys[i] = k.(int)
			i++
		}
		sort.Ints(convKeys)
		for j, k := range convKeys {
			keys[j] = k
		}
	case string:
		convKeys := make([]string, l)
		for k := range tm {
			convKeys[i] = k.(string)
			i++
		}
		sort.Strings(convKeys)
		for j, k := range convKeys {
			keys[j] = k
		}
	}
	return keys
}

// UnmarshalJSON parse a JSON representation of a TagMap.
func (tm *TagMap) UnmarshalJSON(b []byte) (err error) {
	smap, imap := make(map[string]uint64), TagMap{}
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
