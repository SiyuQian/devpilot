package skillmgr

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestInstallSkillPaths covers path-traversal and input-validation behavior
// of InstallSkill. Happy paths verify expected files land inside skillDir;
// rejection paths verify a descriptive error is returned AND the filesystem
// under baseDir contains no files outside skillDir.
func TestInstallSkillPaths(t *testing.T) {
	type wantFile struct {
		relPath string // path under baseDir (forward slashes)
		content string
	}

	cases := []struct {
		name       string
		skillName  string
		files      []SkillFile
		wantErr    bool
		wantErrHas string // substring the error must contain (when wantErr)
		wantFiles  []wantFile
		skipOnWin  bool // skip on Windows when test depends on unix path semantics
		onlyOnWin  bool
	}{
		{
			name:      "happy path single file",
			skillName: "pm",
			files: []SkillFile{
				{Path: "SKILL.md", Content: []byte("top")},
			},
			wantFiles: []wantFile{
				{relPath: "pm/SKILL.md", content: "top"},
			},
		},
		{
			name:      "happy path nested file",
			skillName: "pm",
			files: []SkillFile{
				{Path: "SKILL.md", Content: []byte("top")},
				{Path: "subdir/SKILL.md", Content: []byte("nested")},
			},
			wantFiles: []wantFile{
				{relPath: "pm/SKILL.md", content: "top"},
				{relPath: "pm/subdir/SKILL.md", content: "nested"},
			},
		},
		{
			name:       "empty skill name",
			skillName:  "",
			files:      []SkillFile{{Path: "SKILL.md", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "skillName",
		},
		{
			name:       "dot skill name",
			skillName:  ".",
			files:      []SkillFile{{Path: "SKILL.md", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "skillName",
		},
		{
			name:       "dotdot skill name",
			skillName:  "..",
			files:      []SkillFile{{Path: "SKILL.md", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "skillName",
		},
		{
			name:       "skill name with forward slash",
			skillName:  "foo/bar",
			files:      []SkillFile{{Path: "SKILL.md", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "skillName",
		},
		{
			name:       "skill name with parent traversal",
			skillName:  "../evil",
			files:      []SkillFile{{Path: "SKILL.md", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "skillName",
		},
		{
			name:       "skill name with backslash",
			skillName:  `foo\bar`,
			files:      []SkillFile{{Path: "SKILL.md", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "skillName",
		},
		{
			name:       "absolute skill name",
			skillName:  "/etc/passwd",
			files:      []SkillFile{{Path: "SKILL.md", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "skillName",
			skipOnWin:  true,
		},
		{
			name:       "file path with parent traversal prefix",
			skillName:  "pm",
			files:      []SkillFile{{Path: "../escape.md", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "file path",
		},
		{
			name:       "file path with nested traversal that escapes after clean",
			skillName:  "pm",
			files:      []SkillFile{{Path: "foo/../../bar", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "file path",
		},
		{
			name:       "absolute file path",
			skillName:  "pm",
			files:      []SkillFile{{Path: "/etc/passwd", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "file path",
			skipOnWin:  true,
		},
		{
			name:       "empty file path",
			skillName:  "pm",
			files:      []SkillFile{{Path: "", Content: []byte("x")}},
			wantErr:    true,
			wantErrHas: "file path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("unix-specific path semantics")
			}
			if tc.onlyOnWin && runtime.GOOS != "windows" {
				t.Skip("windows-specific path semantics")
			}

			baseDir := t.TempDir()
			err := InstallSkill(baseDir, tc.skillName, tc.files)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("InstallSkill: expected error, got nil")
				}
				if tc.wantErrHas != "" && !strings.Contains(err.Error(), tc.wantErrHas) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrHas)
				}
				assertNoFilesOutsideSkillDir(t, baseDir, tc.skillName)
				return
			}

			if err != nil {
				t.Fatalf("InstallSkill: %v", err)
			}
			for _, wf := range tc.wantFiles {
				parts := strings.Split(wf.relPath, "/")
				full := filepath.Join(append([]string{baseDir}, parts...)...)
				data, rerr := os.ReadFile(full)
				if rerr != nil {
					t.Fatalf("reading %s: %v", full, rerr)
				}
				if string(data) != wf.content {
					t.Errorf("%s = %q, want %q", wf.relPath, string(data), wf.content)
				}
			}
			assertOnlyFilesUnder(t, baseDir, filepath.Join(baseDir, tc.skillName))
		})
	}
}

// assertNoFilesOutsideSkillDir walks baseDir and fails if any regular file
// exists outside filepath.Join(baseDir, skillName). The skillDir itself may
// or may not exist (rejection paths typically don't create it).
func assertNoFilesOutsideSkillDir(t *testing.T, baseDir, skillName string) {
	t.Helper()
	skillDir := filepath.Clean(filepath.Join(baseDir, skillName))
	_ = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		cleaned := filepath.Clean(path)
		if cleaned == skillDir || strings.HasPrefix(cleaned, skillDir+string(os.PathSeparator)) {
			return nil
		}
		t.Errorf("unexpected file written outside skillDir: %s", path)
		return nil
	})
}

// assertOnlyFilesUnder walks baseDir and fails if any regular file exists
// outside skillDir. Used on happy paths to confirm no accidental escape.
func assertOnlyFilesUnder(t *testing.T, baseDir, skillDir string) {
	t.Helper()
	skillDir = filepath.Clean(skillDir)
	_ = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		cleaned := filepath.Clean(path)
		if cleaned == skillDir || strings.HasPrefix(cleaned, skillDir+string(os.PathSeparator)) {
			return nil
		}
		t.Errorf("file written outside skillDir: %s", path)
		return nil
	})
}
