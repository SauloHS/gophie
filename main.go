package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gophie/internal/apiclient"
	"gophie/internal/tools"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const mascot = `█▀▀▀▀ █▀▀█ █▀▀█ █  █ █ █▀▀
█ ▀▀█ █  █ █▀▀▀ █▀▀█ █ █▀▀
▀▀▀▀▀ ▀▀▀▀ ▀    ▀  ▀ ▀ ▀▀▀
`

var langByExt = map[string]string{
	".go":         "go",
	".py":         "python",
	".c":          "c",
	".cpp":        "cpp",
	".js":         "javascript",
	".ts":         "typescript",
	".rs":         "rust",
	".java":       "java",
	".rb":         "ruby",
	".php":        "php",
	".swift":      "swift",
	".kt":         "kotlin",
	".scala":      "scala",
	".sh":         "bash",
	".bash":       "bash",
	".zsh":        "zsh",
	".fish":       "fish",
	".ps1":        "powershell",
	".lua":        "lua",
	".pl":         "perl",
	".r":          "r",
	".m":          "objective-c",
	".mm":         "objective-cpp",
	".cs":         "csharp",
	".vb":         "vb.net",
	".groovy":     "groovy",
	".gradle":     "gradle",
	".clj":        "clojure",
	".cljs":       "clojurescript",
	".edn":        "edn",
	".ex":         "elixir",
	".exs":        "elixir",
	".erl":        "erlang",
	".hrl":        "erlang",
	".hs":         "haskell",
	".lhs":        "haskell",
	".ml":         "ocaml",
	".mli":        "ocaml",
	".fs":         "fsharp",
	".fsi":        "fsharp",
	".fsx":        "fsharp",
	".jl":         "julia",
	".nim":        "nim",
	".nims":       "nim",
	".cr":         "crystal",
	".d":          "d",
	".zig":        "zig",
	".v":          "vlang",
	".pas":        "pascal",
	".pp":         "pascal",
	".asm":        "assembly",
	".s":          "assembly",
	".html":       "html",
	".htm":        "html",
	".css":        "css",
	".scss":       "scss",
	".sass":       "sass",
	".less":       "less",
	".json":       "json",
	".xml":        "xml",
	".yaml":       "yaml",
	".yml":        "yaml",
	".toml":       "toml",
	".ini":        "ini",
	".cfg":        "ini",
	".conf":       "conf",
	".sql":        "sql",
	".md":         "markdown",
	".markdown":   "markdown",
	".tex":        "latex",
	".rst":        "rst",
	".adoc":       "asciidoc",
	".vim":        "vim",
	".emacs":      "elisp",
	".el":         "elisp",
	".lisp":       "lisp",
	".scm":        "scheme",
	".rkt":        "racket",
	".prolog":     "prolog",
	".pro":        "prolog",
	".dart":       "dart",
	".go.mod":     "go",
	".go.sum":     "go",
	".lock":       "lock",
	".gemfile":    "ruby",
	".dockerfile": "dockerfile",
	".makefile":   "makefile",
	".cmake":      "cmake",
}

func detectLang(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	return langByExt[ext]
}

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderLeft(false).
			BorderRight(false).
			BorderForeground(lipgloss.Color("#888888"))

	assistantMarkerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D97757"))
	toolOkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3FB950"))

	toolErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F85149"))

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D97757"))

	confirmBlueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A0A0BB"))

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	userInputStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3A3A3A")).
			Padding(0, 1)
)

func renderLeftPanel(model, dir string, colWidth int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Center).
		Width(colWidth).
		Render("Welcome Back!")
	art := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D97757")).
		Align(lipgloss.Center).
		Width(colWidth).
		Render(mascot)
	info := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Align(lipgloss.Center).
		Width(colWidth).
		Render(model + "· Gophie\n" + dir)
	return lipgloss.JoinVertical(lipgloss.Center, title, "", art, "", info)
}

