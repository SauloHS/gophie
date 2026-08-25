package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

	convNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#0891b2"))
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

	taValue string
	taLines int
	taWidth int

	modelName string

	picking     bool
	pickKind    string // "models" or "resume"
	pickModels  []string
	resumeNames []string
	pickQuery   string
	pickSel     int
	pickLoading bool
	pickErr     error

	named    bool
	naming   bool
	convName string
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

type modelsMsg struct {
	models []string
	err    error
}

type convNameMsg struct {
	name string
	err  error
}

type resumesMsg struct {
	names []string
	err   error
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

const systemPrompt = "You are Gophie. You are a coding assistant running in the terminal. Your goal is to help the user write, understand, debug, and modify code. Rules: Be direct and concise, but not dry. Don't repeat what the user said. When showing code, use code blocks with the language identified. Don't invent filenames, functions, or libraries you're not sure exist. Prefer simple and idiomatic solutions in the language in question, rather than overly 'clever' solutions. If you don't know the answer with certainty, say so clearly instead of risking a wrong answer. Be educated, and don't be rude."

func initialHistory() []apiclient.Message {
	return []apiclient.Message{
		{Role: "system", Content: systemPrompt},
	}
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
		client:  client,
		history: initialHistory(),
		viewport: vp,
		textarea: ta,
		messages: []string{},
		workDir:  dir,
		taLines:  1,

		modelName: "hy3-free",
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
		if m.picking {
			picks := m.filteredPicks()
			switch msg.String() {
			case "up":
				if m.pickSel > 0 {
					m.pickSel--
				}
			case "down":
				if m.pickSel < len(picks)-1 {
					m.pickSel++
				}
			case "backspace":
				r := []rune(m.pickQuery)
				if len(r) > 0 {
					m.pickQuery = string(r[:len(r)-1])
					m.pickSel = 0
				}
			case "enter":
				switch m.pickKind {
				case "resume":
					if m.pickSel < len(picks) {
						name := picks[m.pickSel]
						m.picking = false
						m.loadConversation(name)
					}
					return m, nil
				default: // models
					if m.pickSel < len(picks) {
						m.modelName = picks[m.pickSel]
						m.messages = append(m.messages, hintStyle.Render("model changed to "+m.modelName))
					}
					m.picking = false
				}
			case "esc":
				m.picking = false
			case "ctrl+c":
				return m, tea.Quit
			default:
				if msg.Type == tea.KeyRunes {
					m.pickQuery += string(msg.Runes)
					m.pickSel = 0
				}
				return m, nil
			}
			m.refreshViewport()
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
			if input == "/model" {
				m.textarea.Reset()
				m.openPicker("models")
				return m, fetchModelsCmd(m.client)
			}
			if input == "/resume" {
				m.textarea.Reset()
				m.openPicker("resume")
				return m, listResumesCmd(m.workDir)
			}
			if strings.HasPrefix(input, "/rename") {
				arg := strings.TrimSpace(strings.TrimPrefix(input, "/rename"))
				m.textarea.Reset()
				m.renameConversation(arg)
				return m, nil
			}
			if input == "/clear" && !m.loading {
				m.textarea.Reset()
				m.clearContext()
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
			return m, tea.Batch(callAPI(ctx, m.client, m.history, m.modelName), tickCmd(m.tickGen))
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

		var next tea.Cmd
		if msg.err == nil {
			switch {
			case !m.named && !m.naming:
				// First completed answer: ask the model to name the
				// conversation so it can be saved to .gophie/.
				m.naming = true
				next = nameConversationCmd(m.client, m.modelName, m.history)
			case m.named && m.convName != "":
				if err := m.saveConversation(); err != nil {
					m.messages = append(m.messages, "error saving conversation: "+err.Error())
				}
			}
		}
		m.refreshViewport()
		m.viewport.GotoBottom()
		return m, next
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
		// Keep tool results in the main history immediately so they survive
		// errors/interrupts later in the chain (and land in the saved .json).
		m.history = msg.req.messages
		m.messages = append(m.messages, renderToolBlock(msg.req.call, msg.execErr))
		m.refreshViewport()
		m.viewport.GotoBottom()
		m.loading = true
		return m, advanceAgentCmd(msg.req)
	case toolAbortMsg:
		m.loading = false
		m.confirming = false
		// A denied call must still answer its tool_call or the next request
		// carries a dangling tool_calls message.
		msgs := append(msg.req.messages, apiclient.Message{
			Role:       "tool",
			ToolCallID: msg.req.call.ID,
			Content:    "Error: operation cancelled by the user.",
		})
		msg.req.messages = msgs
		m.history = msgs
		m.messages = append(m.messages, renderToolCancelled(msg.req.call))
		m.refreshViewport()
		m.viewport.GotoBottom()
		return m, nil
	case modelsMsg:
		m.pickLoading = false
		if msg.err != nil {
			m.pickErr = msg.err
		} else {
			m.pickModels = msg.models
			for i, id := range msg.models {
				if id == m.modelName {
					m.pickSel = i
					break
				}
			}
		}
		m.refreshViewport()
		return m, nil
	case convNameMsg:
		m.naming = false
		if msg.err != nil || msg.name == "" || m.named { // m.named: renamed/resumed meanwhile
			return m, nil
		}
		m.convName = msg.name
		m.named = true
		if err := m.saveConversation(); err != nil {
			m.messages = append(m.messages, "error saving conversation: "+err.Error())
			m.refreshViewport()
			m.viewport.GotoBottom()
		}
		return m, nil
	case resumesMsg:
		m.pickLoading = false
		if msg.err != nil {
			m.pickErr = msg.err
		} else {
			m.resumeNames = msg.names
		}
		m.refreshViewport()
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

	value := m.textarea.Value()
	lineCount := calcVisualLines(value, m.textarea.Width())
	if lineCount > maxTextareaLines {
		lineCount = maxTextareaLines
	}
	if lineCount < 1 {
		lineCount = 1
	}

	// Re-run the re-wrap/redraw hack only when the content or geometry
	// actually changed. Navigation keys leave all three untouched, so the
	// cursor keeps its position (SetValue resets it to the end).
	if value != m.taValue || lineCount != m.taLines || m.textarea.Width() != m.taWidth {
		m.taValue = value
		m.taLines = lineCount
		m.taWidth = m.textarea.Width()
		m.textarea.SetHeight(lineCount)
		m.textarea.SetValue(value)
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
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
	pickHeight := 0
	nameHeight := 0
	if m.confirming {
		confirmHeight = lipgloss.Height(m.renderConfirmFooter()) + 2
	}
	if m.picking {
		pickHeight = lipgloss.Height(m.renderPicker()) + 1
	}
	if m.convName != "" {
		nameHeight = 1
	}
	m.viewport.Height = m.height - textareaHeight - spacing - confirmHeight - pickHeight - nameHeight - len(m.suggestionItems())
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
	if m.picking {
		b.WriteString(borderStyle.Width(m.width - 2).Render(m.renderPicker()))
		b.WriteString("\n")
	}
	if m.convName != "" {
		b.WriteString(convNameStyle.Render("◈ " + m.convName))
		b.WriteString("\n")
	}
	for _, s := range m.suggestionItems() {
		b.WriteString(userInputStyle.Render(s.name) + " " + hintStyle.Render(s.desc))
		b.WriteString("\n")
	}
	b.WriteString(borderStyle.Width(m.width - 2).Render(renderTextarea(m.textarea)))
	return b.String()
}

func callAPI(ctx context.Context, client *apiclient.Client, history []apiclient.Message, model string) tea.Cmd {
	return func() tea.Msg {
		messages := append([]apiclient.Message(nil), history...)
		return stepAgent(ctx, client, model, tools.AllTools(), messages)
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
	case "glob_files":
		label = "Glob"
		subtitle = desc
	case "grep_files":
		label = "Grep"
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
		prompt = fmt.Sprintf("Allow Gophie to create %s?", desc)
	case "run_bash":
		prompt = fmt.Sprintf("Allow Gophie to run: %s?", desc)
	case "edit_file":
		prompt = fmt.Sprintf("Allow Gophie to edit %s?", desc)
	case "glob_files":
		prompt = fmt.Sprintf("Allow Gophie to search files matching %s?", desc)
	case "grep_files":
		prompt = fmt.Sprintf("Allow Gophie to search for %s?", desc)
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

func fetchModelsCmd(client *apiclient.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, err := client.Models(ctx)
		return modelsMsg{models: models, err: err}
	}
}

func listResumesCmd(workDir string) tea.Cmd {
	return func() tea.Msg {
		dir := filepath.Join(workDir, ".gophie")
		entries, err := os.ReadDir(dir)
		if err != nil {
			return resumesMsg{err: err}
		}
		type conv struct {
			name string
			mod  time.Time
		}
		var convs []conv
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			var mod time.Time
			if info, err := e.Info(); err == nil {
				mod = info.ModTime()
			}
			convs = append(convs, conv{strings.TrimSuffix(e.Name(), ".json"), mod})
		}
		sort.Slice(convs, func(i, j int) bool { return convs[i].mod.After(convs[j].mod) })
		names := make([]string, len(convs))
		for i, c := range convs {
			names[i] = c.name
		}
		return resumesMsg{names: names}
	}
}

func (m *model) loadConversation(name string) {
	path := filepath.Join(m.workDir, ".gophie", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		m.messages = append(m.messages, "error loading conversation: "+err.Error())
		m.refreshViewport()
		m.viewport.GotoBottom()
		return
	}
	var payload struct {
		Name     string              `json:"name"`
		Model    string              `json:"model"`
		Messages []apiclient.Message `json:"messages"`
	}
	if err := json.Unmarshal(data, &payload); err != nil || len(payload.Messages) == 0 {
		m.messages = append(m.messages, "error loading conversation: invalid format")
		m.refreshViewport()
		m.viewport.GotoBottom()
		return
	}
	m.history = payload.Messages
	m.convName = name
	if payload.Name != "" {
		m.convName = payload.Name
	}
	m.named = true
	m.naming = false
	if payload.Model != "" {
		m.modelName = payload.Model
	}
	m.rebuildTranscript()
	m.refreshViewport()
	m.viewport.GotoBottom()
}

// rebuildTranscript recreates the on-screen messages from a loaded history.
func (m *model) rebuildTranscript() {
	m.messages = nil
	for _, msg := range m.history {
		switch msg.Role {
		case "user":
			m.messages = append(m.messages, userInputStyle.Width(m.width-2).Render(msg.Content))
		case "assistant":
			rendered, err := m.mdRenderer.Render(msg.Content)
			if err != nil {
				rendered = msg.Content
			}
			if txt := strings.TrimSpace(rendered); txt != "" {
				m.messages = append(m.messages, "● "+txt)
			}
		case "tool":
			preview := []rune(strings.ReplaceAll(msg.Content, "\n", " "))
			if len(preview) > 120 {
				preview = append(preview[:120], []rune("...")...)
			}
			m.messages = append(m.messages, hintStyle.Render(" ⎿ "+string(preview)))
		}
	}
}

func (m *model) renameConversation(arg string) {
	if arg == "" {
		m.messages = append(m.messages, hintStyle.Render("usage: /rename <name>"))
		m.refreshViewport()
		return
	}
	newName := sanitizeFilename(arg)
	old := m.convName
	if m.named && old != "" && old != newName {
		oldPath := filepath.Join(m.workDir, ".gophie", old+".json")
		if _, err := os.Stat(oldPath); err == nil {
			newPath := filepath.Join(m.workDir, ".gophie", newName+".json")
			if err := os.Rename(oldPath, newPath); err != nil {
				m.messages = append(m.messages, "error renaming conversation: "+err.Error())
				m.refreshViewport()
				return
			}
		}
	}
	m.convName = newName
	m.named = true
	if len(m.history) > 1 {
		if err := m.saveConversation(); err != nil {
			m.messages = append(m.messages, "error saving conversation: "+err.Error())
		}
	}
	m.messages = append(m.messages, hintStyle.Render("conversation renamed to "+newName))
	m.refreshViewport()
	m.viewport.GotoBottom()
}

func (m *model) clearContext() {
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}
	m.loading = false
	m.confirming = false
	m.history = initialHistory()
	m.messages = []string{hintStyle.Render("context cleared")}
	m.named = false
	m.naming = false
	m.convName = ""
	m.refreshViewport()
}

// nameConversationCmd asks the model for a short filename-friendly name for
// the conversation so it can be persisted under .gophie/.
func nameConversationCmd(client *apiclient.Client, model string, history []apiclient.Message) tea.Cmd {
	return func() tea.Msg {
		var b strings.Builder
		for _, msg := range history {
			if (msg.Role != "user" && msg.Role != "assistant") || strings.TrimSpace(msg.Content) == "" {
				continue
			}
			fmt.Fprintf(&b, "%s: %s\n", msg.Role, msg.Content)
			if b.Len() > 2000 {
				break
			}
		}

		prompt := []apiclient.Message{
			{Role: "system", Content: "You name chat conversations. Reply with ONLY a short name for this conversation: 2 to 4 words, lowercase, hyphen-separated, filename-safe (a-z, 0-9, hyphens only). No quotes, no punctuation, no file extension."},
			{Role: "user", Content: b.String()},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := client.Chat(ctx, prompt, model, nil)
		if err != nil {
			return convNameMsg{err: err}
		}
		if len(resp.Choice) == 0 {
			return convNameMsg{err: fmt.Errorf("api did not return any choice")}
		}
		return convNameMsg{name: sanitizeFilename(resp.Choice[0].Message.Content)}
	}
}

func sanitizeFilename(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	if len(out) > 48 {
		out = out[:48]
	}
	out = strings.Trim(out, "-_")
	if out == "" {
		out = "conversation"
	}
	return out
}

func (m *model) saveConversation() error {
	dir := filepath.Join(m.workDir, ".gophie")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	payload := struct {
		Name     string              `json:"name"`
		Model    string              `json:"model"`
		Messages []apiclient.Message `json:"messages"`
	}{
		Name:     m.convName,
		Model:    m.modelName,
		Messages: m.history,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing conversation: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, m.convName+".json"), data, 0o644)
}

type slashCommand struct {
	name string
	desc string
}

var slashCommands = []slashCommand{
	{"/model", "Change the model"},
	{"/resume", "Resume a saved conversation"},
	{"/rename", "Rename this conversation"},
	{"/clear", "Clear the context"},
}

func (m *model) suggestionItems() []slashCommand {
	if m.picking || m.confirming {
		return nil
	}
	v := m.textarea.Value()
	if !strings.HasPrefix(v, "/") || strings.Contains(v, " ") {
		return nil
	}
	var out []slashCommand
	for _, c := range slashCommands {
		if strings.HasPrefix(c.name, v) {
			out = append(out, c)
		}
	}
	return out
}

// fuzzyScore returns >0 when every rune of query appears in target in order,
// ranking tighter matches higher; 0 means no match.
func fuzzyScore(query, target string) int {
	q := []rune(strings.ToLower(query))
	t := []rune(strings.ToLower(target))
	if len(q) == 0 {
		return 1
	}
	score, qi, consec := 0, 0, 0
	for i, rc := range t {
		if qi < len(q) && rc == q[qi] {
			score += 10 + consec*5
			if i == 0 {
				score += 15
			} else {
				switch t[i-1] {
				case '-', '_', ' ', '/', '.':
					score += 10
				}
			}
			consec++
			qi++
		} else {
			consec = 0
		}
	}
	if qi < len(q) {
		return 0
	}
	return score
}

func filterFuzzy(items []string, query string) []string {
	if query == "" {
		return items
	}
	type scored struct {
		item  string
		score int
	}
	var matches []scored
	for _, it := range items {
		if s := fuzzyScore(query, it); s > 0 {
			matches = append(matches, scored{it, s})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
	out := make([]string, len(matches))
	for i, mt := range matches {
		out[i] = mt.item
	}
	return out
}

func (m *model) filteredPicks() []string {
	if m.pickKind == "resume" {
		return filterFuzzy(m.resumeNames, m.pickQuery)
	}
	return filterFuzzy(m.pickModels, m.pickQuery)
}

func (m *model) openPicker(kind string) {
	m.picking = true
	m.pickKind = kind
	m.pickQuery = ""
	m.pickSel = 0
	m.pickLoading = true
	m.pickErr = nil
	if kind == "models" {
		for i, id := range m.pickModels {
			if id == m.modelName {
				m.pickSel = i
				break
			}
		}
	}
	m.refreshViewport()
}

func (m *model) renderPicker() string {
	width := m.width - 2
	topLine := confirmBlueStyle.Render(strings.Repeat("─", width))
	dashedLine := hintStyle.Render(strings.Repeat("- ", width/2))

	label := confirmBlueStyle.Bold(true).Render("Select Model")
	footer := hintStyle.Render("↑/↓ move · type to filter · enter select · esc cancel")
	if m.pickKind == "resume" {
		label = confirmBlueStyle.Bold(true).Render("Resume Conversation")
		footer = hintStyle.Render("↑/↓ move · type to filter · enter resume · esc cancel")
	}

	lines := []string{
		topLine,
		label,
		hintStyle.Render("search: "+m.pickQuery+"▏"),
	}
	if m.pickKind == "models" {
		lines = append(lines, hintStyle.Render("current: "+m.modelName))
	}
	lines = append(lines, dashedLine)

	var body string
	switch {
	case m.pickErr != nil:
		body = toolErrStyle.Render("error: " + m.pickErr.Error())
	case m.pickLoading:
		body = hintStyle.Render("Loading...")
	default:
		items := m.filteredPicks()
		switch {
		case len(items) == 0:
			body = hintStyle.Render("No matches.")
		default:
			const visible = 8
			start := min(max(m.pickSel-visible/2, 0), max(len(items)-visible, 0))
			end := min(start+visible, len(items))
			var opts []string
			for i := start; i < end; i++ {
				cursor := "  "
				number := fmt.Sprintf("%d. ", i+1)
				text := items[i]
				if i == m.pickSel {
					cursor = confirmBlueStyle.Render("❯ ")
					text = confirmBlueStyle.Render(text)
				}
				opts = append(opts, cursor+number+text)
			}
			body = strings.Join(opts, "\n")
		}
	}
	lines = append(lines, body, footer)

	return strings.Join(lines, "\n")
}

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
