package dimension

import (
	"bytes"
	"sort"

	"github.com/petethepig/pyroscope/pkg/storage/segment"
)

type Dimension struct {
	// keys are sorted
	keys []segment.Key
}

func New() *Dimension {
	return &Dimension{
		keys: []segment.Key{},
	}
}

func (d *Dimension) Insert(key segment.Key) {
	i := sort.Search(len(d.keys), func(i int) bool {
		return bytes.Compare(d.keys[i], key) >= 0
	})

	if i < len(d.keys) && bytes.Compare(d.keys[i], key) == 0 {
		return
	}

	if i > len(d.keys)-1 || !bytes.Equal(d.keys[i], key) {
		d.keys = append(d.keys, key)
		copy(d.keys[i+1:], d.keys[i:])
		d.keys[i] = key
	}
}

type advanceResult int

const (
	match advanceResult = iota
	noMatch
	end
)

type sortableDim struct {
	keys []segment.Key
	i    int
	l    int
}

func (sd *sortableDim) current() segment.Key {
	return sd.keys[sd.i]
}

func (sd *sortableDim) advance(cmp segment.Key) advanceResult {
	for {
		v := bytes.Compare(sd.current(), cmp)
		switch v {
		case 0:
			return match
		case 1:
			return noMatch
		}
		sd.i++
		if sd.i == sd.l {
			return end
		}
	}
}

type sortableDims []*sortableDim

func (s sortableDims) Len() int {
	return len(s)
}
func (s sortableDims) Less(i, j int) bool {
	return bytes.Compare(s[i].current(), s[j].current()) >= 0
}
func (s sortableDims) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// finds keys that are present in all dimensions
func Intersection(input ...*Dimension) []segment.Key {
	if len(input) == 0 {
		return []segment.Key{}
	} else if len(input) == 1 {
		return input[0].keys
	}

	result := []segment.Key{}

	dims := []*sortableDim{}

	for _, v := range input {
		if len(v.keys) == 0 {
			return []segment.Key{}
		}
		dims = append(dims, &sortableDim{
			keys: v.keys,
			i:    0,
			l:    len(v.keys),
		})
	}

	for {
		// step 1: find the dimension with the smallest element
		sd := sortableDims(dims)
		sort.Sort(sd)

		// step 2: for each other dimension move the pointer until found the matching element or an element higher
		val := dims[0].current()
		allMatch := true
		for _, dim := range sd[1:] {
			res := dim.advance(val)
			switch res {
			case noMatch:
				allMatch = false
			case end:
				return result
			}
		}

		// step 3: if all series are on the same matching element, add it to result
		if allMatch {
			result = append(result, val)
		}
		for _, dim := range sd {
			dim.i++
			if dim.i == dim.l {
				return result
			}
		}
	}
}