package gen

import (
	"math"
)

type Pair struct {
	K string
	V interface{}
}

func (p *Pair) Key() string {
	return p.K
}

func (p *Pair) Int() int {
	if i, ok := p.V.(int); ok {
		return i
	}
	// log.Printf("error reading int: %s %v", p.K, p.V)
	return math.MinInt64
}

func (p *Pair) String() string {
	if str, ok := p.V.(string); ok {
		return str
	}
	// log.Printf("error reading string: %s %v", p.K, p.V)
	return ""
}

func (p *Pair) Strings() []string {
	if slices, ok := p.V.([]string); ok {
		return slices
	}
	// log.Printf("error reading strings: %s %v", p.K, p.V)
	return nil
}