func renderRightPanel(colWidth int) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#D97757")).
		Width(colWidth)
	bodyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		Width(colWidth)
	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true).
		Width(colWidth)
	tips := lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render("Tips for getting started"),
		bodyStyle.Render("Ask Gophie to analyze your codebase"),
	)
	whatsNew := lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render("What's new"),
		bodyStyle.Render("First and only version released"),
		bodyStyle.Render("TUI written using Go and BubbleTea"),
		dimStyle.Render("https://github.com/SauloHS/gophie"),
	)
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).Render(strings.Repeat("─", colWidth))
	return lipgloss.JoinVertical(lipgloss.Left, tips, "", separator, "", whatsNew)
}

func renderWelcomeBox(width int, model, dir string) string {
	if width < 20 {
		return ""
	}
	boxWidth := width - 4
	innerWidth := boxWidth - 28
	colWidth := (innerWidth - 3) / 2

	left := renderLeftPanel(model, dir, colWidth)
	right := renderRightPanel(colWidth)

	panelHeight := lipgloss.Height(right)
	if lipgloss.Height(left) > panelHeight {
		panelHeight = lipgloss.Height(left)
	}
	dividerLines := make([]string, panelHeight)
	for i := range dividerLines {
		dividerLines[i] = "│"
	}
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).
		Padding(0, 1).
		Render(strings.Join(dividerLines, "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)

	borderColor := lipgloss.Color("#D97757")
	titleText := " Gophie v1.0.1 "
	totalWidth := boxWidth + 1
	topLineFill := totalWidth - lipgloss.Width(titleText) - 2
	if topLineFill < 0 {
		topLineFill = 0
	}
	topBorder := "╭─" + titleText + strings.Repeat("─", topLineFill) + "╮"
	styledTop := lipgloss.NewStyle().Foreground(borderColor).Render(topBorder)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(borderColor).
		Padding(0, 2).
		Width(boxWidth)

	styledBody := boxStyle.Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, styledTop, styledBody)
}

type model struct {
	client       *apiclient.Client
	history      []apiclient.Message
	viewport     viewport.Model
	textarea     textarea.Model
	loading      bool
	messages     []string
	width        int
	height       int
	workDir      string
	spinnerFrame int
	mdRenderer   *glamour.TermRenderer
	cancelFunc   context.CancelFunc

	confirming bool
	pending    toolRequestMsg
	confirmSel int

	tickGen int
}

type toolRequestMsg struct {
	messages      []apiclient.Message
	assistant     apiclient.Message
	call          apiclient.ToolCall
	callIndex     int
	toolCount     int
	client        *apiclient.Client
	execCtx       context.Context
	modelName     string
	tools         []apiclient.Tool
	precedingText string
}

type toolDoneMsg struct {
	req     toolRequestMsg
	result  string
	execErr error
}

type toolAbortMsg struct {
	req toolRequestMsg
}

type responseMsg struct {
	content string
	history []apiclient.Message
	err     error
}

var spinnerFrames = []string{"·", "✶", "✢", "✽"}

type tickMsg struct {
	t   time.Time
	gen int
}

func tickCmd(gen int) tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{t: t, gen: gen}
	})
}

