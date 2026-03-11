package skillmgr

import (
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
)

// SkillFile represents a single file to be written when installing a skill.
type SkillFile struct {
	// Path is relative to .claude/skills/<skillName>/
	Path    string
	Content []byte
}

// FetchLatestTag returns the latest release tag for the given GitHub repo.
func FetchLatestTag(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	return fetchLatestTagFromURL(url)
}

func fetchLatestTagFromURL(url string) (string, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

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

// FetchSkill fetches all files for the named skill from the GitHub repo at the given tag.
// It returns a flat list of SkillFile with paths relative to .claude/skills/<skillName>/.
func FetchSkill(owner, repo, skillName, tag string) ([]SkillFile, error) {
	baseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	return fetchSkillFromBase(baseURL, skillName, tag)
}

func fetchSkillFromBase(baseURL, skillName, tag string) ([]SkillFile, error) {
	basePath := fmt.Sprintf(".claude/skills/%s", skillName)
	return fetchContentsRecursive(baseURL, basePath, tag, "")
}

// fetchContentsRecursive lists a directory via the GitHub Contents API and downloads
// each file, recursing into subdirectories. pathPrefix is the relative path from the
// skill root (empty for top-level).
func fetchContentsRecursive(baseURL, basePath, ref, pathPrefix string) ([]SkillFile, error) {
	apiPath := basePath
	if pathPrefix != "" {
		apiPath = basePath + "/" + pathPrefix
	}

	url := fmt.Sprintf("%s/contents/%s?ref=%s", baseURL, apiPath, ref)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("fetching contents at %s: %w", apiPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("skill %q not found at ref %s", basePath, ref)
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("GitHub API rate limit exceeded; try again later")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, apiPath)
	}

	var entries []struct {
		Type        string `json:"type"`
		Name        string `json:"name"`
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decoding contents response: %w", err)
	}

	var files []SkillFile
	for _, entry := range entries {
		relPath := entry.Name
		if pathPrefix != "" {
			relPath = pathPrefix + "/" + entry.Name
		}

		switch entry.Type {
		case "file":
			content, err := downloadFile(entry.DownloadURL)
			if err != nil {
				return nil, fmt.Errorf("downloading %s: %w", relPath, err)
			}
			files = append(files, SkillFile{Path: relPath, Content: content})
		case "dir":
			sub, err := fetchContentsRecursive(baseURL, basePath, ref, relPath)
			if err != nil {
				return nil, err
			}
			files = append(files, sub...)
		}
	}
	return files, nil
}

func downloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
