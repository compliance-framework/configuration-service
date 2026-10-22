// Package sspscope stores an SSP's scope: the labels that identify the SSP's evidence.
//
// A scope is a list of entries; evidence is in scope when it matches any entry, and it
// matches an entry when it carries all of the entry's key=value labels. Users never
// delete an entry: they add an exception, which removes the entry and stops SSP
// templates from learning it again.
package sspscope

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/compliance-framework/api/internal/service/relational"
)

// Limits on a scope, enforced on every write.
const (
	MaxEntries         = 500
	MaxExceptions      = 500
	MaxLabelsPerEntry  = 10
	MaxLabelPartLength = 256
)

var (
	// ErrInvalidLabels is returned when an entry's or exception's labels are malformed
	// (empty, too many, an empty or over-long key or value, a duplicate key).
	ErrInvalidLabels = errors.New("invalid scope labels")
	// ErrLimitExceeded is returned when a write would take the SSP past MaxEntries or
	// MaxExceptions.
	ErrLimitExceeded = errors.New("scope limit exceeded")
)

// Normalize is the single entry point for labels entering a scope. It trims keys and
// values, lower-cases keys (evidence label keys match case-insensitively), keeps value
// casing, sorts by key and rejects empty, duplicate-key, over-limit and hash-ambiguous
// input. It returns
// the normalized labels and their MatchHash.
func Normalize(labels []relational.ScopeLabel) ([]relational.ScopeLabel, string, error) {
	if len(labels) == 0 {
		return nil, "", fmt.Errorf("%w: at least one label is required", ErrInvalidLabels)
	}
	if len(labels) > MaxLabelsPerEntry {
		return nil, "", fmt.Errorf("%w: at most %d labels per entry", ErrInvalidLabels, MaxLabelsPerEntry)
	}
	out := make([]relational.ScopeLabel, 0, len(labels))
	seen := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		key := strings.ToLower(strings.TrimSpace(label.Key))
		value := strings.TrimSpace(label.Value)
		if key == "" || value == "" {
			return nil, "", fmt.Errorf("%w: label keys and values must not be empty", ErrInvalidLabels)
		}
		if utf8.RuneCountInString(key) > MaxLabelPartLength || utf8.RuneCountInString(value) > MaxLabelPartLength {
			return nil, "", fmt.Errorf("%w: label keys and values must be at most %d characters", ErrInvalidLabels, MaxLabelPartLength)
		}
		// MatchHash joins key=value pairs with "\n": a key holding "=" or a key or value
		// holding a control character could forge another entry's hash.
		if strings.Contains(key, "=") || strings.IndexFunc(key+value, unicode.IsControl) >= 0 {
			return nil, "", fmt.Errorf("%w: keys must not contain '=' and labels must not contain control characters", ErrInvalidLabels)
		}
		if _, dup := seen[key]; dup {
			return nil, "", fmt.Errorf("%w: duplicate key %q", ErrInvalidLabels, key)
		}
		seen[key] = struct{}{}
		out = append(out, relational.ScopeLabel{Key: key, Value: value})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, MatchHash(out), nil
}

// MatchHash is the SHA-256 hex of the sorted lower(key)=lower(value) pairs joined by
// "\n". Evidence labels match case-insensitively, so two entries that match the same
// evidence hash the same. labels must already be sorted by key (Normalize does that).
func MatchHash(labels []relational.ScopeLabel) string {
	pairs := make([]string, len(labels))
	for i, label := range labels {
		pairs[i] = strings.ToLower(label.Key) + "=" + strings.ToLower(label.Value)
	}
	sum := sha256.Sum256([]byte(strings.Join(pairs, "\n")))
	return hex.EncodeToString(sum[:])
}
