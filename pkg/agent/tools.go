package agent

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	copilot "github.com/github/copilot-sdk/go"
)

// repoTools returns the set of custom tools exposed to the Copilot agent so it
// can explore any repository without a pre-built index.
func repoTools(workDir string) []copilot.Tool {
	return []copilot.Tool{
		listDirTool(workDir),
		readFileTool(workDir),
		searchPatternTool(workDir),
		findSymbolTool(workDir),
	}
}

// ── list_directory ──────────────────────────────────────────────────────────

type listDirParams struct {
	Path       string `json:"path"       jsonschema:"Relative path to list (use '.' for repo root)"`
	MaxDepth   int    `json:"max_depth"  jsonschema:"How many directory levels to recurse (1–5, default 2)"`
	ShowHidden bool   `json:"show_hidden" jsonschema:"Include dot-files and dot-directories"`
}

func listDirTool(workDir string) copilot.Tool {
	return copilot.DefineTool(
		"list_directory",
		"List files and directories in a path relative to the repo root. Use this to map the project structure before reading files.",
		func(p listDirParams, _ copilot.ToolInvocation) (string, error) {
			if p.MaxDepth <= 0 {
				p.MaxDepth = 2
			}
			if p.MaxDepth > 5 {
				p.MaxDepth = 5
			}
			target := filepath.Join(workDir, filepath.Clean(p.Path))
			if !strings.HasPrefix(target, workDir) {
				return "", fmt.Errorf("path escapes the repository root")
			}

			var sb strings.Builder
			err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil // skip unreadable entries
				}
				rel, _ := filepath.Rel(target, path)
				if rel == "." {
					return nil
				}

				// Respect hidden-file toggle.
				parts := strings.Split(rel, string(os.PathSeparator))
				for _, part := range parts {
					if !p.ShowHidden && strings.HasPrefix(part, ".") {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}

				depth := strings.Count(rel, string(os.PathSeparator))
				if depth >= p.MaxDepth {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				indent := strings.Repeat("  ", depth)
				suffix := ""
				if d.IsDir() {
					suffix = "/"
				}
				sb.WriteString(indent + d.Name() + suffix + "\n")
				return nil
			})
			if err != nil {
				return "", err
			}
			return sb.String(), nil
		})
}

// ── read_file ────────────────────────────────────────────────────────────────

type readFileParams struct {
	Path      string `json:"path"       jsonschema:"File path relative to the repo root"`
	StartLine int    `json:"start_line" jsonschema:"First line to return (1-based, 0 means beginning)"`
	EndLine   int    `json:"end_line"   jsonschema:"Last line to return (0 means end of file)"`
}

func readFileTool(workDir string) copilot.Tool {
	return copilot.DefineTool(
		"read_file",
		"Read the contents of a file in the repository. Optionally specify a line range to limit output.",
		func(p readFileParams, _ copilot.ToolInvocation) (string, error) {
			target := filepath.Join(workDir, filepath.Clean(p.Path))
			if !strings.HasPrefix(target, workDir) {
				return "", fmt.Errorf("path escapes the repository root")
			}

			f, err := os.Open(target) // #nosec G304 — path validated above
			if err != nil {
				return "", err
			}
			defer f.Close()

			var lines []string
			scanner := bufio.NewScanner(f)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				if p.StartLine > 0 && lineNum < p.StartLine {
					continue
				}
				if p.EndLine > 0 && lineNum > p.EndLine {
					break
				}
				lines = append(lines, fmt.Sprintf("%4d  %s", lineNum, scanner.Text()))
			}
			if err := scanner.Err(); err != nil {
				return "", err
			}
			return strings.Join(lines, "\n"), nil
		})
}

// ── search_pattern ───────────────────────────────────────────────────────────

