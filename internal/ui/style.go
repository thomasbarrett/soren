package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ParagraphStyle struct {
	initialPrefix    string
	subsequentPrefix string
	style            lipgloss.Style
	markdown         bool
	width            int
}

func NewParagraphStyle() ParagraphStyle {
	return ParagraphStyle{
		initialPrefix:    "",
		subsequentPrefix: "",
		style:            lipgloss.NewStyle(),
		width:            80,
	}
}

func (s ParagraphStyle) InitialPrefix(prefix string) ParagraphStyle {
	s.initialPrefix = prefix

	return s
}

func (s ParagraphStyle) SubsequentPrefix(prefix string) ParagraphStyle {
	s.subsequentPrefix = prefix

	return s
}

func (s ParagraphStyle) Width(width int) ParagraphStyle {
	s.width = width

	return s
}

func (s ParagraphStyle) Style(style lipgloss.Style) ParagraphStyle {
	s.style = style

	return s
}

func (s ParagraphStyle) Markdown() ParagraphStyle {
	s.markdown = true

	return s
}

func (s *ParagraphStyle) Render(text string) string {
	style := s.style.Width(s.width - len(s.subsequentPrefix))
	text = style.Render(text)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i == 0 {
			lines[i] = s.initialPrefix + line
		} else {
			lines[i] = s.subsequentPrefix + line
		}
	}

	return strings.Join(lines, "\n")
}
