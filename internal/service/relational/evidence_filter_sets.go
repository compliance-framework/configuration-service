package relational

import (
	"encoding/json"
	"sort"

	"github.com/compliance-framework/api/internal/converters/labelfilter"
	"gorm.io/gorm"
)

// EvidenceStatusCountsForFilterSets is the batched form of EvidenceStatusCountsForFilters:
// it computes the latest-evidence status counts for many filter sets (one per control or
// responsibility, say) from a single load of the latest evidence streams, instead of one
// full evidence aggregate per set. Each set's result is what EvidenceStatusCountsForFilters
// returns for it. The database is only read when at least one set has filters.
// Postgres-first, like LoadLatestEvidenceStreams.
func EvidenceStatusCountsForFilterSets[K comparable](db *gorm.DB, sets map[K][]labelfilter.Filter) (map[K][]StatusCount, error) {
	needsStreams := false
	for _, filters := range sets {
		if len(filters) > 0 {
			needsStreams = true
			break
		}
	}
	var streams []LatestEvidenceStream
	if needsStreams {
		var err error
		streams, err = LoadLatestEvidenceStreams(db, nil)
		if err != nil {
			return nil, err
		}
	}
	return CountStatusesForFilterSets(streams, sets)
}

// CountStatusesForFilterSets evaluates every filter set against the latest evidence
// streams in memory, with the semantics of EvidenceStatusCountsForFilters except one:
// label names and values are trimmed as well as case-folded (LoadLatestEvidenceStreams
// and labelfilter.MatchLabels both trim, as lineage does), so a label stored as " ec2"
// matches ec2 here where the SQL path, which only case-folds, did not. Otherwise:
//   - a set with no filters counts nothing;
//   - filters without a scope are ignored, unless no filter in the set has one, in which
//     case every stream matches;
//   - a stream matching any scoped filter of the set is counted once, under its state;
//   - rows are sorted by status.
//
// The streams are indexed by label once, and each distinct filter is matched through
// the index once however many sets share it, so the cost tracks the matches rather
// than filters × streams. An unknown query operator is always an error, even where
// labelfilter.MatchLabels would short-circuit before reaching it.
func CountStatusesForFilterSets[K comparable](streams []LatestEvidenceStream, sets map[K][]labelfilter.Filter) (map[K][]StatusCount, error) {
	result := make(map[K][]StatusCount, len(sets))
	matchesByFilter := map[string][]int{}
	var allStreams []int
	var index *labelfilter.Index // built on the first scoped filter

	// seen[i] == generation marks stream i as already counted for the current set.
	seen := make([]int, len(streams))
	generation := 0

	for key, filters := range sets {
		if len(filters) == 0 {
			result[key] = []StatusCount{}
			continue
		}
		generation++
		counts := map[string]int64{}
		count := func(indexes []int) {
			for _, i := range indexes {
				if seen[i] == generation {
					continue
				}
				seen[i] = generation
				counts[streams[i].State]++
			}
		}

		scoped := false
		for _, filter := range filters {
			if filter.Scope == nil {
				continue
			}
			scoped = true
			if index == nil {
				index = NewLatestStreamIndex(streams)
			}
			indexes, err := matchingStreams(index, filter.Scope, matchesByFilter)
			if err != nil {
				return nil, err
			}
			count(indexes)
		}
		if !scoped {
			if allStreams == nil {
				allStreams = make([]int, len(streams))
				for i := range streams {
					allStreams[i] = i
				}
			}
			count(allStreams)
		}

		rows := make([]StatusCount, 0, len(counts))
		for status, c := range counts {
			rows = append(rows, StatusCount{Count: c, Status: status})
		}
		sort.Slice(rows, func(i, j int) bool {
			return rows[i].Status < rows[j].Status
		})
		result[key] = rows
	}
	return result, nil
}

// NewLatestStreamIndex indexes the streams' labels (already normalized by
// LoadLatestEvidenceStreams); Match reports positions in streams.
func NewLatestStreamIndex(streams []LatestEvidenceStream) *labelfilter.Index {
	labels := make([]map[string][]string, len(streams))
	for i := range streams {
		labels[i] = streams[i].Labels
	}
	return labelfilter.NewIndex(labels)
}

// matchingStreams returns the indexes of the streams the scope matches, memoised by the
// scope's JSON encoding so a filter shared by many sets is evaluated once.
func matchingStreams(index *labelfilter.Index, scope *labelfilter.Scope, cache map[string][]int) ([]int, error) {
	encoded, err := json.Marshal(scope)
	if err != nil {
		return nil, err
	}
	if indexes, ok := cache[string(encoded)]; ok {
		return indexes, nil
	}
	indexes, err := index.Match(scope)
	if err != nil {
		return nil, err
	}
	cache[string(encoded)] = indexes
	return indexes, nil
}
