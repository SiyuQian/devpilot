package skillmgr

import (
	"context"
	"encoding/json"
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
)

// SkillFile represents a single file to be written when installing a skill.
type SkillFile struct {
	// Path is relative to skills/<skillName>/
	Path    string
	Content []byte
}

// FetchLatestTag returns the latest release tag for the given GitHub repo.
func FetchLatestTag(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	return fetchLatestTagFromURL(url)
}

func fetchLatestTagFromURL(url string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request for latest release: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no releases found")
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("GitHub API rate limit exceeded; try again later")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d for releases", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decoding release response: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("release has no tag name")
	}
	return release.TagName, nil
}

// FetchSkill fetches all files for the named skill by reading index.json
// from raw.githubusercontent.com, then downloading each file listed in the index.
func FetchSkill(owner, repo, skillName, tag string) ([]SkillFile, error) {
	ctx := context.Background()
	entries, err := FetchIndex(ctx, owner, repo, tag)
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
			rawBaseURL, owner, repo, tag, CatalogDir, skillName, filePath)
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
