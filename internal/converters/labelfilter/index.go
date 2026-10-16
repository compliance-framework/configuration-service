package labelfilter

import (
	"fmt"
	"math/bits"
	"strings"
)

// Index answers "which of these label sets does a scope match?" for many scopes over
// one fixed list of label sets, without testing every scope against every set. It
// keeps one bitset per (label, value) pair and evaluates a scope with set algebra:
// "=" is the pair's bitset, "!=" its complement, AND/OR intersect/union.
//
// Match agrees with MatchLabels on every label set, with one exception: an unknown
// query operator is always an error, even where MatchLabels' short-circuit evaluation
// would never reach it.
type Index struct {
	n        int
	postings map[string]map[string]bitset
}

// NewIndex indexes the label sets, which must be normalized like NormalizeLabels does.
// Match reports positions in this slice.
func NewIndex(labels []map[string][]string) *Index {
	idx := &Index{n: len(labels), postings: map[string]map[string]bitset{}}
	for i, set := range labels {
		for key, values := range set {
			byValue, ok := idx.postings[key]
			if !ok {
				byValue = map[string]bitset{}
				idx.postings[key] = byValue
			}
			for _, value := range values {
				b, ok := byValue[value]
				if !ok {
					b = newBitset(idx.n)
					byValue[value] = b
				}
				b.set(i)
			}
		}
	}
	return idx
}

// Match returns the ascending positions of the label sets the scope matches. A nil
// scope matches every set.
func (idx *Index) Match(scope *Scope) ([]int, error) {
	if scope == nil {
		return idx.all().positions(), nil
	}
	b, err := idx.matchScope(*scope)
	if err != nil {
		return nil, err
	}
	return b.positions(), nil
}

// The bitsets returned by the match helpers may be shared postings: callers must not
// modify them.

func (idx *Index) matchScope(scope Scope) (bitset, error) {
	if scope.IsCondition() {
		return idx.matchCondition(*scope.Condition), nil
	}
	if scope.IsQuery() {
		return idx.matchQuery(*scope.Query)
	}
	return idx.all(), nil
}

func (idx *Index) matchCondition(cond Condition) bitset {
	key := strings.ToLower(strings.TrimSpace(cond.Label))
	val := strings.ToLower(strings.TrimSpace(cond.Value))

	posting, ok := idx.postings[key][val]
	if !ok {
		posting = newBitset(idx.n)
	}
	if cond.Operator == "!=" {
		return posting.complement(idx.n)
	}
	return posting
}

func (idx *Index) matchQuery(query Query) (bitset, error) {
	switch strings.ToLower(query.Operator) {
	case "and":
		out := idx.all()
		for _, scope := range query.Scopes {
			b, err := idx.matchScope(scope)
			if err != nil {
				return nil, err
			}
			out.and(b)
		}
		return out, nil
	case "or":
		if len(query.Scopes) == 0 {
			return idx.all(), nil
		}
		out := newBitset(idx.n)
		for _, scope := range query.Scopes {
			b, err := idx.matchScope(scope)
			if err != nil {
				return nil, err
			}
			out.or(b)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unrecognised query operator: %s", query.Operator)
	}
}

func (idx *Index) all() bitset {
	return newBitset(idx.n).complement(idx.n)
}

type bitset []uint64

func newBitset(n int) bitset { return make(bitset, (n+63)/64) }

func (b bitset) set(i int) { b[i/64] |= 1 << (uint(i) % 64) }

// complement returns a new bitset holding the positions below n that b does not.
func (b bitset) complement(n int) bitset {
	out := make(bitset, len(b))
	for i, w := range b {
		out[i] = ^w
	}
	if tail := n % 64; tail != 0 {
		out[len(out)-1] &= 1<<uint(tail) - 1
	}
	return out
}

func (b bitset) and(o bitset) {
	for i := range b {
		b[i] &= o[i]
	}
}

func (b bitset) or(o bitset) {
	for i := range b {
		b[i] |= o[i]
	}
}

func (b bitset) positions() []int {
	out := []int{}
	for i, w := range b {
		for w != 0 {
			out = append(out, i*64+bits.TrailingZeros64(w))
			w &= w - 1
		}
	}
	return out
}
