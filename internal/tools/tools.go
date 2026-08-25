// Package tools manages api tool calls
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gophie/internal/apiclient"

	"github.com/bmatcuk/doublestar/v4"
)

type ReadArgs struct {
	Path string `json:"path"`
}

type WriteArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type BashArgs struct {
	Command string `json:"command"`
}

type EditArgs struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

type GrepArgs struct {
	Pattern     string `json:"pattern"`
	FilePattern string `json:"file_pattern"`
}

const maxGrepResults = 100

type GrepMatch struct {
	Path string
	Line int
	Text string
}

func GrepFiles(args GrepArgs) ([]GrepMatch, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting working dir: %w", err)
	}

	re, err := regexp.Compile(args.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %s: %w", args.Pattern, err)
	}

	filePattern := args.FilePattern
	if filePattern == "" {
		filePattern = "**/*"
	}

	fsys := os.DirFS(workDir)
	files, err := doublestar.Glob(fsys, filePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid file_pattern %s: %w", filePattern, err)
	}

	var matches []GrepMatch
	for _, relPath := range files {
		info, err := fs.Stat(fsys, relPath)
		if err != nil || info.IsDir() {
			continue
		}

		data, err := fs.ReadFile(fsys, relPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				matches = append(matches, GrepMatch{Path: relPath, Line: i + 1, Text: line})
				if len(matches) >= maxGrepResults {
					return matches, nil
				}
			}
		}
	}

	return matches, nil
}

const bashTimeout = 120 * time.Second

const maxReadBytes = 200_000

type GlobArgs struct {
	Pattern string `json:"pattern"`
}

const maxGlobResults = 200

func GlobFiles(args GlobArgs) ([]string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting working dir: %w", err)
	}

	matches, err := doublestar.Glob(os.DirFS(workDir), args.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %s: %w", args.Pattern, err)
	}

	if len(matches) > maxGlobResults {
		matches = matches[:maxGlobResults]
	}

	return matches, nil
}

func ReadFile(args ReadArgs) (string, int, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", 0, fmt.Errorf("error getting working dir: %w", err)
	}

	targetPath := args.Path
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(workDir, targetPath)
	}

	cleanPath, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return "", 0, fmt.Errorf("invalid path %s: %w", args.Path, err)
	}

	rel, err := filepath.Rel(workDir, cleanPath)
	if err != nil {
		return "", 0, fmt.Errorf("invalid path %s: %w", args.Path, err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", 0, fmt.Errorf("cannot read files outside working directory")
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", 0, fmt.Errorf("error stating file %s: %w", args.Path, err)
	}

	if info.Size() > maxReadBytes {
		return "", 0, fmt.Errorf("file too large: %d bytes, max is %d", info.Size(), maxReadBytes)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", 0, fmt.Errorf("error reading file %s: %w", args.Path, err)
	}

	lines := strings.Split(string(data), "\n")
	var b strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&b, "%d\t%s\n", i+1, line)
	}

	return b.String(), len(lines), nil
}

func RunBash(args BashArgs) (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("error getting working dir: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), bashTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cmd", "/C", args.Command)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("command timed out after %s", bashTimeout)
	}
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

func EditFile(args EditArgs) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working dir: %w", err)
	}

	targetPath := args.Path
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(workDir, targetPath)
	}

	cleanPath, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return fmt.Errorf("invalid path %s: %w", args.Path, err)
	}

	rel, err := filepath.Rel(workDir, cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path %s: %w", args.Path, err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("cannot edit files outside working directory")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("error reading file %s: %w", args.Path, err)
	}
	content := string(data)

	count := strings.Count(content, args.OldString)
	if count == 0 {
		return fmt.Errorf("old_string not found in %s", args.Path)
	}
	if count > 1 {
		return fmt.Errorf("old_string appears %d times in %s, must be unique", count, args.Path)
	}

	newContent := strings.Replace(content, args.OldString, args.NewString, 1)

	if err := os.WriteFile(cleanPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("error writing file %s: %w", args.Path, err)
	}

	return nil
}

