package prreview

import (
	"fmt"
	"regexp"
	"strings"
)

// Artifact is one dependency named by the diff, keyed by where it appeared.
// Exactly one of Module/Package/Crate is set depending on the ecosystem.
type Artifact struct {
	ManifestLine string `json:"manifest_line,omitempty"`
	ImportLine   string `json:"import_line,omitempty"`
	Module       string `json:"module,omitempty"`
	Package      string `json:"package,omitempty"`
	Crate        string `json:"crate,omitempty"`
	Version      string `json:"version"`
}

// Manifest is the per-ecosystem dependency list handed to the skill's
// import-verifier agent.
type Manifest struct {
	Go     []Artifact `json:"go"`
	NPM    []Artifact `json:"npm"`
	Python []Artifact `json:"python"`
	Rust   []Artifact `json:"rust"`
}

// baseManifests carries the base-ref manifest contents used to exclude
// dependencies that already existed before the PR. Empty strings mean the
// file could not be fetched; exclusion is then skipped (best-effort).
type baseManifests struct {
	goMod       string
	packageJSON string
}

var (
	goRequireRE   = regexp.MustCompile(`^\s*(?:require\s+)?([\w./-]+\.[\w./-]+)\s+(v[\w.+-]+)`)
	goImportRE    = regexp.MustCompile(`^\s*(?:import\s+)?(?:[\w.]+\s+)?"([^"]+)"`)
	npmDepRE      = regexp.MustCompile(`^\s*"(@?[\w./-]+)"\s*:\s*"([~^><=*]?[^"]*)",?\s*$`)
	jsImportRE    = regexp.MustCompile(`(?:from\s+|require\s*\(\s*|import\s+)["']([^"']+)["']`)
	pyRequireRE   = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9._-]*)\s*(?:\[[^\]]*\])?\s*([=<>!~^][^;#\s]*)?`)
	tomlSectionRE = regexp.MustCompile(`^\s*\[([^\]]+)\]`)
	tomlDepRE     = regexp.MustCompile(`^\s*([A-Za-z0-9_-]+)\s*=\s*(?:"([^"]+)"|\{.*?version\s*=\s*"([^"]+)")`)
	setupPyDepRE  = regexp.MustCompile(`^\s*["']([A-Za-z0-9][A-Za-z0-9._-]*)\s*([=<>!~^][^"']*)?["'],?\s*$`)
)

var nodeBuiltins = map[string]bool{
	"assert": true, "buffer": true, "child_process": true, "cluster": true,
	"console": true, "constants": true, "crypto": true, "dgram": true,
	"dns": true, "domain": true, "events": true, "fs": true, "http": true,
	"http2": true, "https": true, "module": true, "net": true, "os": true,
	"path": true, "perf_hooks": true, "process": true, "punycode": true,
	"querystring": true, "readline": true, "repl": true, "stream": true,
	"string_decoder": true, "timers": true, "tls": true, "tty": true,
	"url": true, "util": true, "v8": true, "vm": true, "worker_threads": true,
	"zlib": true,
}

// extractManifest walks the diff's added lines and assembles the dependency
// manifest per the import-verifier extraction rules.
func extractManifest(files []FileDiff, base baseManifests) Manifest {
	m := Manifest{Go: []Artifact{}, NPM: []Artifact{}, Python: []Artifact{}, Rust: []Artifact{}}
	for _, f := range files {
		if f.Deleted {
			continue
		}
		name := f.Path[strings.LastIndex(f.Path, "/")+1:]
		switch {
		case name == "go.mod":
			m.Go = append(m.Go, extractGoMod(f)...)
		case strings.HasSuffix(name, ".go"):
			m.Go = append(m.Go, extractGoImports(f, base.goMod)...)
		case name == "package.json":
			m.NPM = append(m.NPM, extractPackageJSON(f, base.packageJSON)...)
		case isJSSource(name):
			m.NPM = append(m.NPM, extractJSImports(f, base.packageJSON)...)
		case strings.HasPrefix(name, "requirements") && strings.HasSuffix(name, ".txt"):
			m.Python = append(m.Python, extractRequirements(f)...)
		case name == "pyproject.toml":
			m.Python = append(m.Python, extractTOMLDeps(f, "tool.poetry.dependencies", "tool.poetry.dev-dependencies")...)
		case name == "setup.py":
			m.Python = append(m.Python, extractSetupPy(f)...)
		case name == "Cargo.toml":
			for _, a := range extractTOMLDeps(f, "dependencies", "dev-dependencies") {
				a.Crate, a.Package = a.Package, ""
				m.Rust = append(m.Rust, a)
			}
		}
	}
	return m
}

func isJSSource(name string) bool {
	for _, ext := range []string{".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs"} {
		if strings.HasSuffix(name, ext) && !strings.HasSuffix(name, ".d.ts") {
			return true
		}
	}
	return false
}

func lineRef(path string, line int) string { return fmt.Sprintf("%s:%d", path, line) }

func extractGoMod(f FileDiff) []Artifact {
	var out []Artifact
	for _, l := range f.Lines {
		if l.Origin != '+' {
			continue
		}
		if m := goRequireRE.FindStringSubmatch(l.Text); m != nil {
			out = append(out, Artifact{ManifestLine: lineRef(f.Path, l.NewLine), Module: m[1], Version: m[2]})
		}
	}
	return out
}

// goStdlib reports whether an import path is standard library: its first
// segment contains no dot (no host), the canonical Go heuristic.
func goStdlib(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

func extractGoImports(f FileDiff, baseGoMod string) []Artifact {
	var out []Artifact
	inImport := false
	for _, l := range f.Lines {
		trimmed := strings.TrimSpace(l.Text)
		if strings.HasPrefix(trimmed, "import (") {
			inImport = true
			continue
		}
		if inImport && trimmed == ")" {
			inImport = false
			continue
		}
		if l.Origin != '+' {
			continue
		}
		looksImport := inImport || strings.HasPrefix(trimmed, "import ")
		if !looksImport {
			continue
		}
		m := goImportRE.FindStringSubmatch(l.Text)
		if m == nil || goStdlib(m[1]) {
			continue
		}
		if baseGoMod != "" && importCoveredByGoMod(m[1], baseGoMod) {
			continue
		}
		out = append(out, Artifact{ImportLine: lineRef(f.Path, l.NewLine), Module: m[1], Version: ""})
	}
	return out
}

// importCoveredByGoMod reports whether the import path is rooted in a module
// already required by the given go.mod contents (or is the module itself).
func importCoveredByGoMod(importPath, goMod string) bool {
	for line := range strings.SplitSeq(goMod, "\n") {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "require"))
		if len(fields) == 0 {
			continue
		}
		mod := fields[0]
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			mod = strings.TrimSpace(rest)
		}
		if !strings.Contains(mod, ".") {
			continue
		}
		if importPath == mod || strings.HasPrefix(importPath, mod+"/") {
			return true
		}
	}
	return false
}

