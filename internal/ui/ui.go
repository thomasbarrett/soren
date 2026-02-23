package ui

import (
	"fmt"
	"strings"

	"github.com/thomasbarrett/soren/internal/agent"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/openai/openai-go"
)

// Mode represents the current interaction mode
type Mode int

const (
	ModeNormal Mode = iota
	ModePlan
	ModeAutoAccept
)

// Styles for the UI
var (
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	purpleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	userStyle   = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "255", Dark: "255"}).
			Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "0"})
	greyLine = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render
)

// Model holds chat state
type Model struct {
	agent       *agent.Agent
	queue       []openai.ChatCompletionMessageParamUnion
	textInput   textinput.Model
	width       int
	currentMode Mode
}

// New creates a new UI model
func New(a *agent.Agent) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message"
	ti.Prompt = "❯ "
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return Model{
		agent:       a,
		textInput:   ti,
		width:       50,
		currentMode: ModeNormal,
	}
}

// Init starts ticking
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles input, ticks, and resizing
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			input := strings.TrimSpace(m.textInput.Value())
			if input != "" {
				m.queue = append(m.queue, openai.UserMessage(input))
				m.textInput.SetValue("")
			}
		case "shift+tab":
			m.currentMode = (m.currentMode + 1) % 3
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.textInput.Width = m.width

	case openai.ChatCompletionMessageParamUnion:
		cmds = append(cmds, m.agent.Update(msg))
	}

	if !m.agent.Busy() && len(m.queue) > 0 {
		msg := m.queue[0]
		m.queue = m.queue[1:]
		cmds = append(cmds, m.agent.Update(msg))
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the chat UI
func (m Model) View() string {
	var builder strings.Builder
	width := m.width
	if width <= 0 {
		width = 50
	}

	builder.WriteString(m.agent.View(width))

	for _, msg := range m.queue {
		if msg.OfUser != nil {
			content := msg.OfUser.Content.OfString.String()
			builder.WriteString(fmt.Sprintf("%s\n\n", userStyle.Render(fmt.Sprintf("\u276f %s ", content))))
		}
	}

	builder.WriteString(greyLine(strings.Repeat("─", width)) + "\n")
	builder.WriteString(m.textInput.View() + "\n")
	builder.WriteString(greyLine(strings.Repeat("─", width)) + "\n")

	var modeStr string
	switch m.currentMode {
	case ModeNormal:
		modeStr = "? for shortcuts"
	case ModePlan:
		modeStr = cyanStyle.Render("⏸ plan mode on")
	case ModeAutoAccept:
		modeStr = purpleStyle.Render("⏵⏵ accept edits on")
	}
	builder.WriteString(modeStr)
	return builder.String()
}
