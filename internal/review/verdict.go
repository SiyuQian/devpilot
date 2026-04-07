package review

import "strings"

// IsApproved parses the structured review output and returns true if the verdict is APPROVE.
// It looks for the "## Verdict" section and checks for "APPROVE" on the line(s) following it.
func IsApproved(stdout string) bool {
	lines := strings.Split(stdout, "\n")
	inVerdict := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "## Verdict" {
			inVerdict = true
			continue
		}
		if inVerdict {
			if trimmed == "" {
				continue
			}
			// Check if this line starts with APPROVE
			if strings.HasPrefix(trimmed, "APPROVE") {
				return true
			}
			if strings.HasPrefix(trimmed, "REQUEST_CHANGES") {
				return false
			}
			// If we hit a new section header, verdict section is over
			if strings.HasPrefix(trimmed, "##") {
				return false
			}
		}
	}
	return false
}
