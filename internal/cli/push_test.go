package cli

import "testing"

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "standard plan heading",
			content: "# Task Runner Implementation Plan\n\n> For Claude...\n",
			want:    "Task Runner Implementation Plan",
		},
		{
			name:    "heading after blank lines",
			content: "\n\n# My Plan\n\nBody text",
			want:    "My Plan",
		},
		{
			name:    "no heading",
			content: "Just some text\nNo heading here",
			want:    "",
		},
		{
			name:    "ignores ## subheadings",
			content: "## Not This\n# This One\n",
			want:    "This One",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.content)
			if got != tt.want {
				t.Errorf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}
