package review

import "testing"

func TestMergeAndFilter(t *testing.T) {
	findings := []Finding{
		{File: "a.go", Line: 1, Severity: "CRITICAL", Title: "Bug"},
		{File: "b.go", Line: 2, Severity: "WARNING", Title: "Perf"},
		{File: "c.go", Line: 3, Severity: "SUGGESTION", Title: "Style"},
	}
	scores := []ScoreEntry{
		{Index: 0, Score: 90},
		{Index: 1, Score: 40},
		{Index: 2, Score: 60},
	}

	// Threshold 50: should keep index 0 and 2
	result := mergeAndFilter(findings, scores, 50)
	if len(result) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(result))
	}
	if result[0].Title != "Bug" || result[0].Score != 90 {
		t.Errorf("first finding: got %+v", result[0])
	}
	if result[1].Title != "Style" || result[1].Score != 60 {
		t.Errorf("second finding: got %+v", result[1])
	}
}

func TestMergeAndFilter_ThresholdZero(t *testing.T) {
	findings := []Finding{
		{File: "a.go", Line: 1, Severity: "WARNING", Title: "A"},
	}
	scores := []ScoreEntry{{Index: 0, Score: 10}}
	result := mergeAndFilter(findings, scores, 0)
	if len(result) != 1 {
		t.Fatalf("threshold 0 should keep all, got %d", len(result))
	}
}

func TestMergeAndFilter_ThresholdHundred(t *testing.T) {
	findings := []Finding{
		{File: "a.go", Line: 1, Severity: "WARNING", Title: "A"},
	}
	scores := []ScoreEntry{{Index: 0, Score: 99}}
	result := mergeAndFilter(findings, scores, 100)
	if len(result) != 0 {
		t.Fatalf("threshold 100 should filter 99, got %d", len(result))
	}
}

func TestMergeAndFilter_AllFiltered(t *testing.T) {
	findings := []Finding{
		{File: "a.go", Line: 1, Severity: "WARNING", Title: "A"},
		{File: "b.go", Line: 2, Severity: "SUGGESTION", Title: "B"},
	}
	scores := []ScoreEntry{{Index: 0, Score: 20}, {Index: 1, Score: 30}}
	result := mergeAndFilter(findings, scores, 50)
	if len(result) != 0 {
		t.Fatalf("all should be filtered, got %d", len(result))
	}
}

func TestMergeAndFilter_MissingScore(t *testing.T) {
	findings := []Finding{
		{File: "a.go", Line: 1, Severity: "WARNING", Title: "A"},
	}
	// No scores provided — should default to 50
	result := mergeAndFilter(findings, nil, 50)
	if len(result) != 1 {
		t.Fatalf("missing score defaults to 50, should be kept at threshold 50, got %d", len(result))
	}
}
