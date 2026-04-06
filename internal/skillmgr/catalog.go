package skillmgr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// CatalogEntry describes a skill available from the default source.
type CatalogEntry struct {
	Name        string
	Description string
}

// FetchCatalog discovers available skills by listing skills/ from the
// GitHub repo at the given ref, fetching each skill's SKILL.md frontmatter to
// extract name and description.
func FetchCatalog(ctx context.Context, owner, repo, ref string) ([]CatalogEntry, error) {
	baseURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	return fetchCatalogFromBase(ctx, baseURL, ref)
}

func fetchCatalogFromBase(ctx context.Context, baseURL, ref string) ([]CatalogEntry, error) {
	dirs, err := listSkillDirs(ctx, baseURL, ref)
	if err != nil {
		return nil, err
	}

	type result struct {
		entry CatalogEntry
		err   error
	}

	results := make([]result, len(dirs))
	var wg sync.WaitGroup
	for i, name := range dirs {
		wg.Add(1)
		go func(idx int, skillName string) {
			defer wg.Done()
			entry, ferr := fetchSkillMeta(ctx, baseURL, skillName, ref)
			results[idx] = result{entry: entry, err: ferr}
		}(i, name)
	}
	wg.Wait()

	var catalog []CatalogEntry
	for _, r := range results {
		if r.err != nil {
			log.Printf("Warning: skipping skill: %v", r.err)
			continue
		}
		catalog = append(catalog, r.entry)
	}
	return catalog, nil
}

// listSkillDirs lists subdirectory names under skills/.
func listSkillDirs(ctx context.Context, baseURL, ref string) ([]string, error) {
	apiURL := fmt.Sprintf("%s/contents/%s?ref=%s", baseURL, CatalogDir, url.QueryEscape(ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for skills dir: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing skills dir: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("skills directory not found at ref %s", ref)
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("GitHub API rate limit exceeded; try again later")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d for skills listing", resp.StatusCode)
	}

	var entries []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("decoding skills directory: %w", err)
	}

	var dirs []string
	for _, e := range entries {
		if e.Type != "dir" {
			continue
		}
		dirs = append(dirs, e.Name)
	}
	return dirs, nil
}

// skillFrontmatter represents the YAML frontmatter of a SKILL.md file.
type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// fetchSkillMeta fetches SKILL.md for a single skill and parses its frontmatter.
// It reads the base64-encoded content directly from the Contents API response
// to avoid a second HTTP request.
func fetchSkillMeta(ctx context.Context, baseURL, skillName, ref string) (CatalogEntry, error) {
	apiURL := fmt.Sprintf("%s/contents/%s/%s/SKILL.md?ref=%s",
		baseURL, CatalogDir, url.PathEscape(skillName), url.QueryEscape(ref))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("creating request for %s SKILL.md: %w", skillName, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("fetching %s SKILL.md: %w", skillName, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return CatalogEntry{}, fmt.Errorf("SKILL.md not found for %s (HTTP %d)", skillName, resp.StatusCode)
	}

	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return CatalogEntry{}, fmt.Errorf("decoding SKILL.md metadata for %s: %w", skillName, err)
	}

	if file.Encoding != "base64" {
		return CatalogEntry{}, fmt.Errorf("unexpected encoding %q for %s SKILL.md", file.Encoding, skillName)
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("decoding base64 content for %s: %w", skillName, err)
	}

	fm, err := parseFrontmatter(content)
	if err != nil {
		return CatalogEntry{}, fmt.Errorf("parsing frontmatter for %s: %w", skillName, err)
	}

	return CatalogEntry{Name: skillName, Description: fm.Description}, nil
}

// parseFrontmatter extracts YAML frontmatter from a SKILL.md file.
func parseFrontmatter(content []byte) (skillFrontmatter, error) {
	const sep = "---"
	s := string(content)

	start := strings.Index(s, sep)
	if start == -1 {
		return skillFrontmatter{}, fmt.Errorf("no frontmatter found")
	}
	afterStart := start + len(sep)
	end := strings.Index(s[afterStart:], sep)
	if end == -1 {
		return skillFrontmatter{}, fmt.Errorf("unterminated frontmatter")
	}

	fmBytes := []byte(s[afterStart : afterStart+end])
	var fm skillFrontmatter
	if err := yaml.NewDecoder(bytes.NewReader(fmBytes)).Decode(&fm); err != nil {
		return skillFrontmatter{}, fmt.Errorf("decoding YAML: %w", err)
	}
	fm.Description = strings.TrimSpace(fm.Description)
	return fm, nil
}
