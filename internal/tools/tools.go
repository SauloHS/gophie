// Package tools manages api tool calls
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gophie/internal/apiclient"
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

const bashTimeout = 120 * time.Second

const maxReadBytes = 200_000

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
	default:
		return call.Function.Name
	}
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
	}
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

func WriteContent(call apiclient.ToolCall) string {
	return parseWriteArgs(call).Content
}
