package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/thomasbarrett/soren/internal/agent"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the UI
var (
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	purpleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	greyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	blueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("167"))
	userStyle   = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "255", Dark: "255"}).
			Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "0"})
)

// turnKind identifies the kind of a committed conversation turn.
type turnKind int

const (
	turnUser turnKind = iota
	turnAssistant
	turnTools
)

// toolCallState tracks one in-flight or completed tool call.
type toolCallState struct {
	id     string
	name   string
	args   string
	result string
	done   bool
}

// chatTurn is a committed entry in the conversation display.
type chatTurn struct {
	kind      turnKind
	thinking  string // reasoning content that preceded this turn (may be empty)
	content   string
	toolCalls []toolCallState
}

// streamDone is returned by waitForEvent when the event channel closes.
type streamDone struct{}

// waitForEvent returns a tea.Cmd that reads the next event from ch.
func waitForEvent(ch <-chan agent.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return streamDone{}
		}
		return event
	}
}

// Model holds chat state
type Model struct {
	session     *agent.Session
	queue       []string
	eventCh     <-chan agent.StreamEvent
	turns       []chatTurn
	inFlight    []toolCallState
	thinking    bool
	thinkingBuf string // live thinking tokens for the current turn
	textBuf     string // live response tokens for the current turn
	errMsg      string // last error message to display
	textInput   textinput.Model
	width       int
}