func initialModel(client *apiclient.Client) model {
	ta := textarea.New()
	ta.Placeholder = "What should Gophie do?"
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.SetHeight(1)
	ta.Prompt = ""

	vp := viewport.New(80, 20)

	dir, _ := os.Getwd()

	return model{
		client: client,
		history: []apiclient.Message{
			{Role: "system", Content: "You are Gophie. You are a coding assistant running in the terminal. Your goal is to help the user write, understand, debug, and modify code. Rules: Be direct and concise, but not dry. Don't repeat what the user said. When showing code, use code blocks with the language identified. Don't invent filenames, functions, or libraries you're not sure exist. Prefer simple and idiomatic solutions in the language in question, rather than overly 'clever' solutions. If you don't know the answer with certainty, say so clearly instead of risking a wrong answer. Be educated, and don't be rude."},
		},
		viewport: vp,
		textarea: ta,
		messages: []string{},
		workDir:  dir,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var taCmd, vpCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			switch msg.Type {
			case tea.KeyUp:
				if m.confirmSel > 0 {
					m.confirmSel--
				}
				m.refreshViewport()
				return m, nil
			case tea.KeyDown:
				if m.confirmSel < 1 {
					m.confirmSel++
				}
				m.refreshViewport()
				return m, nil
			case tea.KeyEnter:
				return m.resolveConfirmation()
			case tea.KeyEsc:
				m.confirmSel = 2
				return m.resolveConfirmation()
			}
			switch msg.String() {
			case "1":
				m.confirmSel = 0
				return m.resolveConfirmation()
			case "2":
				m.confirmSel = 1
				return m.resolveConfirmation()
			}
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEsc:
			if m.loading && m.cancelFunc != nil {
				m.cancelFunc()
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyEnter:
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" || m.loading {
				return m, nil
			}
			m.history = append(m.history, apiclient.Message{Role: "user", Content: input})
			m.messages = append(m.messages, userInputStyle.Width(m.width-2).Render(input))
			m.refreshViewport()
			m.viewport.GotoBottom()
			m.textarea.Reset()
			m.textarea.SetHeight(1)
			m.recalcLayout()

			ctx, cancel := context.WithCancel(context.Background())
			m.cancelFunc = cancel

			m.tickGen++
			m.loading = true
			return m, tea.Batch(callAPI(ctx, m.client, m.history), tickCmd(m.tickGen))
		}
	case responseMsg:
		m.loading = false

		if msg.err != nil {
			if errors.Is(msg.err, context.Canceled) {
				m.messages = append(m.messages, loadingStyle.Render(" ⎿ Interrupted by user"))
			} else {
				m.messages = append(m.messages, "error: "+msg.err.Error())
			}
		} else {
			if len(msg.history) > 0 {
				m.history = msg.history
			} else {
				m.history = append(m.history, apiclient.Message{Role: "assistant", Content: msg.content})
			}
			rendered, err := m.mdRenderer.Render(msg.content)
			if err != nil {
				rendered = msg.content
			}
			m.messages = append(m.messages, "● "+strings.TrimSpace(rendered))
		}
		m.refreshViewport()
		m.viewport.GotoBottom()
		return m, nil
	case toolRequestMsg:
		m.loading = false
		m.confirming = true
		m.pending = msg
		if msg.callIndex == 0 && strings.TrimSpace(msg.precedingText) != "" {
			rendered, err := m.mdRenderer.Render(msg.precedingText)
			if err != nil {
				rendered = msg.precedingText
			}
			m.messages = append(m.messages, "● "+strings.TrimSpace(rendered))
		}
		m.refreshViewport()
		m.viewport.GotoBottom()
		return m, nil
	case toolDoneMsg:
		m.messages = append(m.messages, renderToolBlock(msg.req.call, msg.execErr))
		m.refreshViewport()
		m.viewport.GotoBottom()
		m.loading = true
		return m, advanceAgentCmd(msg.req)
	case toolAbortMsg:
		m.loading = false
		m.confirming = false
		m.messages = append(m.messages, renderToolCancelled(msg.req.call))
		m.refreshViewport()
		m.viewport.GotoBottom()
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width - 4)
		m.recalcLayout()
		m.refreshViewport()
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(m.width-4),
		)
		m.mdRenderer = renderer
		return m, nil
	case tickMsg:
		if m.loading && msg.gen == m.tickGen {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			m.refreshViewport()
			return m, tickCmd(m.tickGen)
		}
		return m, nil
	}

	switch msg.(type) {
	case tea.KeyMsg:
		m.textarea, taCmd = m.textarea.Update(msg)
	case tea.MouseMsg:
		m.viewport, vpCmd = m.viewport.Update(msg)
	default:
		m.textarea, taCmd = m.textarea.Update(msg)
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	lineCount := calcVisualLines(m.textarea.Value(), m.textarea.Width())
	if lineCount > 9 {
		lineCount = 9
	}
	if lineCount < 1 {
		lineCount = 1
	}
	value := m.textarea.Value()
	m.textarea.SetHeight(lineCount)
	m.textarea.SetValue(value)
	m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.recalcLayout()
	return m, tea.Batch(taCmd, vpCmd)
}

