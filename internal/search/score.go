package search

import (
	"sort"
	"strings"

	"agwctl/internal/gw"
)

// Field weights per PLAN.md section 4. Each field counts at most once, so a
// long description cannot out-rank a precise name hit.
const (
	weightName        = 3.0
	weightNamePrefix  = 2.0
	weightTarget      = 1.5
	weightDescription = 1.0
)

var stopwords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "or": true, "of": true,
	"for": true, "to": true, "in": true, "on": true, "with": true, "is": true,
	"are": true, "this": true, "that": true, "by": true, "from": true, "as": true,
	"at": true, "be": true, "it": true, "its": true,
}

type Ranked struct {
	Row   gw.ToolRow
	Score float64
}

// Tokenize lowercases the query, splits on non-alphanumeric runes and drops
// stopwords plus tokens shorter than three runes.
func Tokenize(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	tokens := fields[:0]
	for _, f := range fields {
		if len(f) < 3 || stopwords[f] {
			continue
		}
		tokens = append(tokens, f)
	}
	return tokens
}

func containsAny(field string, tokens []string) bool {
	for _, t := range tokens {
		if strings.Contains(field, t) {
			return true
		}
	}
	return false
}

func prefixAny(field string, tokens []string) bool {
	for _, w := range strings.FieldsFunc(field, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		for _, t := range tokens {
			if strings.HasPrefix(w, t) {
				return true
			}
		}
	}
	return false
}

func Score(row gw.ToolRow, tokens []string) float64 {
	if len(tokens) == 0 {
		return 0
	}
	name := strings.ToLower(row.Name)
	target := strings.ToLower(row.Target)
	desc := strings.ToLower(row.Description)

	var s float64
	if containsAny(name, tokens) {
		s += weightName
	}
	if prefixAny(name, tokens) {
		s += weightNamePrefix
	}
	if target != "" && containsAny(target, tokens) {
		s += weightTarget
	}
	if containsAny(desc, tokens) {
		s += weightDescription
	}
	return s
}

// Search ranks rows by lexical score and returns up to limit hits. An empty
// result is a valid answer; the caller exits 0 either way.
func Search(rows []gw.ToolRow, query string, limit int) []Ranked {
	tokens := Tokenize(query)
	if len(tokens) == 0 || limit <= 0 {
		return nil
	}
	var ranked []Ranked
	for _, row := range rows {
		if s := Score(row, tokens); s > 0 {
			ranked = append(ranked, Ranked{Row: row, Score: s})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score > ranked[j].Score
		}
		return ranked[i].Row.Name < ranked[j].Row.Name
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked
}
