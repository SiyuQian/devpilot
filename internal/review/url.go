package review

import (
	"fmt"
	"regexp"
)

// PRInfo holds parsed components of a GitHub pull request URL.
type PRInfo struct {
	Owner  string
	Repo   string
	Number string
	URL    string
}

var prURLPattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/pull/(\d+)`)

// ParsePRURL extracts owner, repo, and PR number from a GitHub PR URL.
func ParsePRURL(url string) (*PRInfo, error) {
	matches := prURLPattern.FindStringSubmatch(url)
	if matches == nil {
		return nil, fmt.Errorf("invalid GitHub PR URL: %s (expected https://github.com/{owner}/{repo}/pull/{number})", url)
	}
	return &PRInfo{
		Owner:  matches[1],
		Repo:   matches[2],
		Number: matches[3],
		URL:    url,
	}, nil
}
