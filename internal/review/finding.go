package review

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Finding represents a single code review finding from Round 1.
type Finding struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	EndLine     int    `json:"end_line,omitempty"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// ScoredFinding pairs a Finding with its confidence score from Round 2.
type ScoredFinding struct {
	Finding
	Score int `json:"score"`
}

// ReviewOutput is the JSON structure returned by Round 1.
type ReviewOutput struct {
	Summary    string    `json:"summary"`
	Findings   []Finding `json:"findings"`
	Assessment string    `json:"assessment"`
}

// ScoreEntry is a single scoring result from Round 2.
type ScoreEntry struct {
	Index int `json:"index"`
	Score int `json:"score"`
}

// ParseReviewOutput parses the JSON output from Round 1.
func ParseReviewOutput(data string) (*ReviewOutput, error) {
	cleaned := cleanJSON(data)
	var out ReviewOutput
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("parse review output: %w", err)
	}
	return &out, nil
}

// ParseScores parses the JSON array output from Round 2.
func ParseScores(data string) ([]ScoreEntry, error) {
	cleaned := cleanJSON(data)
	var scores []ScoreEntry
	if err := json.Unmarshal([]byte(cleaned), &scores); err != nil {
		return nil, fmt.Errorf("parse scores: %w", err)
	}
	return scores, nil
}

// cleanJSON strips markdown fences and surrounding whitespace.
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip ```json ... ``` wrapping
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s[3:], "\n"); idx >= 0 {
			s = s[3+idx+1:]
		}
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}
