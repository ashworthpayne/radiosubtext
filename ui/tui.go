package ui

import (
	"fmt"
	"strings"

	"radiosubtext/proto"
	"radiosubtext/radio"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	inbox     []proto.Message
	input     textinput.Model
	radio     *radio.Radio
	callSign  string
	group     string
	sendQueue chan proto.Message
}

func NewModel(r *radio.Radio, callSign string, group string) model {
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 40

	return model{
		inbox:     []proto.Message{},
		input:     ti,
		radio:     r,
		callSign:  callSign,
		group:     group,
		sendQueue: make(chan proto.Message, 10),
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			body := strings.TrimSpace(m.input.Value())
			if body != "" {
				m.sendQueue <- proto.Message{
					From:  m.callSign,
					Group: m.group,
					Cmd:   "MSG",
					Body:  body,
				}
				m.input.Reset()
			}
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var out strings.Builder
	out.WriteString("📻  D-STAR Chat Terminal\n")
	out.WriteString("----------------------------------\n")

	for _, msg := range m.inbox {
		line := fmt.Sprintf("[%s] %s: %s\n", msg.Group, msg.From, msg.Body)
		out.WriteString(line)
	}

	out.WriteString("\n> " + m.input.View())
	return out.String()
}

// External function to inject messages from radio
func (m *model) Push(msg proto.Message) {
	m.inbox = append(m.inbox, msg)
	if len(m.inbox) > 100 {
		m.inbox = m.inbox[len(m.inbox)-100:]
	}
}