type searchPatternParams struct {
	Pattern      string `json:"pattern"      jsonschema:"Regular expression to search for"`
	Path         string `json:"path"         jsonschema:"Directory or file to search (relative to repo root, default '.')"`
	FileGlob     string `json:"file_glob"    jsonschema:"Restrict search to files matching this glob (e.g. '*.go')"`
	ContextLines int    `json:"context_lines" jsonschema:"Lines of context to print around each match (0–5)"`
}

func searchPatternTool(workDir string) copilot.Tool {
	return copilot.DefineTool(
		"search_pattern",
		"Search files in the repository for a regular expression. Returns matching lines with file path and line number, like grep -rn.",
		func(p searchPatternParams, _ copilot.ToolInvocation) (string, error) {
			return searchFiles(workDir, p)
		})
}

// ── find_symbol ──────────────────────────────────────────────────────────────

type findSymbolParams struct {
	Symbol   string `json:"symbol"    jsonschema:"Identifier name to locate (function, type, constant, variable)"`
	FileGlob string `json:"file_glob" jsonschema:"Restrict search to files matching this glob (e.g. '*.go')"`
}

func findSymbolTool(workDir string) copilot.Tool {
	return copilot.DefineTool(
		"find_symbol",
		"Find where a named symbol (function, type, variable) is defined in the repository. Returns file and line number.",
		func(p findSymbolParams, _ copilot.ToolInvocation) (string, error) {
			if p.Symbol == "" {
				return "", fmt.Errorf("symbol is required")
			}
			// Build a pattern that matches common definition forms across languages.
			// Go: func Foo / type Foo / var Foo / const Foo
			// Generic: class Foo / def Foo / interface Foo / struct Foo
			pattern := fmt.Sprintf(`\b(func|type|var|const|class|def|interface|struct)\s+%s\b`, regexp.QuoteMeta(p.Symbol))
			return searchFiles(workDir, searchPatternParams{
				Pattern:      pattern,
				Path:         ".",
				FileGlob:     p.FileGlob,
				ContextLines: 1,
			})
		})
}

// searchFiles is the shared implementation used by both search_pattern and find_symbol.
func searchFiles(workDir string, p searchPatternParams) (string, error) {
	if p.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	re, err := regexp.Compile(p.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid pattern: %w", err)
	}
	if p.ContextLines < 0 {
		p.ContextLines = 0
	}
	if p.ContextLines > 5 {
		p.ContextLines = 5
	}
	if p.Path == "" {
		p.Path = "."
	}

	target := filepath.Join(workDir, filepath.Clean(p.Path))
	if !strings.HasPrefix(target, workDir) {
		return "", fmt.Errorf("path escapes the repository root")
	}

	const maxMatches = 200
	var sb strings.Builder
	matchCount := 0

	err = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if p.FileGlob != "" {
			matched, _ := filepath.Match(p.FileGlob, d.Name())
			if !matched {
				return nil
			}
		}
		if matchCount >= maxMatches {
			return filepath.SkipAll
		}

		f, err := os.Open(path) // #nosec G304 — path is under workDir
		if err != nil {
			return nil
		}
		defer f.Close()

		var fileLines []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fileLines = append(fileLines, scanner.Text())
		}

		rel, _ := filepath.Rel(workDir, path)
		for i, line := range fileLines {
			if re.MatchString(line) {
				start := i - p.ContextLines
				if start < 0 {
					start = 0
				}
				end := i + p.ContextLines + 1
				if end > len(fileLines) {
					end = len(fileLines)
				}
				for j := start; j < end; j++ {
					prefix := "  "
					if j == i {
						prefix = "> "
					}
					sb.WriteString(fmt.Sprintf("%s%s:%d: %s\n", prefix, rel, j+1, fileLines[j]))
				}
				sb.WriteString("\n")
				matchCount++
				if matchCount >= maxMatches {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if matchCount == 0 {
		return "(no matches)", nil
	}
	if matchCount >= maxMatches {
		sb.WriteString(fmt.Sprintf("\n[truncated: showing first %d matches]\n", maxMatches))
	}
	return sb.String(), nil
}