func (m model) resolveConfirmation() (tea.Model, tea.Cmd) {
	m.confirming = false
	m.recalcLayout()
	switch m.confirmSel {
	case 0:
		m.tickGen++
		m.loading = true
		m.refreshViewport()
		return m, tea.Batch(resumeToolCmd(m.pending, true), tickCmd(m.tickGen))
	default:
		m.loading = false
		return m, resumeToolCmd(m.pending, false)
	}
}

func (m *model) renderFileLines(path, content string) string {
	lang := detectLang(path)
	fenced := "```" + lang + "\n" + content + "\n```"

	rendered, err := m.mdRenderer.Render(fenced)
	if err != nil {
		rendered = content
	}
	rendered = strings.Trim(rendered, "\n")

	lines := strings.Split(rendered, "\n")
	var b strings.Builder
	for i, line := range lines {
		numStyle := hintStyle.Render(fmt.Sprintf("%2d ", i+1))
		b.WriteString(numStyle + line + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *model) recalcLayout() {
	textareaHeight := m.textarea.Height() + 2
	spacing := 2
	confirmHeight := 0
	if m.confirming {
		confirmHeight = lipgloss.Height(m.renderConfirmFooter()) + 2
	}
	m.viewport.Height = m.height - textareaHeight - spacing - confirmHeight
}

func calcVisualLines(text string, width int) int {
	if width <= 0 || text == "" {
		return 1
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	return strings.Count(wrapped, "\n") + 1
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	b.WriteString(m.viewport.View())
	b.WriteString("\n\n")
	if m.confirming {
		b.WriteString(m.renderConfirmFooter())
		b.WriteString("\n\n")
	}
	b.WriteString(borderStyle.Width(m.width - 2).Render(renderTextarea(m.textarea)))
	return b.String()
}

func callAPI(ctx context.Context, client *apiclient.Client, history []apiclient.Message) tea.Cmd {
	return func() tea.Msg {
		messages := append([]apiclient.Message(nil), history...)
		return stepAgent(ctx, client, "hy3-free", tools.AllTools(), messages)
	}
}

func renderToolCancelled(call apiclient.ToolCall) string {
	line1 := toolErrStyle.Render("●") + " " + tools.CallLabel(call)
	line2 := hintStyle.Render(" ⎿ ") + toolErrStyle.Render("Operation cancelled by the user.")
	return line1 + "\n" + line2
}

func stepAgent(ctx context.Context, client *apiclient.Client, modelName string, toolsList []apiclient.Tool, messages []apiclient.Message) tea.Msg {
	resp, err := client.Chat(ctx, messages, modelName, toolsList)
	if err != nil {
		return responseMsg{err: err}
	}
	if len(resp.Choice) == 0 {
		return responseMsg{err: fmt.Errorf("api did not return any choice")}
	}

	assistantMsg := resp.Choice[0].Message
	if len(assistantMsg.ToolCalls) == 0 {
		messages = append(messages, assistantMsg)
		return responseMsg{content: assistantMsg.Content, history: messages}
	}

	messages = append(messages, assistantMsg)
	return toolRequestMsg{
		messages:      messages,
		assistant:     assistantMsg,
		call:          assistantMsg.ToolCalls[0],
		callIndex:     0,
		toolCount:     len(assistantMsg.ToolCalls),
		client:        client,
		execCtx:       ctx,
		modelName:     modelName,
		tools:         toolsList,
		precedingText: assistantMsg.Content,
	}
}

func resumeToolCmd(req toolRequestMsg, approve bool) tea.Cmd {
	return func() tea.Msg {
		if !approve {
			return toolAbortMsg{req: req}
		}
		result, execErr := tools.Execute(req.call)
		content := result
		if execErr != nil {
			content = "Error: " + execErr.Error()
		}
		req.messages = append(req.messages, apiclient.Message{
			Role:       "tool",
			ToolCallID: req.call.ID,
			Content:    content,
		})
		return toolDoneMsg{req: req, result: result, execErr: execErr}
	}
}

func advanceAgentCmd(req toolRequestMsg) tea.Cmd {
	return func() tea.Msg {
		nextIndex := req.callIndex + 1
		if nextIndex < req.toolCount {
			req.call = req.assistant.ToolCalls[nextIndex]
			req.callIndex = nextIndex
			return req
		}
		return stepAgent(req.execCtx, req.client, req.modelName, req.tools, req.messages)
	}
}

func renderToolBlock(call apiclient.ToolCall, execErr error) string {
	color := toolOkStyle
	if execErr != nil {
		color = toolErrStyle
	}
	line1 := color.Render("●") + " " + tools.CallLabel(call)
	line2 := hintStyle.Render(" ⎿ ") + tools.CallResult(call, execErr)
	return line1 + "\n" + line2
}

func (m *model) renderConfirmHeader() string {
	call := m.pending.call
	desc := tools.Describe(call)

	topLine := confirmBlueStyle.Render(strings.Repeat("─", m.width-2))
	dashedLine := hintStyle.Render(strings.Repeat("- ", (m.width-2)/2))

	var label, subtitle, body string

	switch call.Function.Name {
	case "write_file":
		label = "Create file"
		subtitle = desc
		body = m.renderFileLines(desc, tools.WriteContent(call))
	case "edit_file":
		label = "Edit file"
		subtitle = desc
		body = m.renderEditDiff(call)
	case "run_bash":
		label = "Run Command"
		subtitle = desc
	default: // read_file
		label = "Read File"
		subtitle = desc
	}

	styledLabel := confirmBlueStyle.Bold(true).Render(label)
	styledSubtitle := hintStyle.Render(subtitle)

	lines := []string{topLine, styledLabel, styledSubtitle, dashedLine}
	if body != "" {
		lines = append(lines, body, dashedLine)
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderEditDiff(call apiclient.ToolCall) string {
	oldStr, newStr := tools.EditPreview(call)

	var b strings.Builder
	for _, line := range strings.Split(oldStr, "\n") {
		b.WriteString(toolErrStyle.Render("- "+line) + "\n")
	}
	for _, line := range strings.Split(newStr, "\n") {
		b.WriteString(toolOkStyle.Render("+ "+line) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m *model) renderConfirmFooter() string {
	call := m.pending.call
	desc := tools.Describe(call)

	var prompt string
	switch call.Function.Name {
	case "write_file":
		prompt = fmt.Sprintf("Do you want to create %s?", desc)
	case "run_bash":
		prompt = fmt.Sprintf("Allow Gophie to run: %s?", desc)
	case "edit_file":
		prompt = fmt.Sprintf("Do you want to edit %s?", desc)
	default:
		prompt = fmt.Sprintf("Allow Gophie to read %s?", desc)
	}
	options := []string{"Yes", "No"}
	var optLines []string
	for i, opt := range options {
		cursor := "  "
		number := fmt.Sprintf("%d. ", i+1)
		optLabel := opt
		if i == m.confirmSel {
			cursor = confirmBlueStyle.Render("❯ ")
			optLabel = confirmBlueStyle.Render(opt)
		}
		optLines = append(optLines, cursor+number+optLabel)
	}

	lines := []string{prompt}
	lines = append(lines, optLines...)
	return strings.Join(lines, "\n")
}

const maxTextareaLines = 9

func renderTextarea(ta textarea.Model) string {
	raw := ta.View()
	lines := strings.Split(raw, "\n")

	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))

	for i := range lines {
		if i == 0 {
			lines[i] = promptStyle.Render("❯ ") + lines[i]
		} else {
			lines[i] = "  " + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func (m *model) refreshViewport() {
	welcomeBox := renderWelcomeBox(m.width, "opencode_zen/hy3-free ", m.workDir)
	content := welcomeBox
	if len(m.messages) > 0 {
		content += "\n\n" + strings.Join(m.messages, "\n\n")
	}
	if m.loading {
		content += "\n" + loadingStyle.Render("  "+spinnerFrames[m.spinnerFrame]+" Gophieing...")
	}
	if m.confirming {
		content += "\n\n" + m.renderConfirmHeader()
	}

	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func main() {
	apiKey := os.Getenv("OPENCODE_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENCODE_API_KEY variable not set")
		os.Exit(1)
	}

	client := apiclient.NewClient(apiKey)

	p := tea.NewProgram(initialModel(client), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
