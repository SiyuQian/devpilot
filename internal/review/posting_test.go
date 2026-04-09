package review

import "testing"

func TestParseDiffRanges(t *testing.T) {
	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -10,5 +10,7 @@ func main() {
+new line 1
+new line 2
@@ -30,3 +32,4 @@ func helper() {
+another line
diff --git a/util.go b/util.go
--- a/util.go
+++ b/util.go
@@ -1,3 +1,5 @@
+added
+also added
`
	ranges := parseDiffRanges(diff)
	if len(ranges) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(ranges))
	}

	// First hunk: main.go +10,7
	if ranges[0].File != "main.go" || ranges[0].StartLine != 10 || ranges[0].LineCount != 7 {
		t.Errorf("range[0] = %+v", ranges[0])
	}
	// Second hunk: main.go +32,4
	if ranges[1].File != "main.go" || ranges[1].StartLine != 32 || ranges[1].LineCount != 4 {
		t.Errorf("range[1] = %+v", ranges[1])
	}
	// Third hunk: util.go +1,5
	if ranges[2].File != "util.go" || ranges[2].StartLine != 1 || ranges[2].LineCount != 5 {
		t.Errorf("range[2] = %+v", ranges[2])
	}
}

func TestIsLineInDiffRange(t *testing.T) {
	ranges := []DiffRange{
		{File: "main.go", StartLine: 10, LineCount: 7},
		{File: "main.go", StartLine: 32, LineCount: 4},
	}

	tests := []struct {
		file string
		line int
		want bool
	}{
		{"main.go", 10, true},   // start of range
		{"main.go", 16, true},   // end of range
		{"main.go", 17, false},  // just past range
		{"main.go", 9, false},   // just before range
		{"main.go", 32, true},   // second range
		{"main.go", 35, true},   // end of second range
		{"main.go", 36, false},  // past second range
		{"other.go", 10, false}, // wrong file
	}

	for _, tt := range tests {
		got := isLineInDiffRange(ranges, tt.file, tt.line)
		if got != tt.want {
			t.Errorf("isLineInDiffRange(%q, %d) = %v, want %v", tt.file, tt.line, got, tt.want)
		}
	}
}

func TestIsLineInDiffRange_NilRanges(t *testing.T) {
	if isLineInDiffRange(nil, "foo.go", 10) {
		t.Error("nil ranges should return false")
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		line      string
		wantStart int
		wantCount int
	}{
		{"@@ -1,3 +10,7 @@ func main()", 10, 7},
		{"@@ -0,0 +1,5 @@", 1, 5},
		{"@@ -5 +5 @@ single line", 5, 1},
	}
	for _, tt := range tests {
		start, count := parseHunkHeader(tt.line)
		if start != tt.wantStart || count != tt.wantCount {
			t.Errorf("parseHunkHeader(%q) = (%d, %d), want (%d, %d)", tt.line, start, count, tt.wantStart, tt.wantCount)
		}
	}
}

func TestFormatCommentBody(t *testing.T) {
	f := ScoredFinding{
		Finding: Finding{
			Severity:    "WARNING",
			Title:       "Missing check",
			Explanation: "Should check error",
			Suggestion:  "if err != nil { return err }",
		},
		Score: 75,
	}
	body := formatCommentBody(f)
	if body == "" {
		t.Error("comment body should not be empty")
	}
}
