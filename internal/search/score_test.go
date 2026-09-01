package search

import (
	"testing"

	"agwctl/internal/gw"
)

func row(name, desc string) gw.ToolRow {
	target := ""
	if name != "" {
		if before, _, found := split(name); found {
			target = before
		}
	}
	return gw.ToolRow{Name: name, Target: target, Description: desc}
}

func split(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func TestTokenizeDropsStopwordsAndShortTokens(t *testing.T) {
	got := Tokenize("Search THE web for docs")
	want := []string{"search", "web", "docs"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestScoreCapsEachFieldOnce(t *testing.T) {
	r := row("exa_web_search_exa", "Search the web. Search again. Search more.")
	got := Score(r, Tokenize("search"))
	if got != weightName+weightNamePrefix+weightDescription {
		t.Fatalf("score = %v", got)
	}
	if Score(row("hindsight-personal_recall", "recall"), Tokenize("search")) != 0 {
		t.Fatal("unrelated row must score zero")
	}
}

func TestSearchRanksAndTiesByName(t *testing.T) {
	rows := []gw.ToolRow{
		row("ref-context_ref_search_documentation", "Search documentation"),
		row("brave-search_web_search", "Search the web"),
		row("exa_web_search_exa", "Search the web and fetch pages"),
		row("openviking_find", "Find resources"),
	}
	got := Search(rows, "search web", 5)
	if len(got) == 0 {
		t.Fatal("no results")
	}
	for i := 1; i < len(got); i++ {
		a, b := got[i-1], got[i]
		if a.Score < b.Score || (a.Score == b.Score && a.Row.Name > b.Row.Name) {
			t.Fatalf("bad order: %v before %v", a, b)
		}
	}
	// brave-search_web_search wins: name and target both hit the query.
	if got[0].Row.Name != "brave-search_web_search" {
		t.Fatalf("top hit = %v", got[0].Row.Name)
	}
}

func TestSearchStopsStopwordOnlyQueries(t *testing.T) {
	rows := []gw.ToolRow{row("exa_web_search_exa", "search")}
	if got := Search(rows, "the for and", 5); len(got) != 0 {
		t.Fatalf("stopword-only query returned %v", got)
	}
}

func TestSearchHonorsLimit(t *testing.T) {
	rows := []gw.ToolRow{
		row("a_search", "search one"), row("b_search", "search two"), row("c_search", "search three"),
	}
	if got := Search(rows, "search", 2); len(got) != 2 {
		t.Fatalf("limit broken: %v", got)
	}
}
