package gen

import (
	"math"
	"strconv"
)

type Pair struct {
	K string
	V interface{}
}

// func (p Pair) Key() string {
// 	return p.K
// }

func (p Pair) Int() (string, int) {
	if i, ok := p.V.(int); ok {
		return p.K, i
	} else if s, ok := p.V.(string); ok {
		i, err := strconv.Atoi(s)
		if err == nil {
			return p.K, i
		}
	}
	return p.K, math.MinInt64
}

func (p Pair) String() (string, string) {
	if str, ok := p.V.(string); ok {
		return p.K, str
	}
	return p.K, ""
}

func (p Pair) Strings() (string, []string) {
	if slices, ok := p.V.([]string); ok {
		return p.K, slices
	}
	return p.K, nil
}