// New creates a new UI model
func New(s *agent.Session) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message"
	ti.Prompt = "❯ "
	ti.Focus()
	ti.CharLimit = 256

	return Model{
		session:   s,
		textInput: ti,
		width:     50,
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
				m.queue = append(m.queue, input)
				m.textInput.SetValue("")
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.textInput.Width = m.width

	case agent.EventThinking:
		if len(m.inFlight) > 0 {
			m.turns = append(m.turns, chatTurn{kind: turnTools, thinking: m.thinkingBuf, toolCalls: m.inFlight})
			m.inFlight = nil
			m.thinkingBuf = ""
			m.textBuf = ""
		}
		m.thinking = true
		cmds = append(cmds, waitForEvent(m.eventCh))

	case agent.EventThinkingDelta:
		m.thinking = false
		m.thinkingBuf += msg.Token
		cmds = append(cmds, waitForEvent(m.eventCh))

	case agent.EventOutputDelta:
		m.thinking = false
		m.textBuf += msg.Token
		cmds = append(cmds, waitForEvent(m.eventCh))

	case agent.EventToolUse:
		m.thinking = false
		m.inFlight = append(m.inFlight, toolCallState{
			id:   msg.ID,
			name: msg.Name,
			args: msg.Arguments,
		})
		cmds = append(cmds, waitForEvent(m.eventCh))

	case agent.EventToolResult:
		for i := range m.inFlight {
			if m.inFlight[i].id == msg.CallID {
				m.inFlight[i].result = msg.Content
				m.inFlight[i].done = true
				break
			}
		}
		cmds = append(cmds, waitForEvent(m.eventCh))

	case agent.EventOutput:
		m.thinking = false
		if len(m.inFlight) > 0 {
			m.turns = append(m.turns, chatTurn{kind: turnTools, thinking: m.thinkingBuf, toolCalls: m.inFlight})
			m.inFlight = nil
		}
		m.turns = append(m.turns, chatTurn{kind: turnAssistant, thinking: m.thinkingBuf, content: msg.Content})
		m.thinkingBuf = ""
		m.textBuf = ""
		m.eventCh = nil

	case agent.EventError:
		m.thinking = false
		m.inFlight = nil
		m.thinkingBuf = ""
		m.textBuf = ""
		m.errMsg = msg.Err.Error()
		m.eventCh = nil

	case streamDone:
		// Goroutine exited without a terminal event (shouldn't happen).
		m.thinking = false
		m.eventCh = nil
	}

	// Start next queued message when no stream is active.
	if m.eventCh == nil && len(m.queue) > 0 {
		input := m.queue[0]
		m.queue = m.queue[1:]
		m.turns = append(m.turns, chatTurn{kind: turnUser, content: input})
		m.errMsg = ""
		m.eventCh = m.session.Stream(context.Background(), input)
		cmds = append(cmds, waitForEvent(m.eventCh))
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the chat UI
func (m Model) View() string {
	if m.width <= 0 {
		return ""
	}

	var b strings.Builder

	const pad = 2 // width of "⏺ " prefix

	thinkingStyle := NewParagraphStyle().
		Style(greyStyle).
		Width(m.width).
		InitialPrefix("\u2502 ").
		SubsequentPrefix("\u2502 ")

	responseStyle := NewParagraphStyle().
		Width(m.width).
		InitialPrefix(greyStyle.Render("\u23fa") + " ").
		SubsequentPrefix("  ")

	renderText := func(text string) string {
		// Render Markdown using Glamour
		margin := uint(0)
		style := styles.LightStyleConfig
		if lipgloss.DefaultRenderer().HasDarkBackground() {
			style = styles.DarkStyleConfig
		}
		style.Document.Margin = &margin

		renderer, _ := glamour.NewTermRenderer(
			glamour.WithStyles(style),
			glamour.WithWordWrap(m.width-pad),
		)

		rendered, err := renderer.Render(text)
		if err != nil {
			rendered = text // fallback to plain text
		}

		rendered = strings.TrimSpace(rendered)

		return responseStyle.Render(rendered) + "\n\n"
	}

	// Committed conversation turns.
	for _, t := range m.turns {
		switch t.kind {
		case turnUser:
			b.WriteString(fmt.Sprintf("%s\n\n", userStyle.Render(fmt.Sprintf("\u276f %s ", t.content))))
		case turnAssistant:
			if t.thinking != "" {
				b.WriteString(thinkingStyle.Render(strings.TrimSpace(t.thinking)) + "\n\n")
			}
			b.WriteString(renderText(t.content))
		case turnTools:
			if t.thinking != "" {
				b.WriteString(thinkingStyle.Render(strings.TrimSpace(t.thinking)) + "\n\n")
			}
			for _, tc := range t.toolCalls {
				b.WriteString(blueStyle.Render("\u23fa") + " " + tc.name + "(" + tc.args + ")\n")
				b.WriteString("  \u23bf  Done (" + tc.result + ")\n\n")
			}
		}
	}

	// Live thinking tokens for the current turn.
	if m.thinkingBuf != "" {
		b.WriteString(thinkingStyle.Render(strings.TrimSpace(m.thinkingBuf)) + "\n\n")
	}

	// In-flight tool calls for the current turn.
	for _, tc := range m.inFlight {
		b.WriteString(blueStyle.Render("\u23fa") + " " + tc.name + "(" + tc.args + ")\n")
		if tc.done {
			b.WriteString("  \u23bf  Done (" + tc.result + ")\n\n")
		} else {
			b.WriteString("  \u23bf  " + greyStyle.Render("\u23fa") + " Running...\n\n")
		}
	}

	// Live response tokens for the current turn.
	if m.textBuf != "" {
		b.WriteString(renderText(m.textBuf))
	}

	// Spinner shown only before the first token arrives.
	if m.thinking {
		b.WriteString(greyStyle.Render("\u23fa") + " Thinking...\n\n")
	}

	// Error message from the last stream.
	if m.errMsg != "" {
		b.WriteString(redStyle.Render("\u23fa") + " " + redStyle.Render(m.errMsg) + "\n\n")
	}

	for _, pending := range m.queue {
		b.WriteString(userStyle.Render("\u276f "+pending+" ") + "\n\n")
	}

	b.WriteString(greyStyle.Render(strings.Repeat("\u2500", m.width)) + "\n")
	b.WriteString(m.textInput.View() + "\n")
	b.WriteString(greyStyle.Render(strings.Repeat("\u2500", m.width)) + "\n")

	return b.String()
}
