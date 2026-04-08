package skillmgr

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultOwner = "siyuqian"
	defaultRepo  = "devpilot"

	// DefaultSource is the default GitHub source for devpilot skills.
	DefaultSource = "github.com/" + defaultOwner + "/" + defaultRepo

	// CatalogDir is the directory name for skill sources in the catalog repository.
	CatalogDir = "skills"

	// InstallDir is the directory where skills are installed in a project.
	InstallDir = ".claude/skills"

	// defaultRef is the git ref used when fetching skills without a pinned version.
	defaultRef = "main"
)

// SkillFile represents a single file to be written when installing a skill.
type SkillFile struct {
	// Path is relative to skills/<skillName>/
	Path    string
	Content []byte
}

// FetchSkill fetches all files for the named skill by reading index.json
// from raw.githubusercontent.com, then downloading each file listed in the index.
func FetchSkill(owner, repo, skillName, ref string) ([]SkillFile, error) {
	ctx := context.Background()
	entries, err := FetchIndex(ctx, owner, repo, ref)
	if err != nil {
		return nil, fmt.Errorf("fetching index for skill %s: %w", skillName, err)
	}

	var skillEntry *IndexEntry
	for i := range entries {
		if entries[i].Name == skillName {
			skillEntry = &entries[i]
			break
		}
	}
	if skillEntry == nil {
		return nil, fmt.Errorf("skill %q not found in index", skillName)
	}

	var files []SkillFile
	for _, filePath := range skillEntry.Files {
		url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s",
			rawBaseURL, owner, repo, ref, CatalogDir, skillName, filePath)
		content, err := downloadRawFile(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("downloading %s: %w", filePath, err)
		}
		files = append(files, SkillFile{Path: filePath, Content: content})
	}
	return files, nil
}

func downloadRawFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}
