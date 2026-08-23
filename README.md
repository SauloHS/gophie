# Gophie

Gophie is a terminal-based coding assistant written in Go, using [Bubble Tea](https://github.com/charmbracelet/bubbletea).

It's an independent, open-source project **inspired by the general concept of terminal-based AI coding assistants** (such as Claude Code and similar tools). Gophie is not affiliated with, endorsed by, or associated with Anthropic, OpenCode, or any other company. All product names, logos, and brands mentioned are property of their respective owners and are used, if at all, for identification purposes only.

Gophie's implementation (UI layout, code, prompts, tool-calling logic) is original work, built from scratch in Go. No code, assets, trademarks, or proprietary material from any third-party product were copied or reused.

## What it does

Gophie is a TUI (terminal user interface) that lets you chat with an LLM and lets it read and write files in your working directory, with your explicit confirmation before any file operation runs.

- Chat interface with Markdown rendering (via [Glamour](https://github.com/charmbracelet/glamour))
- Tool calling: `read_file` and `write_file`, both sandboxed to the current working directory
- Explicit, interactive confirmation before any file is created or modified — you always approve or deny before anything touches disk
- Syntax-highlighted file previews before confirming a write

## Requirements

- Go 1.21 or newer
- An API key for the LLM backend Gophie talks to (currently configured for [OpenCode Zen](https://opencode.ai))

## Installation

```bash
git clone https://github.com/SauloHS/gophie.git
cd gophie
go build .
```

## Usage

Set your API key as an environment variable:

```bash
set OPENCODE_API_KEY="your-api-key-here"
```

Then run:

```bash
gophie.exe
```

Type your request in the input box and press Enter. If Gophie needs to read or write a file to help you, it will show a confirmation prompt first — nothing happens on disk without your approval.

## Project status

Gophie is a personal, work-in-progress hobby project. Expect rough edges, incomplete features, and breaking changes between commits.

## License

Gophie is licensed under the [GNU General Public License v3.0](LICENSE).

## Disclaimer

This project is provided "as is", for educational and personal use, without warranty of any kind. If you are a rights holder and believe any part of this repository infringes on your intellectual property, please open an issue or contact the maintainer directly so it can be addressed promptly.