func extractPackageJSON(f FileDiff, basePackageJSON string) []Artifact {
	var out []Artifact
	for _, l := range f.Lines {
		if l.Origin != '+' {
			continue
		}
		m := npmDepRE.FindStringSubmatch(l.Text)
		if m == nil || !looksLikeSemverRange(m[2]) {
			continue
		}
		if pkgInPackageJSON(m[1], basePackageJSON) {
			continue
		}
		out = append(out, Artifact{ManifestLine: lineRef(f.Path, l.NewLine), Package: m[1], Version: m[2]})
	}
	return out
}

// looksLikeSemverRange filters package.json values to dependency-shaped ones,
// excluding script bodies and metadata strings.
func looksLikeSemverRange(v string) bool {
	if v == "" || v == "*" || strings.HasPrefix(v, "workspace:") || strings.HasPrefix(v, "npm:") {
		return v != ""
	}
	c := v[0]
	return c == '^' || c == '~' || c == '>' || c == '<' || c == '=' || (c >= '0' && c <= '9')
}

func pkgInPackageJSON(pkg, packageJSON string) bool {
	return packageJSON != "" && strings.Contains(packageJSON, `"`+pkg+`"`)
}

func extractJSImports(f FileDiff, basePackageJSON string) []Artifact {
	var out []Artifact
	for _, l := range f.Lines {
		if l.Origin != '+' {
			continue
		}
		m := jsImportRE.FindStringSubmatch(l.Text)
		if m == nil {
			continue
		}
		pkg := npmPackageName(m[1])
		if pkg == "" || nodeBuiltins[pkg] || pkgInPackageJSON(pkg, basePackageJSON) {
			continue
		}
		out = append(out, Artifact{ImportLine: lineRef(f.Path, l.NewLine), Package: pkg, Version: ""})
	}
	return out
}

// npmPackageName reduces an import specifier to its registry package name,
// or "" when it isn't a registry package (relative path, node builtin).
func npmPackageName(spec string) string {
	if strings.HasPrefix(spec, ".") || strings.HasPrefix(spec, "/") || strings.HasPrefix(spec, "node:") {
		return ""
	}
	parts := strings.Split(spec, "/")
	if strings.HasPrefix(spec, "@") {
		if len(parts) < 2 {
			return ""
		}
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

func extractRequirements(f FileDiff) []Artifact {
	var out []Artifact
	for _, l := range f.Lines {
		trimmed := strings.TrimSpace(l.Text)
		if l.Origin != '+' || trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		if m := pyRequireRE.FindStringSubmatch(trimmed); m != nil {
			out = append(out, Artifact{ManifestLine: lineRef(f.Path, l.NewLine), Package: m[1], Version: m[2]})
		}
	}
	return out
}

func extractSetupPy(f FileDiff) []Artifact {
	var out []Artifact
	for _, l := range f.Lines {
		if l.Origin != '+' {
			continue
		}
		if m := setupPyDepRE.FindStringSubmatch(l.Text); m != nil {
			out = append(out, Artifact{ManifestLine: lineRef(f.Path, l.NewLine), Package: m[1], Version: m[2]})
		}
	}
	return out
}

// extractTOMLDeps pulls added `name = "version"` entries that sit under one
// of the wanted TOML sections. Section state is tracked across context and
// added lines within the visible hunks.
func extractTOMLDeps(f FileDiff, sections ...string) []Artifact {
	wanted := map[string]bool{}
	for _, s := range sections {
		wanted[s] = true
	}
	var out []Artifact
	inWanted := false
	for _, l := range f.Lines {
		if m := tomlSectionRE.FindStringSubmatch(l.Text); m != nil {
			inWanted = wanted[m[1]]
			continue
		}
		if !inWanted || l.Origin != '+' {
			continue
		}
		if m := tomlDepRE.FindStringSubmatch(l.Text); m != nil {
			version := m[2]
			if version == "" {
				version = m[3]
			}
			out = append(out, Artifact{ManifestLine: lineRef(f.Path, l.NewLine), Package: m[1], Version: version})
		}
	}
	return out
}