func WriteFile(args WriteArgs) (int, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("error getting working dir: %w", err)
	}

	targetPath := args.Path
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(workDir, targetPath)
	}

	cleanPath, err := filepath.Abs(filepath.Clean(targetPath))
	if err != nil {
		return 0, fmt.Errorf("invalid path %s: %w", args.Path, err)
	}

	rel, err := filepath.Rel(workDir, cleanPath)
	if err != nil {
		return 0, fmt.Errorf("invalid path %s: %w", args.Path, err)
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return 0, fmt.Errorf("cannot write files outside working directory")
	}

	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return 0, fmt.Errorf("error creating directory for %s: %w", args.Path, err)
	}

	if err := os.WriteFile(cleanPath, []byte(args.Content), 0o644); err != nil {
		return 0, fmt.Errorf("error writing file %s: %w", args.Path, err)
	}

	lineCount := strings.Count(args.Content, "\n") + 1
	return lineCount, nil
}

func Execute(call apiclient.ToolCall) (string, error) {
	switch call.Function.Name {
	case "read_file":
		var args ReadArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error interpreting arguments: %w", err)
		}
		result, _, err := ReadFile(args)
		return result, err
	case "write_file":
		var args WriteArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error interpreting arguments: %w", err)
		}
		lineCount, err := WriteFile(args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Wrote %d lines to %s.", lineCount, args.Path), nil
	case "run_bash":
		var args BashArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error interpreting arguments: %w", err)
		}
		return RunBash(args)
	case "edit_file":
		var args EditArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error interpreting arguments: %w", err)
		}
		if err := EditFile(args); err != nil {
			return "", err
		}
		return fmt.Sprintf("Edited %s successfully.", args.Path), nil
	case "glob_files":
		var args GlobArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error interpreting arguments: %w", err)
		}
		matches, err := GlobFiles(args)
		if err != nil {
			return "", err
		}
		if len(matches) == 0 {
			return "No files matched.", nil
		}
		return strings.Join(matches, "\n"), nil

	case "grep_files":
		var args GrepArgs
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("error interpreting arguments: %w", err)
		}
		matches, err := GrepFiles(args)
		if err != nil {
			return "", err
		}
		if len(matches) == 0 {
			return "No matches found.", nil
		}
		var b strings.Builder
		for _, mtch := range matches {
			fmt.Fprintf(&b, "%s:%d: %s\n", mtch.Path, mtch.Line, mtch.Text)
		}
		return strings.TrimRight(b.String(), "\n"), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", call.Function.Name)
	}
}

func Describe(call apiclient.ToolCall) string {
	switch call.Function.Name {
	case "read_file":
		return parseReadArgs(call).Path
	case "write_file":
		return parseWriteArgs(call).Path
	case "run_bash":
		return parseBashArgs(call).Command
	case "edit_file":
		return parseEditArgs(call).Path
	case "glob_files":
		return parseGlobArgs(call).Pattern
	case "grep_files":
		return parseGrepArgs(call).Pattern
	default:
		return call.Function.Name
	}
}

func parseGlobArgs(call apiclient.ToolCall) GlobArgs {
	var args GlobArgs
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	return args
}

func parseGrepArgs(call apiclient.ToolCall) GrepArgs {
	var args GrepArgs
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	return args
}

func parseEditArgs(call apiclient.ToolCall) EditArgs {
	var args EditArgs
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	return args
}

func parseBashArgs(call apiclient.ToolCall) BashArgs {
	var args BashArgs
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	return args
}

func parseWriteArgs(call apiclient.ToolCall) WriteArgs {
	var args WriteArgs
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	return args
}

func CallLabel(call apiclient.ToolCall) string {
	switch call.Function.Name {
	case "read_file":
		return "Read(" + Describe(call) + ")"
	case "write_file":
		return "Write(" + Describe(call) + ")"
	case "run_bash":
		return "Bash(" + Describe(call) + ")"
	case "edit_file":
		return "Edit(" + Describe(call) + ")"
	case "glob_files":
		return "Glob(" + Describe(call) + ")"
	case "grep_files":
		return "Grep(" + Describe(call) + ")"
	default:
		return call.Function.Name + "(" + Describe(call) + ")"
	}
}

func CallResult(call apiclient.ToolCall, execErr error) string {
	switch call.Function.Name {
	case "read_file":
		if execErr != nil {
			return "Read failed: " + execErr.Error()
		}
		_, lineCount, err := ReadFile(parseReadArgs(call))
		if err != nil {
			return "Read failed: " + err.Error()
		}
		return fmt.Sprintf("Read %d lines.", lineCount)
	case "write_file":
		if execErr != nil {
			return "Write failed: " + execErr.Error()
		}
		return fmt.Sprintf("Wrote %d lines to %s.", strings.Count(parseWriteArgs(call).Content, "\n")+1, Describe(call))
	case "run_bash":
		if execErr != nil {
			return "Command failed: " + execErr.Error()
		}
		return "Command executed sucessfully."
	case "edit_file":
		if execErr != nil {
			return "Edit failed: " + execErr.Error()
		}
		return "File edited successfully."
	case "glob_files":
		if execErr != nil {
			return "Glob failed: " + execErr.Error()
		}
		matches, _ := GlobFiles(parseGlobArgs(call))
		return fmt.Sprintf("Found %d file(s).", len(matches))
	case "grep_files":
		if execErr != nil {
			return "Grep failed: " + execErr.Error()
		}
		matches, _ := GrepFiles(parseGrepArgs(call))
		return fmt.Sprintf("Found %d match(es).", len(matches))
	default:
		if execErr != nil {
			return "failed: " + execErr.Error()
		}
		return "done."
	}
}

func parseReadArgs(call apiclient.ToolCall) ReadArgs {
	var args ReadArgs
	_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
	return args
}

func AllTools() []apiclient.Tool {
	return []apiclient.Tool{
		readTool(),
		writeTool(),
		bashTool(),
		editTool(),
		globTool(),
		grepTool(),
	}
}

func editTool() apiclient.Tool {
	return apiclient.Tool{
		Type: "function",
		Function: apiclient.ToolFunction{
			Name:        "edit_file",
			Description: "Replaces an exact, unique block of text in an existing file with new text. The old_string must match the file content exactly (including whitespace) and appear exactly once — use this instead of write_file when making a small change to an existing file, to avoid rewriting the whole content.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path of the file to edit, relative to current working dir.",
					},
					"old_string": map[string]any{
						"type":        "string",
						"description": "The exact text to find and replace. Must be unique within the file.",
					},
					"new_string": map[string]any{
						"type":        "string",
						"description": "The text to replace old_string with.",
					},
				},
				"required": []string{"path", "old_string", "new_string"},
			},
		},
	}
}

func EditPreview(call apiclient.ToolCall) (string, string) {
	args := parseEditArgs(call)
	return args.OldString, args.NewString
}

func writeTool() apiclient.Tool {
	return apiclient.Tool{
		Type: "function",
		Function: apiclient.ToolFunction{
			Name:        "write_file",
			Description: "Writes content to a file, creating it if it doesn't exist or overwriting it if it does.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path of the file to write, relative to current working dir.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Full content to write to the file.",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

func readTool() apiclient.Tool {
	return apiclient.Tool{
		Type: "function",
		Function: apiclient.ToolFunction{
			Name:        "read_file",
			Description: "Reads file content and returns it as a string, with line numbers.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path of the file to be read, relative to current working dir.",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func bashTool() apiclient.Tool {
	return apiclient.Tool{
		Type: "function",
		Function: apiclient.ToolFunction{
			Name:        "run_bash",
			Description: "Runs a shell command on Windows (via cmd.exe) in the current working directory. The command times out after 120 seconds. If you need more time than this, ask the user to run in a separate shell. Returns combined stdout and stderr output.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command to execute.",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

func globTool() apiclient.Tool {
	return apiclient.Tool{
		Type: "function",
		Function: apiclient.ToolFunction{
			Name:        "glob_files",
			Description: "Finds files matching a glob pattern (supports ** for recursive directory search, e.g. **/*.go). Returns matching file paths relative to the working directory.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Glob pattern to match, e.g. '**/*.go' or 'src/*.ts'.",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func grepTool() apiclient.Tool {
	return apiclient.Tool{
		Type: "function",
		Function: apiclient.ToolFunction{
			Name:        "grep_files",
			Description: "Searches file contents for a regex pattern. Optionally restrict to files matching file_pattern (glob, supports **). Returns matching lines as path:line: content.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Regular expression to search for.",
					},
					"file_pattern": map[string]any{
						"type":        "string",
						"description": "Optional glob pattern to restrict which files are searched, e.g. '**/*.go'. Defaults to all files.",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}
}

func WriteContent(call apiclient.ToolCall) string {
	return parseWriteArgs(call).Content
}
